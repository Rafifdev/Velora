//go:build windows

package main

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"velora/internal/sources"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

//go:embed internal/assets/fonts/Poppins-Regular.ttf
var poppinsRegularTTF []byte

//go:embed internal/assets/fonts/Poppins-SemiBold.ttf
var poppinsSemiBoldTTF []byte

//go:embed internal/assets/fonts/Poppins-Bold.ttf
var poppinsBoldTTF []byte

//go:embed internal/assets/fonts/Phosphor.ttf
var phosphorRegularTTF []byte

//go:embed internal/assets/fonts/Phosphor-Fill.ttf
var phosphorFillTTF []byte

type DynamicIslandQueueItem struct {
	ID         int64  `json:"id"`
	ExternalID string `json:"externalId"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	CoverURL   string `json:"coverUrl"`
	IsFavorite bool   `json:"isFavorite"`
}

type DynamicIslandSyncPayload struct {
	Title         string                   `json:"title"`
	Artist        string                   `json:"artist"`
	Album         string                   `json:"album"`
	CoverURL      string                   `json:"coverUrl"`
	IsPlaying     bool                     `json:"isPlaying"`
	IsFavorite    bool                     `json:"isFavorite"`
	PositionSec   float64                  `json:"positionSec"`
	DurationSec   float64                  `json:"durationSec"`
	Volume        int                      `json:"volume"`
	Muted         bool                     `json:"muted"`
	Shuffle       bool                     `json:"shuffle"`
	RepeatMode    string                   `json:"repeatMode"`
	EqPreset      string                   `json:"eqPreset"`
	ThemeSeed     string                   `json:"themeSeed"`
	AccentColor   string                   `json:"accentColor"`
	StreakDays     int                      `json:"streakDays"`
	LyricsLoading  bool                     `json:"lyricsLoading"`
	TrackLoading   bool                     `json:"trackLoading"`
	LyricOffsetSec float64                  `json:"lyricOffsetSec"`
	QueueTitle     string                   `json:"queueTitle"`
	QueueIndex    int                      `json:"queueIndex"`
	Queue         []DynamicIslandQueueItem `json:"queue"`
}

const (
	diLogicalW = 640
	diLogicalH = 280

	diModeResting    = "pet"
	diModeClosed     = "idle"
	diModeMusic      = "music"
	diModeLyrics     = "lyrics"
	diModeHeadphones     = "bt"
	diModeQueue          = "queue"
	diModeFullscreenHide = "fs_hide"

	diWsPopup        = 0x80000000
	diWsExTopmost    = 0x00000008
	diWsExToolWindow = 0x00000080
	diWsExLayered    = 0x00080000
	diWsExNoActivate = 0x08000000

	diSwShowNoActivate = 4
	diSwHide           = 0

	diWmDestroy     = 0x0002
	diWmTimer       = 0x0113
	diWmMouseMove   = 0x0200
	diWmLButtonDown = 0x0201
	diWmLButtonUp   = 0x0202
	diWmMouseWheel  = 0x020A
	diWmMouseLeave  = 0x02A3
	diWmMouseHWheel = 0x020E
	diWmSetCursor   = 0x0020
	diWmAppSync    = 0x8000 + 201
	diWmAppClose   = 0x8000 + 202

	diUlwAlpha   = 0x00000002
	diAcSrcOver  = 0x00
	diAcSrcAlpha = 0x01

	diTmeLeave = 0x00000002
	diTimerID  = 1

	diHitNone    = 0
	diHitBody    = 1
	diHitArt     = 2
	diHitBubble  = 3
	diHitPrev    = 4
	diHitPlay    = 5
	diHitNext    = 6
	diHitArrow   = 7
	diHitSeek    = 8
	diHitLyric   = 9
	diHitDot0    = 10
	diHitDot1    = 11
	diHitDot2    = 12
	diHitDot3    = 13
	diHitDot4    = 14
	diHitFav        = 15
	diHitShuffle    = 16
	diHitRepeat     = 17
	diHitMute       = 18
	diHitEqCycle    = 19
	diHitQueueOpen  = 20
	diHitQueueBack  = 21
	diHitQueueRow0  = 22
	diHitQueueRow1  = 23
	diHitQueueRow2  = 24
	diHitQueueFav0  = 25
	diHitQueueFav1  = 26
	diHitQueueFav2  = 27
	diHitQueueMore0 = 28
	diHitQueueMore1 = 29
	diHitQueueMore2 = 30
	diHitWifi       = 31
	diHitGps        = 32
	diHitBt         = 33
	diHitQueueRow   = 34
	diHitQueueFav    = 35
	diHitQueueScroll = 36
	diHitEdgeLeft    = 37
	diHitEdgeRight   = 38
	diHitAudioDev0   = 39
	diHitAudioDev1   = 40
	diHitAudioDev2   = 41
	diHitHpToggle    = 42
)

var (
	diGdiplus  = windows.NewLazySystemDLL("gdiplus.dll")
	diShcore   = windows.NewLazySystemDLL("shcore.dll")
	diWinmm    = windows.NewLazySystemDLL("winmm.dll")
	diSetupapi = windows.NewLazySystemDLL("setupapi.dll")
	diWlanapi  = windows.NewLazySystemDLL("wlanapi.dll")
	diBthApis  = windows.NewLazySystemDLL("BluetoothAPIs.dll")
	diKernel32 = windows.NewLazySystemDLL("kernel32.dll")
	diShell32  = windows.NewLazySystemDLL("shell32.dll")

	diProcGetSystemPowerStatus           = diKernel32.NewProc("GetSystemPowerStatus")
	diProcSHQueryUserNotificationState   = diShell32.NewProc("SHQueryUserNotificationState")

	diProcRegisterClassExW             = user32.NewProc("RegisterClassExW")
	diProcCreateWindowExW              = user32.NewProc("CreateWindowExW")
	diProcDestroyWindow                = user32.NewProc("DestroyWindow")
	diProcDefWindowProcW               = user32.NewProc("DefWindowProcW")
	diProcGetMessageW                  = user32.NewProc("GetMessageW")
	diProcTranslateMessage             = user32.NewProc("TranslateMessage")
	diProcDispatchMessageW             = user32.NewProc("DispatchMessageW")
	diProcPostQuitMessage              = user32.NewProc("PostQuitMessage")
	diProcShowWindow                   = user32.NewProc("ShowWindow")
	diProcSetWindowPos                 = user32.NewProc("SetWindowPos")
	diProcUpdateLayeredWindow          = user32.NewProc("UpdateLayeredWindow")
	diProcGetSystemMetrics             = user32.NewProc("GetSystemMetrics")
	diProcSetTimer                     = user32.NewProc("SetTimer")
	diProcKillTimer                    = user32.NewProc("KillTimer")
	diProcTrackMouseEvent              = user32.NewProc("TrackMouseEvent")
	diProcLoadCursorW                  = user32.NewProc("LoadCursorW")
	diProcSetCursor                    = user32.NewProc("SetCursor")
	diProcGetCursorPos                 = user32.NewProc("GetCursorPos")
	diProcSetCapture                   = user32.NewProc("SetCapture")
	diProcReleaseCapture               = user32.NewProc("ReleaseCapture")
	diProcGetDC                        = user32.NewProc("GetDC")
	diProcReleaseDC                    = user32.NewProc("ReleaseDC")
	diProcIsIconic                     = user32.NewProc("IsIconic")
	diProcIsWindowVisible              = user32.NewProc("IsWindowVisible")
	diProcGetForegroundWindow          = user32.NewProc("GetForegroundWindow")
	diProcGetWindowRect                = user32.NewProc("GetWindowRect")
	diProcGetWindowLongW               = user32.NewProc("GetWindowLongW")
	diProcGetClassNameW                = user32.NewProc("GetClassNameW")
	diProcMonitorFromWindow            = user32.NewProc("MonitorFromWindow")
	diProcGetMonitorInfoW              = user32.NewProc("GetMonitorInfoW")
	diProcSetThreadDpiAwarenessContext = user32.NewProc("SetThreadDpiAwarenessContext")
	diProcGetDpiForSystem              = user32.NewProc("GetDpiForSystem")
	diProcMonitorFromPoint             = user32.NewProc("MonitorFromPoint")
	diProcGetDpiForMonitor             = diShcore.NewProc("GetDpiForMonitor")
	diProcGetModuleHandleW             = kernel32.NewProc("GetModuleHandleW")
	diProcGetTimeFormatEx              = kernel32.NewProc("GetTimeFormatEx")

	diProcCreateCompatibleDC   = gdi32.NewProc("CreateCompatibleDC")
	diProcDeleteDC             = gdi32.NewProc("DeleteDC")
	diProcSelectObject         = gdi32.NewProc("SelectObject")
	diProcAddFontMemResourceEx = gdi32.NewProc("AddFontMemResourceEx")

	diProcWaveOutGetNumDevs      = diWinmm.NewProc("waveOutGetNumDevs")
	diProcWaveOutGetDevCapsW     = diWinmm.NewProc("waveOutGetDevCapsW")
	diProcSetupDiGetClassDevsW   = diSetupapi.NewProc("SetupDiGetClassDevsW")
	diProcSetupDiEnumDeviceInfo  = diSetupapi.NewProc("SetupDiEnumDeviceInfo")
	diProcSetupDiGetDevPropertyW = diSetupapi.NewProc("SetupDiGetDevicePropertyW")
	diProcSetupDiDestroyDevList  = diSetupapi.NewProc("SetupDiDestroyDeviceInfoList")

	diProcWlanOpenHandle       = diWlanapi.NewProc("WlanOpenHandle")
	diProcWlanEnumInterfaces   = diWlanapi.NewProc("WlanEnumInterfaces")
	diProcWlanFreeMemory       = diWlanapi.NewProc("WlanFreeMemory")
	diProcWlanCloseHandle      = diWlanapi.NewProc("WlanCloseHandle")
	diProcBthFindFirstRadio    = diBthApis.NewProc("BluetoothFindFirstRadio")
	diProcBthFindRadioClose    = diBthApis.NewProc("BluetoothFindRadioClose")
	diProcBthIsConnectable     = diBthApis.NewProc("BluetoothIsConnectable")

	diProcGdiplusStartup             = diGdiplus.NewProc("GdiplusStartup")
	diProcGdiplusShutdown            = diGdiplus.NewProc("GdiplusShutdown")
	diProcGdipCreateFromHDC          = diGdiplus.NewProc("GdipCreateFromHDC")
	diProcGdipDeleteGraphics         = diGdiplus.NewProc("GdipDeleteGraphics")
	diProcGdipSetSmoothingMode       = diGdiplus.NewProc("GdipSetSmoothingMode")
	diProcGdipSetPixelOffsetMode     = diGdiplus.NewProc("GdipSetPixelOffsetMode")
	diProcGdipSetTextRendering       = diGdiplus.NewProc("GdipSetTextRenderingHint")
	diProcGdipCreateSolidFill        = diGdiplus.NewProc("GdipCreateSolidFill")
	diProcGdipDeleteBrush            = diGdiplus.NewProc("GdipDeleteBrush")
	diProcGdipCreatePen1             = diGdiplus.NewProc("GdipCreatePen1")
	diProcGdipSetPenStartCap         = diGdiplus.NewProc("GdipSetPenStartCap")
	diProcGdipSetPenEndCap           = diGdiplus.NewProc("GdipSetPenEndCap")
	diProcGdipSetPenLineJoin         = diGdiplus.NewProc("GdipSetPenLineJoin")
	diProcGdipDeletePen              = diGdiplus.NewProc("GdipDeletePen")
	diProcGdipCreatePath             = diGdiplus.NewProc("GdipCreatePath")
	diProcGdipDeletePath             = diGdiplus.NewProc("GdipDeletePath")
	diProcGdipStartPathFigure        = diGdiplus.NewProc("GdipStartPathFigure")
	diProcGdipClosePathFigure        = diGdiplus.NewProc("GdipClosePathFigure")
	diProcGdipAddPathLine            = diGdiplus.NewProc("GdipAddPathLine")
	diProcGdipAddPathArc               = diGdiplus.NewProc("GdipAddPathArc")
	diProcGdipAddPathBezier            = diGdiplus.NewProc("GdipAddPathBezier")
	diProcGdipAddPathString            = diGdiplus.NewProc("GdipAddPathString")
	diProcGdipGetPathWorldBounds       = diGdiplus.NewProc("GdipGetPathWorldBounds")
	diProcGdipCreateMatrix2            = diGdiplus.NewProc("GdipCreateMatrix2")
	diProcGdipDeleteMatrix             = diGdiplus.NewProc("GdipDeleteMatrix")
	diProcGdipTransformPath            = diGdiplus.NewProc("GdipTransformPath")
	diProcGdipFillPath                 = diGdiplus.NewProc("GdipFillPath")
	diProcGdipDrawPath               = diGdiplus.NewProc("GdipDrawPath")
	diProcGdipFillRectangle          = diGdiplus.NewProc("GdipFillRectangle")
	diProcGdipFillEllipse            = diGdiplus.NewProc("GdipFillEllipse")
	diProcGdipDrawEllipse            = diGdiplus.NewProc("GdipDrawEllipse")
	diProcGdipDrawArc                = diGdiplus.NewProc("GdipDrawArc")
	diProcGdipDrawLine               = diGdiplus.NewProc("GdipDrawLine")
	diProcGdipNewPrivateFontColl     = diGdiplus.NewProc("GdipNewPrivateFontCollection")
	diProcGdipDeletePrivateFontColl  = diGdiplus.NewProc("GdipDeletePrivateFontCollection")
	diProcGdipPrivateAddMemoryFont   = diGdiplus.NewProc("GdipPrivateAddMemoryFont")
	diProcGdipCreateFontFamily       = diGdiplus.NewProc("GdipCreateFontFamilyFromName")
	diProcGdipDeleteFontFamily       = diGdiplus.NewProc("GdipDeleteFontFamily")
	diProcGdipCreateFont             = diGdiplus.NewProc("GdipCreateFont")
	diProcGdipDeleteFont             = diGdiplus.NewProc("GdipDeleteFont")
	diProcGdipCreateStringFormat     = diGdiplus.NewProc("GdipCreateStringFormat")
	diProcGdipSetStringFormatTrim    = diGdiplus.NewProc("GdipSetStringFormatTrimming")
	diProcGdipSetStringFormatFlags   = diGdiplus.NewProc("GdipSetStringFormatFlags")
	diProcGdipSetStringFormatAlign     = diGdiplus.NewProc("GdipSetStringFormatAlign")
	diProcGdipSetStringFormatLineAlign = diGdiplus.NewProc("GdipSetStringFormatLineAlign")
	diProcGdipMeasureString          = diGdiplus.NewProc("GdipMeasureString")
	diProcGdipDeleteStringFormat     = diGdiplus.NewProc("GdipDeleteStringFormat")
	diProcGdipDrawString             = diGdiplus.NewProc("GdipDrawString")
	diProcGdipSetClipRect            = diGdiplus.NewProc("GdipSetClipRect")
	diProcGdipResetClip              = diGdiplus.NewProc("GdipResetClip")
)

type diWndClassExW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     windows.Handle
	HIcon         windows.Handle
	HCursor       windows.Handle
	HbrBackground windows.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       windows.Handle
}

type diPoint struct {
	X int32
	Y int32
}

type diSize struct {
	Cx int32
	Cy int32
}

type diBlendFunction struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

type diTrackMouseEventStruct struct {
	CbSize      uint32
	DwFlags     uint32
	HwndTrack   uintptr
	DwHoverTime uint32
}

type diRectF struct {
	X      float32
	Y      float32
	Width  float32
	Height float32
}

type diRect struct {
	Left, Top, Right, Bottom int32
}

type diMonitorInfo struct {
	CbSize    uint32
	RcMonitor diRect
	RcWork    diRect
	DwFlags   uint32
}

type diGdiplusStartupInput struct {
	GdiplusVersion           uint32
	DebugEventCallback       uintptr
	SuppressBackgroundThread int32
	SuppressExternalCodecs   int32
}

type diMsg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      diPoint
}

type diWaveOutCapsW struct {
	WMid           uint16
	WPid           uint16
	VDriverVersion uint32
	SzPname        [32]uint16
	DwFormats      uint32
	WChannels      uint16
	WReserved1     uint16
	DwSupport      uint32
}

type diGuid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

type diDevPropKey struct {
	Fmtid diGuid
	Pid   uint32
}

type diSpDevInfoData struct {
	CbSize    uint32
	ClassGuid diGuid
	DevInst   uint32
	Reserved  uintptr
}

type veloraBrandPalette struct {
	BgDeep       color.RGBA // #090A0C
	BgSurface    color.RGBA // #101113
	Primary      color.RGBA // #B8EAFF (Titanium Ice Cyan)
	PrimaryDim   color.RGBA // #004D64 (Deep Steel Cyan)
	Accent       color.RGBA // #FF9E64 (Sunset Tangerine)
	CoverDom     color.RGBA // Dynamic dominant color from album cover
	HasCoverDom  bool
	TextMain     color.RGBA // #F1F3F5
	TextVariant  color.RGBA // #C1C7CE
	TextOutline  color.RGBA // #8B9198
}

type dynamicIslandController struct {
	ctx          context.Context
	app          *App
	hwnd         uintptr
	cursorArrow  uintptr
	cursorHand   uintptr
	mu           sync.Mutex
	enabled      bool
	winShown     bool
	wasMinimized bool
	hovered      bool
	hoverZone    int
	hoverRatio   float64
	hoverLyric   int
	tracking     bool
	dragActive   bool
	dragStartX   int
	dragStartY   int
	lastHWheelAt time.Time

	dpiScale float64
	physW    int
	physH    int

	fontColl        uintptr
	famRegular      uintptr // "Poppins" (Regular 0, Bold 1)
	famSemiBold     uintptr // "Poppins SemiBold" (Regular 0)
	famPhosphorReg  uintptr // "Phosphor"
	famPhosphorFill uintptr // "Phosphor-Fill"

	d2dFactory      uintptr
	d2dDCRT         uintptr
	d2dBrush        uintptr
	dwFactory       uintptr
	dwMemLoader     uintptr
	dwFontColl      uintptr
	d2dFmtCache     map[d2dFmtKey]uintptr
	d2dAlphaScratch []byte
	lang            string

	lyricsOpen           bool
	queueOpen            bool
	headphonesOpen       bool
	queueScroll          int
	queueScrollDp        float64
	queueTargetScrollDp  float64
	queueDragScroll      bool
	queueDragStartScroll float64
	seekDragging         bool
	hoverQueueIdx        int
	pinnedMode           string
	pinnedUntil time.Time
	activeMode  string
	transStart  time.Time
	shrinking   bool
	fromW, toW  float64
	fromH, toH  float64
	fromR, toR  float64
	fromE, toE  float64
	curW, curH  float64
	curR, curE  float64

	appearProg       float64
	startClock       time.Time
	marqueeStart     time.Time
	lyricIdx         int
	prevLyricIdx     int
	lyricAnimProg    float64
	lyricFromY       float64
	lyricToY         float64
	lyricCurY        float64
	lyricStart       time.Time
	closedLyricFrom  string
	closedLyricTo    string
	closedLyricStart time.Time
	closedLyricProg  float64
	closedWFrom          float64
	closedWTo            float64
	closedWStart         time.Time
	lyricManualOffsetDp  float64
	lyricTargetOffsetDp  float64
	lyricManualUntil     time.Time
	lyricDragScroll      bool
	lyricDragStartOffset float64
	lastMouseRx          float64
	edgeLeftProg         float64
	edgeRightProg        float64
	pageSlideDir         int
	cursorMove           uintptr
	winX                 int
	winXFloat            float64
	winTargetX           float64
	winVelX              float64
	lastTickAt           time.Time
	curTimerMs           uint32
	hwPolling            bool
	liquidSkewDp         float64
	screenW              int
	screenLeft           int
	screenTop            int
	screenMode           string // "primary" | "cursor"
	animSpeedScale       float64
	openTrigger          string // "hover" | "click"
	onlyPlaying          bool
	pauseHideSec         int
	pausedAt             time.Time
	wasPlaying           bool
	holdPending          bool
	holdStart            time.Time
	holdStartScreenX     int32
	holdStartScreenY     int32
	holdStartWinX        int
	winDragActive        bool
	dockSide             int // -1 = docked left, 0 = free/center, +1 = docked right
	dockLeftProg         float64
	dockRightProg        float64
	pillMode             bool
	winY                 int
	compactStyle         string
	sizeScale            float64
	opacityPct           int
	showStatus           bool
	showCover            bool
	coverStyle           string
	coverSpin            bool
	showClosedCover      bool
	closedCoverSpin      bool
	coverSpinRad         float64
	closedSpinRad        float64
	showArtist           bool
	showLyric            bool
	lyricSweep           bool
	showHeart            bool
	showDots             bool
	showProgress         bool
	showClosedProgress   bool
	progressStyle        string
	hideWhenAppActive    bool
	hideInFullscreen     bool
	fullscreenActive     bool
	idleHideSec          int
	lastActivityAt       time.Time
	hideLineProg         float64

	title         string
	artist        string
	album         string
	coverURL      string
	isPlaying     bool
	isFavorite    bool
	posSec         float64
	posUpdatedAt   time.Time
	durSec         float64
	lyricOffsetSec float64
	volume         int
	muted          bool
	shuffle        bool
	repeatMode     string
	eqPreset       string
	themeSeed      string
	accentHex      string
	streakDays     int
	lyricsLoading  bool
	lyricsHasPlain bool
	lyricsSource   string
	trackLoading   bool

	audioDeviceName string
	audioConnType   string // "bluetooth", "wired", "speaker"
	btBatteryPct    int    // -1 if wired/speakers, 0..100 if Bluetooth hands-free battery reported
	sysBatteryPct   int    // -1 if desktop/unknown, 0..100 if laptop/PC battery reported
	isBtAudio       bool
	audioOutputs    []windowsAudioEndpoint
	hpListOpen      bool
	hpListProg      float64
	wifiOn          bool
	gpsOn           bool
	btOn            bool
	lastHwPoll      time.Time

	rawCover       image.Image
	coverImg       *image.RGBA
	coverImgCircle *image.RGBA
	coverImgMini   *image.RGBA
	coverLoading   bool
	lastCoverRetry time.Time
	defaultArt     *image.RGBA
	defaultArtCirc *image.RGBA
	defaultArtSm   *image.RGBA
	defaultArtMini *image.RGBA
	coverAccent    color.RGBA
	hasCoverAcc    bool
	lyrics         []sources.LyricLine
	queueTitle     string
	queueIndex     int
	queueItems     []DynamicIslandQueueItem
	queueArtCache  map[string]*image.RGBA
	queueArtPend   map[string]bool

	gdiToken uintptr
	memDC    uintptr
	dibBmp   uintptr
	oldBmp   uintptr
	dibBits  unsafe.Pointer
}

var (
	globalIslandMu sync.Mutex
	globalIsland   *dynamicIslandController
)

func nusaModeDims(mode string) (w, h, r, e float64) {
	switch mode {
	case diModeMusic:
		return 420, 180, 30, 20
	case diModeLyrics:
		return 520, 180, 30, 20
	case diModeQueue:
		return 420, 272, 30, 20
	case diModeHeadphones:
		return 420, 106, 24, 16
	case diModeFullscreenHide:
		return 132, 4, 2.0, 0
	case diModeClosed:
		return 216, 36, 18, 12
	default: // diModeResting
		return 192, 36, 18, 12
	}
}

func evalCubicBezier(x1, y1, x2, y2, t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	cx := 3.0 * x1
	bx := 3.0*(x2-x1) - cx
	ax := 1.0 - cx - bx

	cy := 3.0 * y1
	by := 3.0*(y2-y1) - cy
	ay := 1.0 - cy - by

	sampleX := func(u float64) float64 { return ((ax*u+bx)*u + cx) * u }
	sampleDerivX := func(u float64) float64 { return (3.0*ax*u+2.0*bx)*u + cx }
	sampleY := func(u float64) float64 { return ((ay*u+by)*u + cy) * u }

	u := t
	for i := 0; i < 8; i++ {
		xErr := sampleX(u) - t
		if math.Abs(xErr) < 1e-6 {
			break
		}
		d := sampleDerivX(u)
		if math.Abs(d) < 1e-6 {
			break
		}
		u -= xErr / d
	}
	if u < 0 {
		u = 0
	} else if u > 1 {
		u = 1
	}
	return sampleY(u)
}

func nusaSpring(t float64) float64 {
	return evalCubicBezier(0.34, 1.32, 0.5, 1.0, t)
}

func nusaShrinkEase(t float64) float64 {
	return evalCubicBezier(0.22, 1.0, 0.36, 1.0, t)
}

func nusaCssEase(t float64) float64 {
	return evalCubicBezier(0.25, 0.1, 0.25, 1.0, t)
}

func nusaContentPop(t float64) float64 {
	return evalCubicBezier(0.2, 0.85, 0.25, 1.0, t)
}

func nusaArcEase(t float64) float64 {
	return evalCubicBezier(0.2, 0.8, 0.2, 1.0, t)
}

func (a *App) initDynamicIsland(ctx context.Context) {
	globalIslandMu.Lock()
	if globalIsland != nil {
		globalIslandMu.Unlock()
		return
	}
	enabled := true
	hideWhenAppActive := true
	hideInFullscreen := true
	idleHideSec := 0
	screenMode := "primary"
	animSpeedScale := 1.0
	openTrigger := "hover"
	onlyPlaying := false
	pauseHideSec := 0
	showStatus := true
	showCover := true
	coverStyle := "square"
	coverSpin := true
	showClosedCover := true
	closedCoverSpin := false
	showArtist := true
	showLyric := true
	lyricSweep := true
	showHeart := true
	showDots := true
	showProgress := true
	showClosedProgress := true
	progressStyle := "thin"
	themeSeed := "monolith"
	accentHex := "#F6847E"
	eqPreset := "spotify"
	lang := "id"
	if a.store != nil {
		if prefs, err := a.store.GetPreferences(); err == nil {
			if v := strings.ToLower(strings.TrimSpace(prefs["content.language"])); v != "" {
				lang = v
			}
			if v, ok := prefs["ui.dynamicIsland"]; ok && v == "false" {
				enabled = false
			}
			if v, ok := prefs["dynamicIsland.hideApp"]; ok && v == "false" {
				hideWhenAppActive = false
			}
			if v, ok := prefs["dynamicIsland.hideFull"]; ok && v == "false" {
				hideInFullscreen = false
			}
			if v := strings.TrimSpace(prefs["dynamicIsland.idleHideSec"]); v != "" {
				if sec, convErr := strconv.Atoi(v); convErr == nil && sec > 0 {
					idleHideSec = sec
				}
			}
			if strings.ToLower(strings.TrimSpace(prefs["dynamicIsland.screen"])) == "cursor" {
				screenMode = "cursor"
			}
			switch strings.ToLower(strings.TrimSpace(prefs["dynamicIsland.animSpeed"])) {
			case "slow":
				animSpeedScale = 1.45
			case "fast":
				animSpeedScale = 0.62
			}
			if strings.ToLower(strings.TrimSpace(prefs["dynamicIsland.openTrigger"])) == "click" {
				openTrigger = "click"
			}
			onlyPlaying = strings.ToLower(strings.TrimSpace(prefs["dynamicIsland.onlyPlaying"])) == "true"
			if v := strings.TrimSpace(prefs["dynamicIsland.pauseHide"]); v != "" {
				if sec, convErr := strconv.Atoi(v); convErr == nil && sec > 0 {
					pauseHideSec = sec
				}
			}
			if v, ok := prefs["dynamicIsland.showStatus"]; ok && v == "false" {
				showStatus = false
			}
			if v, ok := prefs["dynamicIsland.showCover"]; ok && v == "false" {
				showCover = false
			}
			if v := strings.ToLower(strings.TrimSpace(prefs["dynamicIsland.coverStyle"])); v == "circle" || v == "square" {
				coverStyle = v
			}
			if v, ok := prefs["dynamicIsland.coverSpin"]; ok && v == "false" {
				coverSpin = false
			}
			if v, ok := prefs["dynamicIsland.showClosedCover"]; ok && v == "false" {
				showClosedCover = false
			}
			if v, ok := prefs["dynamicIsland.closedCoverSpin"]; ok && v == "true" {
				closedCoverSpin = true
			}
			if v, ok := prefs["dynamicIsland.showArtist"]; ok && v == "false" {
				showArtist = false
			}
			if v, ok := prefs["dynamicIsland.showLyric"]; ok && v == "false" {
				showLyric = false
			}
			if v, ok := prefs["dynamicIsland.lyricSweep"]; ok && v == "false" {
				lyricSweep = false
			}
			if v, ok := prefs["dynamicIsland.showHeart"]; ok && v == "false" {
				showHeart = false
			}
			if v, ok := prefs["dynamicIsland.showDots"]; ok && v == "false" {
				showDots = false
			}
			if v, ok := prefs["dynamicIsland.showProgress"]; ok && v == "false" {
				showProgress = false
			}
			if v, ok := prefs["dynamicIsland.showClosedProgress"]; ok && v == "false" {
				showClosedProgress = false
			}
			if v := strings.ToLower(strings.TrimSpace(prefs["dynamicIsland.progressStyle"])); v == "thick" || v == "dots" || v == "thin" {
				progressStyle = v
			}
			if v := strings.TrimSpace(prefs["theme.seed"]); v != "" {
				themeSeed = v
			}
			if v := strings.TrimSpace(prefs["accent_color"]); v != "" {
				accentHex = v
			}
			if v := strings.TrimSpace(prefs["playback.eqPreset"]); v != "" {
				eqPreset = v
			}
		}
	}
	streakDays := 0
	if a.statsSvc != nil {
		if st, err := a.statsSvc.GetFireStreak(time.Now()); err == nil {
			streakDays = st.CurrentStreak
		}
	}

	now := time.Now()
	ctrl := &dynamicIslandController{
		ctx:             ctx,
		app:             a,
		enabled:         enabled,
		dpiScale:        1.25,
		activeMode:      diModeResting,
		transStart:      now,
		startClock:      now,
		fromW:           166,
		toW:             166,
		curW:            166,
		fromH:           30,
		toH:             30,
		curH:            30,
		fromR:           14,
		toR:             14,
		curR:            14,
		fromE:           10,
		toE:             10,
		curE:            10,
		volume:            80,
		opacityPct:        100,
		hideWhenAppActive: hideWhenAppActive,
		hideInFullscreen:  hideInFullscreen,
		idleHideSec:       idleHideSec,
		screenMode:        screenMode,
		animSpeedScale:    animSpeedScale,
		openTrigger:       openTrigger,
		onlyPlaying:       onlyPlaying,
		pauseHideSec:      pauseHideSec,
		showStatus:        showStatus,
		showCover:         showCover,
		coverStyle:        coverStyle,
		coverSpin:         coverSpin,
		showClosedCover:   showClosedCover,
		closedCoverSpin:   closedCoverSpin,
		showArtist:        showArtist,
		showLyric:         showLyric,
		lyricSweep:        lyricSweep,
		showHeart:         showHeart,
		showDots:           showDots,
		showProgress:       showProgress,
		showClosedProgress: showClosedProgress,
		progressStyle:     progressStyle,
		lastActivityAt:    now,
		repeatMode:        "off",
		eqPreset:        eqPreset,
		themeSeed:       themeSeed,
		accentHex:       accentHex,
		lang:            lang,
		streakDays:      streakDays,
		audioDeviceName: "Velora Audio Output",
		btBatteryPct:    -1,
		queueArtCache:   make(map[string]*image.RGBA),
		queueArtPend:    make(map[string]bool),
	}
	ctrl.pollAudioHardware(false)
	globalIsland = ctrl
	globalIslandMu.Unlock()

	go ctrl.runWindowLoop()
}

func (a *App) shutdownDynamicIsland() {
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIsland = nil
	globalIslandMu.Unlock()
	if ctrl != nil && ctrl.hwnd != 0 {
		_, _, _ = procPostMessageW.Call(ctrl.hwnd, diWmAppClose, 0, 0)
	}
}

func (a *App) SetDynamicIslandEnabled(enabled bool) {
	if a.store != nil {
		val := "false"
		if enabled {
			val = "true"
		}
		_ = a.store.SetPreference("ui.dynamicIsland", val)
	}
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}
	ctrl.mu.Lock()
	ctrl.enabled = enabled
	ctrl.mu.Unlock()
	if ctrl.hwnd != 0 {
		_, _, _ = procPostMessageW.Call(ctrl.hwnd, diWmAppSync, 0, 0)
	}
}

func (a *App) SetDynamicIslandLanguage(lang string) {
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}
	ctrl.mu.Lock()
	ctrl.lang = strings.ToLower(strings.TrimSpace(lang))
	ctrl.mu.Unlock()
	if ctrl.hwnd != 0 {
		_, _, _ = procPostMessageW.Call(ctrl.hwnd, diWmAppSync, 0, 0)
	}
}

type diLocalizedStrings struct {
	Connected     string
	Available     string
	NoLyrics      string
	NoSyncLyrics  string
	LoadingLyrics string
	EmptyQueue    string
	QueueTitle    string
}

func getDIStrings(lang string) diLocalizedStrings {
	switch strings.ToLower(strings.TrimSpace(lang)) {
	case "id":
		return diLocalizedStrings{
			Connected:     "Terhubung",
			Available:     "Tersedia",
			NoLyrics:      "Lirik tidak ditemukan",
			NoSyncLyrics:  "Lirik tersinkron tidak tersedia",
			LoadingLyrics: "Memuat lirik…",
			EmptyQueue:    "Antrean lagu kosong",
			QueueTitle:    "Antrean",
		}
	case "ms":
		return diLocalizedStrings{
			Connected:     "Disambung",
			Available:     "Tersedia",
			NoLyrics:      "Lirik tidak ditemui",
			NoSyncLyrics:  "Lirik segerak tidak tersedia",
			LoadingLyrics: "Memuatkan lirik…",
			EmptyQueue:    "Baris gilir kosong",
			QueueTitle:    "Baris Gilir",
		}
	case "ja":
		return diLocalizedStrings{
			Connected:     "接続済み",
			Available:     "利用可能",
			NoLyrics:      "歌詞が見つかりません",
			NoSyncLyrics:  "同期歌詞は利用できません",
			LoadingLyrics: "歌詞を読み込み中…",
			EmptyQueue:    "再生キューは空です",
			QueueTitle:    "キュー",
		}
	case "ko":
		return diLocalizedStrings{
			Connected:     "연결됨",
			Available:     "사용 가능",
			NoLyrics:      "가사를 찾을 수 없습니다",
			NoSyncLyrics:  "싱크 가사를 사용할 수 없습니다",
			LoadingLyrics: "가사 불러오는 중…",
			EmptyQueue:    "대기열이 비어 있습니다",
			QueueTitle:    "대기열",
		}
	case "hi":
		return diLocalizedStrings{
			Connected:     "कनेक्टेड",
			Available:     "उपलब्ध",
			NoLyrics:      "बोल नहीं मिले",
			NoSyncLyrics:  "सिंक किए गए बोल उपलब्ध नहीं हैं",
			LoadingLyrics: "बोल लोड हो रहे हैं…",
			EmptyQueue:    "कतार खाली है",
			QueueTitle:    "कतार",
		}
	case "pt":
		return diLocalizedStrings{
			Connected:     "Conectado",
			Available:     "Disponível",
			NoLyrics:      "Letra não encontrada",
			NoSyncLyrics:  "Letra sincronizada indisponível",
			LoadingLyrics: "Carregando letra…",
			EmptyQueue:    "Fila vazia",
			QueueTitle:    "Fila",
		}
	case "de":
		return diLocalizedStrings{
			Connected:     "Verbunden",
			Available:     "Verfügbar",
			NoLyrics:      "Keine Songtexte gefunden",
			NoSyncLyrics:  "Synchronisierte Texte nicht verfügbar",
			LoadingLyrics: "Songtext wird geladen…",
			EmptyQueue:    "Warteschlange ist leer",
			QueueTitle:    "Warteschlange",
		}
	case "fr":
		return diLocalizedStrings{
			Connected:     "Connecté",
			Available:     "Disponible",
			NoLyrics:      "Paroles introuvables",
			NoSyncLyrics:  "Paroles synchronisées indisponibles",
			LoadingLyrics: "Chargement des paroles…",
			EmptyQueue:    "File d'attente vide",
			QueueTitle:    "File d'attente",
		}
	case "es":
		return diLocalizedStrings{
			Connected:     "Conectado",
			Available:     "Disponible",
			NoLyrics:      "Letra no encontrada",
			NoSyncLyrics:  "Letra sincronizada no disponible",
			LoadingLyrics: "Cargando letra…",
			EmptyQueue:    "La cola está vacía",
			QueueTitle:    "Cola",
		}
	default: // "en"
		return diLocalizedStrings{
			Connected:     "Connected",
			Available:     "Available",
			NoLyrics:      "No lyrics found",
			NoSyncLyrics:  "Synced lyrics unavailable",
			LoadingLyrics: "Loading lyrics…",
			EmptyQueue:    "Queue is empty",
			QueueTitle:    "Queue",
		}
	}
}

func (a *App) SetDynamicIslandShapeSize(compactStyle, size string, opacity int) {
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}
	cs := strings.ToUpper(strings.TrimSpace(compactStyle))
	if cs != "SATU_BARIS" && cs != "TERSEMBUNYI" {
		cs = "RINGKAS"
	}
	scale := 1.0
	switch strings.ToLower(strings.TrimSpace(size)) {
	case "small":
		scale = 0.88
	case "large":
		scale = 1.08
	}
	if opacity < 0 {
		opacity = 0
	} else if opacity > 100 {
		opacity = 100
	}
	ctrl.mu.Lock()
	wasCentered := ctrl.dockSide == 0 && math.Abs(ctrl.winTargetX-float64(ctrl.centeredWinXLocked(ctrl.screenLeft, ctrl.screenW))) <= 2.0
	ctrl.compactStyle = cs
	ctrl.sizeScale = scale
	ctrl.opacityPct = opacity
	if wasCentered && ctrl.screenW > 0 {
		cX := float64(ctrl.centeredWinXLocked(ctrl.screenLeft, ctrl.screenW))
		ctrl.winTargetX = cX
		ctrl.winXFloat = cX
		ctrl.winX = int(math.Round(cX))
	}
	ctrl.mu.Unlock()
	if ctrl.hwnd != 0 {
		_, _, _ = procPostMessageW.Call(ctrl.hwnd, diWmAppSync, 0, 0)
	}
}

func (a *App) SetDynamicIslandHideWhenAppActive(hide bool) {
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}
	ctrl.mu.Lock()
	ctrl.hideWhenAppActive = hide
	ctrl.mu.Unlock()
	if ctrl.hwnd != 0 {
		_, _, _ = procPostMessageW.Call(ctrl.hwnd, diWmAppSync, 0, 0)
	}
}

func (a *App) SetDynamicIslandHideInFullscreen(hide bool) {
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}
	ctrl.mu.Lock()
	ctrl.hideInFullscreen = hide
	ctrl.mu.Unlock()
	if ctrl.hwnd != 0 {
		_, _, _ = procPostMessageW.Call(ctrl.hwnd, diWmAppSync, 0, 0)
	}
}

func (a *App) SetDynamicIslandIdleHideSeconds(sec int) {
	if sec < 0 {
		sec = 0
	}
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}
	ctrl.mu.Lock()
	ctrl.idleHideSec = sec
	ctrl.lastActivityAt = time.Now()
	ctrl.mu.Unlock()
	if ctrl.hwnd != 0 {
		_, _, _ = procPostMessageW.Call(ctrl.hwnd, diWmAppSync, 0, 0)
	}
}

func resolveMonitorRect(useCursor bool) (left, top, width int) {
	var hMon uintptr
	if useCursor {
		var pt diPoint
		_, _, _ = diProcGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		hMon, _, _ = diProcMonitorFromPoint.Call(uintptr(pt.X), uintptr(pt.Y), 2)
	} else {
		hMon, _, _ = diProcMonitorFromPoint.Call(0, 0, 1)
	}
	if hMon != 0 {
		var mi diMonitorInfo
		mi.CbSize = uint32(unsafe.Sizeof(mi))
		if ok, _, _ := diProcGetMonitorInfoW.Call(hMon, uintptr(unsafe.Pointer(&mi))); ok != 0 {
			w := int(mi.RcMonitor.Right - mi.RcMonitor.Left)
			if w > 0 {
				return int(mi.RcMonitor.Left), int(mi.RcMonitor.Top), w
			}
		}
	}
	sw, _, _ := diProcGetSystemMetrics.Call(0)
	return 0, 0, int(sw)
}

func (a *App) SetDynamicIslandScreen(mode string) {
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}
	m := "primary"
	if strings.ToLower(strings.TrimSpace(mode)) == "cursor" {
		m = "cursor"
	}
	monLeft, monTop, monW := resolveMonitorRect(m == "cursor")
	ctrl.mu.Lock()
	ctrl.screenMode = m
	ctrl.screenLeft = monLeft
	ctrl.screenTop = monTop
	ctrl.screenW = monW
	u := ctrl.effectiveScaleLocked()
	if ctrl.pillMode {
		ctrl.winY = monTop + int(math.Round(10.0*u))
	} else {
		ctrl.winY = monTop
	}
	centerX := ctrl.centeredWinXLocked(monLeft, monW)
	ctrl.winTargetX = float64(centerX)
	ctrl.dockSide = 0
	ctrl.mu.Unlock()
	if ctrl.hwnd != 0 {
		_, _, _ = procPostMessageW.Call(ctrl.hwnd, diWmAppSync, 0, 0)
	}
}

func (a *App) SetDynamicIslandAnimSpeed(speed string) {
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}
	scale := 1.0
	switch strings.ToLower(strings.TrimSpace(speed)) {
	case "slow":
		scale = 1.45
	case "fast":
		scale = 0.62
	}
	ctrl.mu.Lock()
	ctrl.animSpeedScale = scale
	ctrl.mu.Unlock()
}

func (a *App) SetDynamicIslandOpenTrigger(trigger string) {
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}
	t := "hover"
	if strings.ToLower(strings.TrimSpace(trigger)) == "click" {
		t = "click"
	}
	ctrl.mu.Lock()
	ctrl.openTrigger = t
	ctrl.mu.Unlock()
}

func (a *App) SetDynamicIslandOnlyPlaying(only bool) {
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}
	ctrl.mu.Lock()
	ctrl.onlyPlaying = only
	ctrl.mu.Unlock()
	if ctrl.hwnd != 0 {
		_, _, _ = procPostMessageW.Call(ctrl.hwnd, diWmAppSync, 0, 0)
	}
}

func (a *App) SetDynamicIslandPauseHideSeconds(sec int) {
	if sec < 0 {
		sec = 0
	}
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}
	ctrl.mu.Lock()
	ctrl.pauseHideSec = sec
	if !ctrl.isPlaying && ctrl.pausedAt.IsZero() {
		ctrl.pausedAt = time.Now()
	}
	ctrl.mu.Unlock()
	if ctrl.hwnd != 0 {
		_, _, _ = procPostMessageW.Call(ctrl.hwnd, diWmAppSync, 0, 0)
	}
}

func (a *App) SetDynamicIslandContent(showStatus, showCover, showClosedCover, showArtist, showLyric, showHeart, showDots, showProgress, showClosedProgress bool, progressStyle string) {
	style := strings.ToLower(strings.TrimSpace(progressStyle))
	if style != "thick" && style != "dots" && style != "wave" {
		style = "thin"
	}
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}
	ctrl.mu.Lock()
	ctrl.showStatus = showStatus
	ctrl.showCover = showCover
	ctrl.showClosedCover = showClosedCover
	ctrl.showArtist = showArtist
	ctrl.showLyric = showLyric
	ctrl.showHeart = showHeart
	ctrl.showDots = showDots
	ctrl.showProgress = showProgress
	ctrl.showClosedProgress = showClosedProgress
	ctrl.progressStyle = style
	ctrl.mu.Unlock()
	if ctrl.hwnd != 0 {
		_, _, _ = procPostMessageW.Call(ctrl.hwnd, diWmAppSync, 0, 0)
	}
}

func (a *App) SetDynamicIslandCoverOptions(coverStyle string, coverSpin, closedCoverSpin bool) {
	cs := strings.ToLower(strings.TrimSpace(coverStyle))
	if cs != "circle" {
		cs = "square"
	}
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}
	ctrl.mu.Lock()
	ctrl.coverStyle = cs
	ctrl.coverSpin = coverSpin
	ctrl.closedCoverSpin = closedCoverSpin
	ctrl.mu.Unlock()
	if ctrl.hwnd != 0 {
		_, _, _ = procPostMessageW.Call(ctrl.hwnd, diWmAppSync, 0, 0)
	}
}

func (a *App) SetDynamicIslandStyle(style string) {
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}
	isPill := strings.EqualFold(strings.TrimSpace(style), "pill")
	ctrl.mu.Lock()
	ctrl.pillMode = isPill
	if isPill {
		ctrl.dockSide = 0
		ctrl.dockLeftProg = 0
		ctrl.dockRightProg = 0
		u := ctrl.dpiScale
		if u <= 0 {
			u = 1.0
		}
		ctrl.winY = maxInt(0, int(math.Round(6.0*u))-3)
	} else {
		ctrl.winY = 0
	}
	ctrl.mu.Unlock()
	if ctrl.hwnd != 0 {
		_, _, _ = procPostMessageW.Call(ctrl.hwnd, diWmAppSync, 0, 0)
	}
}

func (a *App) NotifyDynamicIslandAudioDevice(deviceName string) {
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}
	ctrl.pollAudioHardware(true)
	if strings.TrimSpace(deviceName) != "" {
		ctrl.mu.Lock()
		ctrl.audioDeviceName = strings.TrimSpace(deviceName)
		ctrl.pinnedMode = diModeHeadphones
		ctrl.pinnedUntil = time.Now().Add(4500 * time.Millisecond)
		ctrl.mu.Unlock()
	}
	if ctrl.hwnd != 0 {
		_, _, _ = procPostMessageW.Call(ctrl.hwnd, diWmAppSync, 0, 0)
	}
}

func (a *App) SyncDynamicIsland(p DynamicIslandSyncPayload) {
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}

	cleanTitle := strings.TrimSpace(p.Title)
	cleanArtist := strings.TrimSpace(p.Artist)
	cleanAlbum := strings.TrimSpace(p.Album)

	ctrl.mu.Lock()
	coverChanged := ctrl.coverURL != p.CoverURL
	trackChanged := cleanTitle != "" && (ctrl.title != cleanTitle || ctrl.artist != cleanArtist)
	ctrl.title = cleanTitle
	ctrl.artist = cleanArtist
	ctrl.album = cleanAlbum
	ctrl.coverURL = p.CoverURL
	ctrl.isPlaying = p.IsPlaying
	ctrl.isFavorite = p.IsFavorite
	if p.PositionSec >= 0 {
		ctrl.posSec = p.PositionSec
		ctrl.posUpdatedAt = time.Now()
	}
	if p.DurationSec >= 0 {
		ctrl.durSec = p.DurationSec
	}
	ctrl.lyricOffsetSec = p.LyricOffsetSec
	if p.Volume >= 0 && p.Volume <= 100 {
		ctrl.volume = p.Volume
	}
	ctrl.muted = p.Muted
	ctrl.shuffle = p.Shuffle
	if p.RepeatMode != "" {
		ctrl.repeatMode = p.RepeatMode
	}
	if p.EqPreset != "" {
		ctrl.eqPreset = p.EqPreset
	}
	if p.ThemeSeed != "" {
		ctrl.themeSeed = p.ThemeSeed
	}
	if p.AccentColor != "" {
		ctrl.accentHex = p.AccentColor
	}
	if p.StreakDays >= 0 {
		ctrl.streakDays = p.StreakDays
	}
	ctrl.lyricsLoading = p.LyricsLoading
	ctrl.trackLoading = p.TrackLoading
	ctrl.queueTitle = strings.TrimSpace(p.QueueTitle)
	prevQueueIdx := ctrl.queueIndex
	ctrl.queueIndex = p.QueueIndex
	ctrl.queueItems = p.Queue
	maxScrollDp := math.Max(0.0, float64(len(ctrl.queueItems))*54.0-162.0)
	if len(ctrl.queueItems) == 0 {
		ctrl.queueScroll = 0
		ctrl.queueScrollDp = 0
		ctrl.queueTargetScrollDp = 0
	} else {
		if p.QueueIndex != prevQueueIdx && p.QueueIndex >= 0 {
			target := math.Max(0.0, math.Min(maxScrollDp, float64(p.QueueIndex)*54.0))
			ctrl.queueTargetScrollDp = target
			if !ctrl.queueOpen {
				ctrl.queueScrollDp = target
			}
		} else {
			ctrl.queueTargetScrollDp = math.Max(0.0, math.Min(maxScrollDp, ctrl.queueTargetScrollDp))
			ctrl.queueScrollDp = math.Max(0.0, math.Min(maxScrollDp, ctrl.queueScrollDp))
		}
		ctrl.queueScroll = int(math.Round(ctrl.queueTargetScrollDp / 54.0))
	}

	var urlsToFetch []string
	startFetch := maxInt(0, p.QueueIndex-4)
	endFetch := minInt(len(p.Queue), startFetch+16)
	for i := startFetch; i < endFetch; i++ {
		q := p.Queue[i]
		if q.CoverURL != "" && ctrl.queueArtCache[q.CoverURL] == nil && !ctrl.queueArtPend[q.CoverURL] {
			ctrl.queueArtPend[q.CoverURL] = true
			urlsToFetch = append(urlsToFetch, q.CoverURL)
		}
	}

	if cleanTitle == "" {
		ctrl.coverImg = nil
		ctrl.coverImgCircle = nil
		ctrl.rawCover = nil
		ctrl.hasCoverAcc = false
	}
	if trackChanged {
		ctrl.lyricIdx = -1
		ctrl.lyricFromY = 0
		ctrl.lyricToY = 0
		ctrl.lyricCurY = 0
		ctrl.marqueeStart = time.Now()
		ctrl.coverSpinRad = 0
		ctrl.closedSpinRad = 0
	}
	needCoverFetch := p.CoverURL != "" && (coverChanged || (ctrl.coverImg == nil && !ctrl.coverLoading))
	if needCoverFetch {
		ctrl.coverLoading = true
	}
	ctrl.mu.Unlock()

	if needCoverFetch {
		go ctrl.loadCoverAsync(p.CoverURL)
	}
	for _, qURL := range urlsToFetch {
		go ctrl.loadQueueArtAsync(qURL)
	}
	if ctrl.hwnd != 0 {
		_, _, _ = procPostMessageW.Call(ctrl.hwnd, diWmAppSync, 0, 0)
	}
}

func (a *App) UpdateDynamicIslandProgress(positionSec float64, durationSec float64) {
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}
	ctrl.mu.Lock()
	jumped := math.Abs(positionSec-ctrl.posSec) >= 0.45
	ctrl.posSec = positionSec
	ctrl.durSec = durationSec
	ctrl.posUpdatedAt = time.Now()
	hwnd := ctrl.hwnd
	ctrl.mu.Unlock()
	if jumped && hwnd != 0 {
		_, _, _ = procPostMessageW.Call(hwnd, diWmAppSync, 0, 0)
	}
}

func (a *App) SetDynamicIslandLyricOffset(offsetSec float64) {
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}
	ctrl.mu.Lock()
	ctrl.lyricOffsetSec = offsetSec
	hwnd := ctrl.hwnd
	ctrl.mu.Unlock()
	if hwnd != 0 {
		_, _, _ = procPostMessageW.Call(hwnd, diWmAppSync, 0, 0)
	}
}

func (a *App) SetDynamicIslandLyricSweep(enabled bool) {
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}
	ctrl.mu.Lock()
	ctrl.lyricSweep = enabled
	hwnd := ctrl.hwnd
	ctrl.mu.Unlock()
	if hwnd != 0 {
		_, _, _ = procPostMessageW.Call(hwnd, diWmAppSync, 0, 0)
	}
}

func (a *App) updateDynamicIslandLyrics(synced []sources.LyricLine, plainLyrics string, source string) {
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}
	var list []sources.LyricLine
	if len(synced) > 0 {
		list = make([]sources.LyricLine, 0, len(synced))
		for _, l := range synced {
			txt := strings.TrimSpace(l.Text)
			if txt != "" {
				list = append(list, sources.LyricLine{TimeSec: l.TimeSec, Text: txt})
			}
		}
	}
	hasPlain := strings.TrimSpace(plainLyrics) != "" && !strings.EqualFold(strings.TrimSpace(source), "fallback")
	ctrl.mu.Lock()
	ctrl.lyrics = list
	ctrl.lyricsHasPlain = hasPlain
	ctrl.lyricsSource = source
	ctrl.lyricsLoading = false
	ctrl.lyricIdx = -1
	ctrl.prevLyricIdx = -1
	ctrl.lyricFromY = 0
	ctrl.lyricToY = 0
	ctrl.lyricCurY = 0
	ctrl.closedLyricFrom = ""
	ctrl.closedLyricTo = ""
	hwnd := ctrl.hwnd
	ctrl.mu.Unlock()
	if hwnd != 0 {
		_, _, _ = procPostMessageW.Call(hwnd, diWmAppSync, 0, 0)
	}
}

func (a *App) updateDynamicIslandState(title, artist, coverURL string, isPlaying bool) {
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	if ctrl == nil {
		return
	}

	cleanTitle := strings.TrimSpace(title)
	cleanArtist := strings.TrimSpace(artist)
	ctrl.mu.Lock()
	coverChanged := ctrl.coverURL != coverURL
	trackChanged := cleanTitle != "" && (ctrl.title != cleanTitle || ctrl.artist != cleanArtist)
	ctrl.title = cleanTitle
	ctrl.artist = cleanArtist
	ctrl.coverURL = coverURL
	ctrl.isPlaying = isPlaying
	if cleanTitle == "" {
		ctrl.coverImg = nil
		ctrl.coverImgCircle = nil
		ctrl.rawCover = nil
		ctrl.hasCoverAcc = false
	}
	if trackChanged {
		ctrl.lyricIdx = -1
		ctrl.lyricFromY = 0
		ctrl.lyricToY = 0
		ctrl.lyricCurY = 0
		ctrl.marqueeStart = time.Now()
		ctrl.coverSpinRad = 0
		ctrl.closedSpinRad = 0
	}
	ctrl.mu.Unlock()

	if coverChanged && coverURL != "" {
		go ctrl.loadCoverAsync(coverURL)
	}
	if ctrl.hwnd != 0 {
		_, _, _ = procPostMessageW.Call(ctrl.hwnd, diWmAppSync, 0, 0)
	}
}

func (c *dynamicIslandController) pollAudioHardware(allowAutoPopup bool) {
	now := time.Now()
	c.mu.Lock()
	if c.hwPolling || (!c.lastHwPoll.IsZero() && now.Sub(c.lastHwPoll) < 2500*time.Millisecond && allowAutoPopup) {
		c.mu.Unlock()
		return
	}
	c.hwPolling = true
	c.lastHwPoll = now
	prevDev := c.audioDeviceName
	prevBt := c.isBtAudio
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.hwPolling = false
		c.mu.Unlock()
	}()

	devName := ""
	connType := "speaker"
	isBt := false

	numDevs, _, _ := diProcWaveOutGetNumDevs.Call()
	if numDevs > 0 {
		var caps diWaveOutCapsW
		res, _, _ := diProcWaveOutGetDevCapsW.Call(0, uintptr(unsafe.Pointer(&caps)), unsafe.Sizeof(caps))
		if res == 0 {
			raw := windows.UTF16ToString(caps.SzPname[:])
			raw = strings.TrimSpace(raw)
			if raw != "" {
				devName = cleanAudioDeviceName(raw)
				low := strings.ToLower(raw)
				if strings.Contains(low, "bluetooth") || strings.Contains(low, "bth") || strings.Contains(low, "hands-free") || strings.Contains(low, "buds") || strings.Contains(low, "airpods") || strings.Contains(low, "soundcore") || strings.Contains(low, "tws") {
					isBt = true
					connType = "bluetooth"
				} else if strings.Contains(low, "headphone") || strings.Contains(low, "headset") || strings.Contains(low, "earphone") || strings.Contains(low, "usb") || strings.Contains(low, "dac") || strings.Contains(low, "jack") {
					connType = "wired"
				}
			}
		}
	}

	// DEVPKEY_Bluetooth_Battery {104EA319-6EE2-4701-BD47-8DDBF425BBE5} pid 2
	btBat, btFriendly := queryBluetoothAudioBattery()
	if btBat >= 0 && btBat <= 100 {
		isBt = true
		connType = "bluetooth"
		if devName == "" || strings.Contains(strings.ToLower(devName), "mapper") {
			devName = btFriendly
		}
	}
	if devName == "" {
		devName = "Speakers (System Audio)"
	}

	sysBat := -1
	type systemPowerStatus struct {
		ACLineStatus        byte
		BatteryFlag         byte
		BatteryLifePercent  byte
		SystemStatusFlag    byte
		BatteryLifeTime     uint32
		BatteryFullLifeTime uint32
	}
	var sps systemPowerStatus
	if r, _, _ := diProcGetSystemPowerStatus.Call(uintptr(unsafe.Pointer(&sps))); r != 0 {
		if sps.BatteryFlag != 128 && sps.BatteryLifePercent <= 100 {
			sysBat = int(sps.BatteryLifePercent)
		}
	}

	wifiOn := queryWifiConnected()
	gpsOn := queryLocationEnabled()
	btOn := queryBluetoothRadioOn() || isBt
	outputs := listWindowsAudioOutputs(btBat, btFriendly)

	c.mu.Lock()
	changed := (prevDev != "" && prevDev != "Velora Audio Output" && prevDev != devName) || (!prevBt && isBt && prevDev != "Velora Audio Output")
	c.audioDeviceName = devName
	c.audioConnType = connType
	c.btBatteryPct = btBat
	c.sysBatteryPct = sysBat
	c.isBtAudio = isBt
	c.audioOutputs = outputs
	c.wifiOn = wifiOn
	c.gpsOn = gpsOn
	c.btOn = btOn
	if changed && allowAutoPopup {
		c.pinnedMode = diModeHeadphones
		c.pinnedUntil = now.Add(4500 * time.Millisecond)
	}
	c.mu.Unlock()
}

func queryWifiConnected() bool {
	var negVer uint32
	var hClient uintptr
	res, _, _ := diProcWlanOpenHandle.Call(2, 0, uintptr(unsafe.Pointer(&negVer)), uintptr(unsafe.Pointer(&hClient)))
	if res != 0 || hClient == 0 {
		return false
	}
	defer diProcWlanCloseHandle.Call(hClient, 0)

	var pList uintptr
	res2, _, _ := diProcWlanEnumInterfaces.Call(hClient, 0, uintptr(unsafe.Pointer(&pList)))
	if res2 != 0 || pList == 0 {
		return false
	}
	defer diProcWlanFreeMemory.Call(pList)

	num := *(*uint32)(unsafe.Pointer(pList))
	if num == 0 {
		return false
	}
	for i := uint32(0); i < num && i < 8; i++ {
		itemPtr := pList + 8 + uintptr(i)*(16+512+4)
		state := *(*uint32)(unsafe.Pointer(itemPtr + 16 + 512))
		if state == 1 { // wlan_interface_state_connected
			return true
		}
	}
	return false
}

func queryLocationEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\location`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	val, _, err := k.GetStringValue("Value")
	if err != nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(val), "Allow")
}

func queryBluetoothRadioOn() bool {
	type bthFindParams struct {
		DwSize uint32
	}
	params := bthFindParams{DwSize: uint32(unsafe.Sizeof(bthFindParams{}))}
	var hRadio uintptr
	hFind, _, _ := diProcBthFindFirstRadio.Call(uintptr(unsafe.Pointer(&params)), uintptr(unsafe.Pointer(&hRadio)))
	if hFind == 0 {
		return false
	}
	_, _, _ = diProcBthFindRadioClose.Call(hFind)
	if hRadio == 0 {
		return false
	}
	defer windows.CloseHandle(windows.Handle(hRadio))
	conn, _, _ := diProcBthIsConnectable.Call(hRadio)
	return conn != 0
}

func (c *dynamicIslandController) toggleWindowsLocation(turnOn bool) {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\CapabilityAccessManager\ConsentStore\location`, registry.SET_VALUE)
	if err == nil {
		val := "Deny"
		if turnOn {
			val = "Allow"
		}
		_ = k.SetStringValue("Value", val)
		_ = k.Close()
	}
	time.Sleep(120 * time.Millisecond)
	c.pollAudioHardware(false)
}

func (c *dynamicIslandController) toggleWindowsRadio(kind string, turnOn bool) {
	targetState := "Off"
	if turnOn {
		targetState = "On"
	}
	psScript := fmt.Sprintf(`Add-Type -AssemblyName System.Runtime.WindowsRuntime
$asTaskGeneric = ([System.WindowsRuntimeSystemExtensions].GetMethods() | Where-Object { $_.Name -eq 'AsTask' -and $_.GetParameters().Count -eq 1 -and $_.GetParameters()[0].ParameterType.Name -eq 'IAsyncOperation`+"`"+`1' })[0]
Function Await($WinRtTask, $ResultType) {
    $asTask = $asTaskGeneric.MakeGenericMethod($ResultType)
    $netTask = $asTask.Invoke($null, @($WinRtTask))
    $netTask.Wait(-1) | Out-Null
    $netTask.Result
}
[Windows.Devices.Radios.Radio,Windows.System.Devices,ContentType=WindowsRuntime] | Out-Null
Await ([Windows.Devices.Radios.Radio]::RequestAccessAsync()) ([Windows.Devices.Radios.RadioAccessStatus]) | Out-Null
$radios = Await ([Windows.Devices.Radios.Radio]::GetRadiosAsync()) ([System.Collections.Generic.IReadOnlyList[Windows.Devices.Radios.Radio]])
foreach ($r in $radios) {
    if ($r.Kind.ToString() -eq '%s') {
        Await ($r.SetStateAsync([Windows.Devices.Radios.RadioState]::%s)) ([Windows.Devices.Radios.RadioAccessStatus]) | Out-Null
    }
}`, kind, targetState)

	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", psScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	_ = cmd.Run()
	time.Sleep(350 * time.Millisecond)
	c.pollAudioHardware(false)
}

func cleanAudioDeviceName(raw string) string {
	if idx := strings.Index(raw, "("); idx != -1 {
		prefix := strings.TrimSpace(raw[:idx])
		inner := strings.TrimSpace(raw[idx+1:])
		inner = strings.TrimSuffix(inner, ")")
		inner = strings.TrimSpace(inner)
		if inner != "" {
			lowPref := strings.ToLower(prefix)
			if strings.Contains(lowPref, "headphone") || strings.Contains(lowPref, "headset") {
				return inner
			}
			if strings.Contains(lowPref, "speaker") {
				return inner + " · Speakers"
			}
			return inner
		}
	}
	return raw
}

func queryBluetoothAudioBattery() (int, string) {
	enumStr, _ := windows.UTF16PtrFromString("BTHENUM")
	hDev, _, _ := diProcSetupDiGetClassDevsW.Call(0, uintptr(unsafe.Pointer(enumStr)), 0, 0x02|0x04)
	if hDev == 0 || hDev == ^uintptr(0) {
		return -1, ""
	}
	defer diProcSetupDiDestroyDevList.Call(hDev)

	pkeyBat := diDevPropKey{
		Fmtid: diGuid{
			Data1: 0x104EA319,
			Data2: 0x6EE2,
			Data3: 0x4701,
			Data4: [8]byte{0xBD, 0x47, 0x8D, 0xDB, 0xF4, 0x25, 0xBB, 0xE5},
		},
		Pid: 2,
	}
	pkeyName := diDevPropKey{
		Fmtid: diGuid{
			Data1: 0xA45C254E,
			Data2: 0xDF1C,
			Data3: 0x4EFD,
			Data4: [8]byte{0x80, 0x20, 0x67, 0xD1, 0x46, 0xA8, 0x50, 0xE0},
		},
		Pid: 14,
	}

	for idx := uintptr(0); idx < 64; idx++ {
		var dev diSpDevInfoData
		dev.CbSize = uint32(unsafe.Sizeof(dev))
		ok, _, _ := diProcSetupDiEnumDeviceInfo.Call(hDev, idx, uintptr(unsafe.Pointer(&dev)))
		if ok == 0 {
			break
		}
		var propType uint32
		var reqSize uint32
		var bat byte
		okBat, _, _ := diProcSetupDiGetDevPropertyW.Call(
			hDev,
			uintptr(unsafe.Pointer(&dev)),
			uintptr(unsafe.Pointer(&pkeyBat)),
			uintptr(unsafe.Pointer(&propType)),
			uintptr(unsafe.Pointer(&bat)),
			1,
			uintptr(unsafe.Pointer(&reqSize)),
			0,
		)
		if okBat != 0 && bat <= 100 {
			var nameBuf [128]uint16
			okName, _, _ := diProcSetupDiGetDevPropertyW.Call(
				hDev,
				uintptr(unsafe.Pointer(&dev)),
				uintptr(unsafe.Pointer(&pkeyName)),
				uintptr(unsafe.Pointer(&propType)),
				uintptr(unsafe.Pointer(&nameBuf[0])),
				uintptr(len(nameBuf)*2),
				uintptr(unsafe.Pointer(&reqSize)),
				0,
			)
			friendly := ""
			if okName != 0 {
				friendly = windows.UTF16ToString(nameBuf[:])
				friendly = strings.TrimSpace(strings.TrimSuffix(friendly, "Hands-Free AG"))
				friendly = strings.TrimSpace(strings.TrimSuffix(friendly, "Avrcp Transport"))
			}
			return int(bat), friendly
		}
	}
	return -1, ""
}

func isTransientTaskSwitcher(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	var clsBuf [128]uint16
	n, _, _ := diProcGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&clsBuf[0])), uintptr(len(clsBuf)))
	if n == 0 {
		return false
	}
	switch windows.UTF16ToString(clsBuf[:n]) {
	case "XamlExplorerHostIslandWindow", "MultitaskingViewFrame", "TaskSwitcherWnd":
		return true
	}
	return false
}

func isForegroundFullscreen(islandHwnd uintptr) bool {
	fg, _, _ := diProcGetForegroundWindow.Call()
	if fg == 0 || fg == islandHwnd {
		return false
	}
	var clsBuf [128]uint16
	n, _, _ := diProcGetClassNameW.Call(fg, uintptr(unsafe.Pointer(&clsBuf[0])), uintptr(len(clsBuf)))
	if n > 0 {
		cls := windows.UTF16ToString(clsBuf[:n])
		switch cls {
		case "Progman", "WorkerW", "Shell_TrayWnd", "Shell_SecondaryTrayWnd", "NotifyIconOverflowWindow", "TopLevelWindowForOverflowXamlIsland":
			return false
		}
	}
	var qState uint32
	if hr, _, _ := diProcSHQueryUserNotificationState.Call(uintptr(unsafe.Pointer(&qState))); hr == 0 {
		// 3 = D3D fullscreen, 4 = presentation mode
		if qState == 3 || qState == 4 {
			return true
		}
	}
	gwlStyle := int32(-16)
	style, _, _ := diProcGetWindowLongW.Call(fg, uintptr(uint32(gwlStyle)))
	const wsCaption = 0x00C00000
	if (uint32(style) & wsCaption) == wsCaption {
		return false
	}
	var winRect diRect
	okR, _, _ := diProcGetWindowRect.Call(fg, uintptr(unsafe.Pointer(&winRect)))
	if okR == 0 {
		return false
	}
	hMon, _, _ := diProcMonitorFromWindow.Call(fg, 2)
	if hMon == 0 {
		return false
	}
	var mi diMonitorInfo
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	okM, _, _ := diProcGetMonitorInfoW.Call(hMon, uintptr(unsafe.Pointer(&mi)))
	if okM == 0 {
		return false
	}
	return winRect.Left <= mi.RcMonitor.Left &&
		winRect.Top <= mi.RcMonitor.Top &&
		winRect.Right >= mi.RcMonitor.Right &&
		winRect.Bottom >= mi.RcMonitor.Bottom
}

func shouldDynamicIslandShow(islandHwnd uintptr) bool {
	globalIslandMu.Lock()
	ctrl := globalIsland
	globalIslandMu.Unlock()
	hideOnFocus := true
	if ctrl != nil {
		ctrl.mu.Lock()
		hideOnFocus = ctrl.hideWhenAppActive
		ctrl.mu.Unlock()
	}
	if !hideOnFocus {
		return true
	}
	globalTaskbarMu.Lock()
	tb := globalTaskbar
	globalTaskbarMu.Unlock()
	if tb == nil || tb.hwnd == 0 {
		return false
	}
	iconic, _, _ := diProcIsIconic.Call(tb.hwnd)
	if iconic != 0 {
		return true
	}
	vis, _, _ := diProcIsWindowVisible.Call(tb.hwnd)
	if vis == 0 {
		return true
	}
	fg, _, _ := diProcGetForegroundWindow.Call()
	if fg != 0 && fg != tb.hwnd && fg != islandHwnd {
		var clsBuf [128]uint16
		n, _, _ := diProcGetClassNameW.Call(fg, uintptr(unsafe.Pointer(&clsBuf[0])), uintptr(len(clsBuf)))
		if n > 0 {
			cls := windows.UTF16ToString(clsBuf[:n])
			switch cls {
			case "Shell_TrayWnd", "Shell_SecondaryTrayWnd", "NotifyIconOverflowWindow", "TopLevelWindowForOverflowXamlIsland":
				return false
			}
		}
		return true
	}
	return false
}

func getWindowsFormattedTime() string {
	var buf [64]uint16
	n, _, _ := diProcGetTimeFormatEx.Call(0, 2, 0, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n > 1 {
		s := strings.TrimSpace(windows.UTF16ToString(buf[:n]))
		if s != "" {
			return s
		}
	}
	return time.Now().Format("3:04 PM")
}

func (c *dynamicIslandController) resolvePaletteLocked() veloraBrandPalette {
	pal := veloraBrandPalette{
		BgDeep:      color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 255}, // Pure Black (#000000) to blend with screen
		BgSurface:   color.RGBA{R: 0x22, G: 0x1C, B: 0x2B, A: 255}, // To Hell and Black (#221C2B)
		Primary:     color.RGBA{R: 0xBE, G: 0xCF, B: 0xE4, A: 255}, // Misty Mountains (#BECFE4)
		PrimaryDim:  color.RGBA{R: 0x32, G: 0x29, B: 0x42, A: 255}, // Deep Plum Container (#322942)
		Accent:      color.RGBA{R: 0xF6, G: 0x84, B: 0x7E, A: 255}, // Shell Pink (#F6847E)
		TextMain:    color.RGBA{R: 0xF5, G: 0xF5, B: 0xF7, A: 255}, // Crisp white
		TextVariant: color.RGBA{R: 0xC3, G: 0xBF, B: 0xC9, A: 255}, // Neutral variant gray
		TextOutline: color.RGBA{R: 0x51, G: 0x4C, B: 0x53, A: 255}, // Blackened Pearl (#514C53)
	}

	if c.hasCoverAcc {
		pal.CoverDom = c.coverAccent
		pal.HasCoverDom = true
	}
	return pal
}

func parseHexColor(hexStr string) (color.RGBA, bool) {
	s := strings.TrimPrefix(strings.TrimSpace(hexStr), "#")
	if len(s) != 6 {
		return color.RGBA{}, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return color.RGBA{}, false
	}
	return color.RGBA{
		R: uint8((v >> 16) & 0xFF),
		G: uint8((v >> 8) & 0xFF),
		B: uint8(v & 0xFF),
		A: 255,
	}, true
}

func extractCoverAccentColor(img image.Image) (color.RGBA, bool) {
	if img == nil {
		return color.RGBA{}, false
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return color.RGBA{}, false
	}
	stepX := maxInt(1, w/16)
	stepY := maxInt(1, h/16)
	var rSum, gSum, bSum, wSum float64
	for y := b.Min.Y; y < b.Max.Y; y += stepY {
		for x := b.Min.X; x < b.Max.X; x += stepX {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if c.A < 128 {
				continue
			}
			rf, gf, bf := float64(c.R), float64(c.G), float64(c.B)
			mx := math.Max(rf, math.Max(gf, bf))
			mn := math.Min(rf, math.Min(gf, bf))
			if mx < 28 || mn > 235 {
				continue
			}
			sat := 0.0
			if mx > 0 {
				sat = (mx - mn) / mx
			}
			weight := 0.25 + sat*1.8
			rSum += rf * weight
			gSum += gf * weight
			bSum += bf * weight
			wSum += weight
		}
	}
	if wSum <= 0 {
		return color.RGBA{}, false
	}
	r := uint8(clamp01(rSum/(wSum*255.0)) * 255.0)
	g := uint8(clamp01(gSum/(wSum*255.0)) * 255.0)
	bl := uint8(clamp01(bSum/(wSum*255.0)) * 255.0)
	return color.RGBA{R: r, G: g, B: bl, A: 255}, true
}

func (c *dynamicIslandController) loadCoverAsync(rawURL string) {
	img := c.fetchImage(rawURL)
	if img == nil {
		c.mu.Lock()
		c.coverLoading = false
		c.mu.Unlock()
		return
	}
	c.mu.Lock()
	u := c.dpiScale
	if c.sizeScale > 0 {
		u *= c.sizeScale
	}
	c.mu.Unlock()
	artPx := maxInt(46, int(math.Round(46.0*u)))
	art := resizeCoverBilinear(img, artPx, artPx, 8.0*u)
	artCirc := resizeCoverBilinear(img, artPx, artPx, float64(artPx)/2.0)
	miniPx := maxInt(21, int(math.Round(20.8*u)))
	artMini := resizeCoverBilinear(img, miniPx, miniPx, float64(miniPx)/2.0)
	dom, hasDom := extractCoverAccentColor(img)
	thumbRaw := resizeCoverBilinear(img, 96, 96, 0)
	c.mu.Lock()
	c.coverLoading = false
	if c.coverURL == rawURL {
		c.rawCover = thumbRaw
		c.coverImg = art
		c.coverImgCircle = artCirc
		c.coverImgMini = artMini
		c.coverAccent = dom
		c.hasCoverAcc = hasDom
	}
	c.mu.Unlock()
	if c.hwnd != 0 {
		_, _, _ = procPostMessageW.Call(c.hwnd, diWmAppSync, 0, 0)
	}
}

func (c *dynamicIslandController) fetchImage(rawURL string) image.Image {
	u := strings.TrimSpace(rawURL)
	if u == "" {
		return nil
	}
	var data []byte
	if strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://") {
		client := &http.Client{Timeout: 6 * time.Second}
		resp, err := client.Get(u)
		if err != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				_ = resp.Body.Close()
			}
			return nil
		}
		data, err = io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		_ = resp.Body.Close()
		if err != nil {
			return nil
		}
	} else {
		cleaned := strings.TrimPrefix(u, "file:///")
		cleaned = filepath.FromSlash(cleaned)
		var err error
		data, err = os.ReadFile(cleaned)
		if err != nil {
			return nil
		}
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	return img
}

func queryMonitorDpiScale() float64 {
	hmon, _, _ := diProcMonitorFromPoint.Call(0, 1) // MONITOR_DEFAULTTOPRIMARY
	if hmon != 0 {
		var dpiX, dpiY uint32
		hr, _, _ := diProcGetDpiForMonitor.Call(hmon, 0, uintptr(unsafe.Pointer(&dpiX)), uintptr(unsafe.Pointer(&dpiY)))
		if hr == 0 && dpiX >= 96 {
			return float64(dpiX) / 96.0
		}
	}
	dpi, _, _ := diProcGetDpiForSystem.Call()
	if dpi >= 96 {
		return float64(dpi) / 96.0
	}
	return 1.25
}

func (c *dynamicIslandController) loadQueueArtAsync(rawURL string) {
	img := c.fetchImage(rawURL)
	c.mu.Lock()
	u := c.dpiScale
	if c.sizeScale > 0 {
		u *= c.sizeScale
	}
	c.mu.Unlock()
	if img != nil {
		smPx := maxInt(38, int(math.Round(38.0*u)))
		art := resizeCoverBilinear(img, smPx, smPx, 6.0*u)
		c.mu.Lock()
		c.queueArtCache[rawURL] = art
		delete(c.queueArtPend, rawURL)
		if len(c.queueArtCache) > 32 {
			c.pruneQueueArtCacheLocked()
		}
		needMain := c.coverURL == rawURL && c.coverImg == nil
		c.mu.Unlock()
		if needMain {
			artPx := maxInt(46, int(math.Round(46.0*u)))
			mainArt := resizeCoverBilinear(img, artPx, artPx, 8.0*u)
			mainArtCirc := resizeCoverBilinear(img, artPx, artPx, float64(artPx)/2.0)
			miniPx := maxInt(21, int(math.Round(20.8*u)))
			miniArt := resizeCoverBilinear(img, miniPx, miniPx, float64(miniPx)/2.0)
			dom, hasDom := extractCoverAccentColor(img)
			thumbRaw := resizeCoverBilinear(img, 96, 96, 0)
			c.mu.Lock()
			if c.coverURL == rawURL && c.coverImg == nil {
				c.rawCover = thumbRaw
				c.coverImg = mainArt
				c.coverImgCircle = mainArtCirc
				c.coverImgMini = miniArt
				c.coverAccent = dom
				c.hasCoverAcc = hasDom
			}
			c.mu.Unlock()
		}
		if c.hwnd != 0 {
			_, _, _ = procPostMessageW.Call(c.hwnd, diWmAppSync, 0, 0)
		}
	} else {
		c.mu.Lock()
		delete(c.queueArtPend, rawURL)
		c.mu.Unlock()
	}
}

func (c *dynamicIslandController) pruneQueueArtCacheLocked() {
	if len(c.queueArtCache) <= 32 {
		return
	}
	keep := make(map[string]bool, 32)
	if c.coverURL != "" {
		keep[c.coverURL] = true
	}
	start := maxInt(0, c.queueIndex-6)
	end := minInt(len(c.queueItems), start+28)
	for i := start; i < end; i++ {
		if u := c.queueItems[i].CoverURL; u != "" {
			keep[u] = true
		}
	}
	for k := range c.queueArtCache {
		if !keep[k] {
			delete(c.queueArtCache, k)
		}
	}
}

func (c *dynamicIslandController) initPoppinsFonts() {
	allTTFs := [][]byte{poppinsRegularTTF, poppinsSemiBoldTTF, poppinsBoldTTF, phosphorRegularTTF, phosphorFillTTF}
	for _, ttf := range allTTFs {
		if len(ttf) > 0 {
			var numFonts uint32
			_, _, _ = diProcAddFontMemResourceEx.Call(
				uintptr(unsafe.Pointer(&ttf[0])),
				uintptr(len(ttf)),
				0,
				uintptr(unsafe.Pointer(&numFonts)),
			)
		}
	}

	var pfc uintptr
	st, _, _ := diProcGdipNewPrivateFontColl.Call(uintptr(unsafe.Pointer(&pfc)))
	if st == 0 && pfc != 0 {
		c.fontColl = pfc
		for _, ttf := range allTTFs {
			if len(ttf) > 0 {
				_, _, _ = diProcGdipPrivateAddMemoryFont.Call(
					pfc,
					uintptr(unsafe.Pointer(&ttf[0])),
					uintptr(len(ttf)),
				)
			}
		}
		nameReg, _ := windows.UTF16PtrFromString("Poppins")
		_, _, _ = diProcGdipCreateFontFamily.Call(uintptr(unsafe.Pointer(nameReg)), pfc, uintptr(unsafe.Pointer(&c.famRegular)))

		nameSemi, _ := windows.UTF16PtrFromString("Poppins SemiBold")
		_, _, _ = diProcGdipCreateFontFamily.Call(uintptr(unsafe.Pointer(nameSemi)), pfc, uintptr(unsafe.Pointer(&c.famSemiBold)))

		namePhReg, _ := windows.UTF16PtrFromString("Phosphor")
		_, _, _ = diProcGdipCreateFontFamily.Call(uintptr(unsafe.Pointer(namePhReg)), pfc, uintptr(unsafe.Pointer(&c.famPhosphorReg)))

		namePhFill, _ := windows.UTF16PtrFromString("Phosphor-Fill")
		_, _, _ = diProcGdipCreateFontFamily.Call(uintptr(unsafe.Pointer(namePhFill)), pfc, uintptr(unsafe.Pointer(&c.famPhosphorFill)))
	}

	if c.famRegular == 0 {
		fallback, _ := windows.UTF16PtrFromString("Segoe UI")
		_, _, _ = diProcGdipCreateFontFamily.Call(uintptr(unsafe.Pointer(fallback)), 0, uintptr(unsafe.Pointer(&c.famRegular)))
	}
	if c.famSemiBold == 0 {
		c.famSemiBold = c.famRegular
	}
}

func (c *dynamicIslandController) destroyPoppinsFonts() {
	if c.famPhosphorFill != 0 {
		_, _, _ = diProcGdipDeleteFontFamily.Call(c.famPhosphorFill)
		c.famPhosphorFill = 0
	}
	if c.famPhosphorReg != 0 {
		_, _, _ = diProcGdipDeleteFontFamily.Call(c.famPhosphorReg)
		c.famPhosphorReg = 0
	}
	if c.famSemiBold != 0 && c.famSemiBold != c.famRegular {
		_, _, _ = diProcGdipDeleteFontFamily.Call(c.famSemiBold)
		c.famSemiBold = 0
	}
	if c.famRegular != 0 {
		_, _, _ = diProcGdipDeleteFontFamily.Call(c.famRegular)
		c.famRegular = 0
	}
	if c.fontColl != 0 {
		_, _, _ = diProcGdipDeletePrivateFontColl.Call(c.fontColl)
		c.fontColl = 0
	}
}

func (c *dynamicIslandController) runWindowLoop() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	_, _, _ = diProcSetThreadDpiAwarenessContext.Call(^uintptr(3)) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4)

	u := queryMonitorDpiScale()
	physW := int(math.Ceil(float64(diLogicalW) * u))
	physH := int(math.Ceil(float64(diLogicalH) * u))
	artPx := int(math.Round(46.0 * u))

	var si diGdiplusStartupInput
	si.GdiplusVersion = 1
	_, _, _ = diProcGdiplusStartup.Call(uintptr(unsafe.Pointer(&c.gdiToken)), uintptr(unsafe.Pointer(&si)), 0)
	defer func() {
		c.destroyPoppinsFonts()
		if c.gdiToken != 0 {
			_, _, _ = diProcGdiplusShutdown.Call(c.gdiToken)
		}
	}()

	c.initPoppinsFonts()
	c.initD2DText()
	defer c.destroyD2DText()

	c.mu.Lock()
	c.dpiScale = u
	c.physW = physW
	c.physH = physH
	pal := c.resolvePaletteLocked()
	c.defaultArt = buildVeloraFallbackArt(artPx, artPx, 8.0*u, pal)
	c.defaultArtCirc = buildVeloraFallbackArt(artPx, artPx, float64(artPx)/2.0, pal)
	smPx := int(math.Round(38.0 * u))
	c.defaultArtSm = buildVeloraFallbackArt(smPx, smPx, 6.0*u, pal)
	miniPx := maxInt(21, int(math.Round(20.8*u)))
	c.defaultArtMini = buildVeloraFallbackArt(miniPx, miniPx, float64(miniPx)/2.0, pal)
	if c.rawCover != nil {
		c.coverImg = resizeCoverBilinear(c.rawCover, artPx, artPx, 8.0*u)
		c.coverImgCircle = resizeCoverBilinear(c.rawCover, artPx, artPx, float64(artPx)/2.0)
		c.coverImgMini = resizeCoverBilinear(c.rawCover, miniPx, miniPx, float64(miniPx)/2.0)
	}
	c.mu.Unlock()

	hInstRaw, _, _ := diProcGetModuleHandleW.Call(0)
	hInst := windows.Handle(hInstRaw)
	className, _ := windows.UTF16PtrFromString("VeloraDynamicIslandClass")
	c.cursorArrow, _, _ = diProcLoadCursorW.Call(0, 32512)
	c.cursorHand, _, _ = diProcLoadCursorW.Call(0, 32649)
	c.cursorMove, _, _ = diProcLoadCursorW.Call(0, 32646) // IDC_SIZEALL

	wc := diWndClassExW{
		CbSize:        uint32(unsafe.Sizeof(diWndClassExW{})),
		LpfnWndProc:   syscall.NewCallback(diWndProc),
		HInstance:     hInst,
		HCursor:       windows.Handle(c.cursorArrow),
		LpszClassName: className,
	}
	_, _, _ = diProcRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))

	screenW, _, _ := diProcGetSystemMetrics.Call(0)
	c.mu.Lock()
	c.screenW = int(screenW)
	x := c.centeredWinXLocked(c.screenLeft, c.screenW)
	c.winX = x
	c.winXFloat = float64(x)
	c.winTargetX = float64(x)
	c.mu.Unlock()

	titlePtr, _ := windows.UTF16PtrFromString("Velora Dynamic Island")
	exStyle := uintptr(diWsExTopmost | diWsExToolWindow | diWsExLayered | diWsExNoActivate)
	hwnd, _, _ := diProcCreateWindowExW.Call(
		exStyle,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(titlePtr)),
		diWsPopup,
		uintptr(x),
		0,
		uintptr(physW),
		uintptr(physH),
		0,
		0,
		uintptr(hInst),
		0,
	)
	if hwnd == 0 {
		return
	}
	c.hwnd = hwnd
	c.initDIB(physW, physH)
	defer c.destroyDIB()

	_, _, _ = diProcSetTimer.Call(hwnd, diTimerID, 15, 0)

	var msg diMsg
	for {
		ret, _, _ := diProcGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		_, _, _ = diProcTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		_, _, _ = diProcDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func (c *dynamicIslandController) initDIB(w, h int) {
	scrDC, _, _ := diProcGetDC.Call(0)
	memDC, _, _ := diProcCreateCompatibleDC.Call(scrDC)
	_, _, _ = diProcReleaseDC.Call(0, scrDC)

	bi := bitmapInfoHeader{
		BiSize:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		BiWidth:       int32(w),
		BiHeight:      -int32(h),
		BiPlanes:      1,
		BiBitCount:    32,
		BiCompression: 0,
	}
	var bits unsafe.Pointer
	bmp, _, _ := procCreateDIBSection.Call(memDC, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	oldBmp, _, _ := diProcSelectObject.Call(memDC, bmp)

	c.memDC = memDC
	c.dibBmp = bmp
	c.oldBmp = oldBmp
	c.dibBits = bits
}

func (c *dynamicIslandController) destroyDIB() {
	if c.memDC != 0 {
		if c.oldBmp != 0 {
			_, _, _ = diProcSelectObject.Call(c.memDC, c.oldBmp)
		}
		if c.dibBmp != 0 {
			_, _, _ = procDeleteObject.Call(c.dibBmp)
		}
		_, _, _ = diProcDeleteDC.Call(c.memDC)
	}
}

func diWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	globalIslandMu.Lock()
	c := globalIsland
	globalIslandMu.Unlock()

	if c == nil {
		ret, _, _ := diProcDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return ret
	}

	switch msg {
	case diWmTimer, diWmAppSync:
		c.tickAndRender()
		return 0
	case diWmSetCursor:
		c.mu.Lock()
		zone := c.hoverZone
		winDrag := c.winDragActive
		moveCur := c.cursorMove
		hand := c.cursorHand
		arrow := c.cursorArrow
		c.mu.Unlock()
		if winDrag && moveCur != 0 {
			_, _, _ = diProcSetCursor.Call(moveCur)
		} else if zone >= diHitArt && hand != 0 {
			_, _, _ = diProcSetCursor.Call(hand)
		} else if arrow != 0 {
			_, _, _ = diProcSetCursor.Call(arrow)
		}
		return 1
	case diWmMouseMove:
		mx := int(int16(lParam & 0xFFFF))
		my := int(int16((lParam >> 16) & 0xFFFF))
		c.handleMouseMove(mx, my)
		return 0
	case diWmMouseLeave:
		c.mu.Lock()
		if c.winDragActive || c.holdPending {
			c.mu.Unlock()
			return 0
		}
		uLeave := c.dpiScale
		if uLeave <= 0 {
			uLeave = 1.0
		}
		var ptLeave diPoint
		_, _, _ = diProcGetCursorPos.Call(uintptr(unsafe.Pointer(&ptLeave)))
		nxLeave := (float64(diLogicalW) - c.curW) / 2.0
		leftPx := float64(c.winX) + nxLeave*uLeave - 4.0*uLeave
		rightPx := float64(c.winX) + (nxLeave+c.curW)*uLeave + 4.0*uLeave
		topPx := float64(c.winY) - 4.0*uLeave
		botPx := float64(c.winY) + c.curH*uLeave + 4.0*uLeave
		if float64(ptLeave.X) >= leftPx && float64(ptLeave.X) <= rightPx && float64(ptLeave.Y) >= topPx && float64(ptLeave.Y) <= botPx {
			c.tracking = false
			c.mu.Unlock()
			return 0
		}
		c.hovered = false
		c.hoverZone = diHitNone
		c.hoverLyric = -1
		c.hoverQueueIdx = -1
		c.tracking = false
		c.dragActive = false
		c.queueDragScroll = false
		c.seekDragging = false
		c.pinnedMode = ""
		c.mu.Unlock()
		return 0
	case diWmMouseHWheel:
		delta := int16((wParam >> 16) & 0xFFFF)
		c.mu.Lock()
		if time.Since(c.lastHWheelAt) > 280*time.Millisecond {
			c.lastHWheelAt = time.Now()
			if delta > 0 {
				c.switchPageByDeltaLocked(1)
			} else if delta < 0 {
				c.switchPageByDeltaLocked(-1)
			}
		}
		c.mu.Unlock()
		return 0
	case diWmMouseWheel:
		delta := int16((wParam >> 16) & 0xFFFF)
		c.mu.Lock()
		mode := c.activeMode
		qLen := len(c.queueItems)
		if mode == diModeQueue && qLen > 3 {
			maxScrollDp := math.Max(0.0, float64(qLen)*54.0-162.0)
			stepDp := -(float64(delta) / 120.0) * 54.0
			c.queueTargetScrollDp = math.Max(0.0, math.Min(maxScrollDp, c.queueTargetScrollDp+stepDp))
			c.queueScroll = int(math.Round(c.queueTargetScrollDp / 54.0))
			c.mu.Unlock()
			return 0
		}
		if mode == diModeLyrics && len(c.lyrics) > 0 && c.lastMouseRx >= 246 {
			stepDp := -(float64(delta) / 120.0) * 32.0
			maxOff := float64(len(c.lyrics)) * 28.0
			c.lyricTargetOffsetDp = math.Max(-maxOff, math.Min(maxOff, c.lyricTargetOffsetDp+stepDp))
			c.lyricManualUntil = time.Now().Add(4500 * time.Millisecond)
			c.mu.Unlock()
			return 0
		}
		ctx := c.ctx
		c.mu.Unlock()
		if ctx != nil {
			if delta > 0 {
				wailsRuntime.EventsEmit(ctx, "island:action", "vol-up")
			} else if delta < 0 {
				wailsRuntime.EventsEmit(ctx, "island:action", "vol-down")
			}
		}
		return 0
	case diWmLButtonDown:
		mx := int(int16(lParam & 0xFFFF))
		my := int(int16((lParam >> 16) & 0xFFFF))
		c.mu.Lock()
		u := c.dpiScale
	if c.sizeScale > 0 {
		u *= c.sizeScale
	}
		if u <= 0 {
			u = 1.0
		}
		ry := float64(my) / u
		maxScrollDp := math.Max(0.0, float64(len(c.queueItems))*54.0-162.0)
		zone, ratio, _ := c.hitTestLocked(mx, my)
		if zone == diHitQueueScroll && c.activeMode == diModeQueue && maxScrollDp > 0 {
			c.queueDragScroll = true
			c.queueTargetScrollDp = clamp01((ry-82.0)/156.0) * maxScrollDp
			c.queueScrollDp = c.queueTargetScrollDp
			c.queueScroll = int(math.Round(c.queueTargetScrollDp / 54.0))
			c.mu.Unlock()
			return 0
		}
		if zone == diHitSeek {
			c.seekDragging = true
			c.hoverZone = diHitSeek
			c.hoverRatio = ratio
			dur := c.durSec
			ctx := c.ctx
			c.mu.Unlock()
			if ctx != nil && dur > 0 {
				wailsRuntime.EventsEmit(ctx, "island:seek", ratio*dur)
			}
			return 0
		}
		c.dragActive = true
		c.dragStartX = mx
		c.dragStartY = my
		c.queueDragStartScroll = c.queueTargetScrollDp
		c.lyricDragStartOffset = c.lyricTargetOffsetDp
		c.lyricDragScroll = false
		var pt diPoint
		_, _, _ = diProcGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		c.holdPending = true
		c.winDragActive = false
		c.holdStart = time.Now()
		c.holdStartScreenX = pt.X
		c.holdStartScreenY = pt.Y
		c.holdStartWinX = c.winX
		c.mu.Unlock()
		_, _, _ = diProcSetCapture.Call(hwnd)
		return 0
	case diWmLButtonUp:
		_, _, _ = diProcReleaseCapture.Call()
		mx := int(int16(lParam & 0xFFFF))
		my := int(int16((lParam >> 16) & 0xFFFF))
		c.mu.Lock()
		c.holdPending = false
		if c.winDragActive {
			c.winDragActive = false
			c.dragActive = false
			c.lyricDragScroll = false
			u := c.dpiScale
	if c.sizeScale > 0 {
		u *= c.sizeScale
	}
			if u <= 0 {
				u = 1.0
			}
			nx := (float64(diLogicalW) - c.curW) / 2.0
			rx := float64(mx)/u - nx
			ry := float64(my) / u
			if rx < 0 || rx > c.curW || ry < 0 || ry > c.curH {
				c.hovered = false
				c.hoverZone = diHitNone
			}
			arrow := c.cursorArrow
			c.mu.Unlock()
			if arrow != 0 {
				_, _, _ = diProcSetCursor.Call(arrow)
			}
			return 0
		}
		if c.lyricDragScroll {
			c.lyricDragScroll = false
			c.dragActive = false
			c.mu.Unlock()
			return 0
		}
		if c.seekDragging {
			c.seekDragging = false
			c.mu.Unlock()
			return 0
		}
		if c.queueDragScroll {
			c.queueDragScroll = false
			c.mu.Unlock()
			return 0
		}
		wasDrag := c.dragActive
		u := c.dpiScale
	if c.sizeScale > 0 {
		u *= c.sizeScale
	}
		if u <= 0 {
			u = 1.0
		}
		dxDp := float64(mx-c.dragStartX) / u
		dyDp := float64(my-c.dragStartY) / u
		c.dragActive = false
		if wasDrag && c.activeMode == diModeQueue && math.Abs(dyDp) > 8.0 && math.Abs(dyDp) > math.Abs(dxDp) {
			c.mu.Unlock()
			return 0
		}
		c.mu.Unlock()
		c.handleClick(mx, my)
		return 0
	case diWmAppClose:
		_, _, _ = diProcKillTimer.Call(hwnd, diTimerID)
		_, _, _ = diProcDestroyWindow.Call(hwnd)
		return 0
	case diWmDestroy:
		_, _, _ = diProcPostQuitMessage.Call(0)
		return 0
	}

	ret, _, _ := diProcDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return ret
}

func (c *dynamicIslandController) effectiveScaleLocked() float64 {
	u := c.dpiScale
	if c.sizeScale > 0 {
		u *= c.sizeScale
	}
	if u <= 0 {
		u = 1.0
	}
	return u
}

func (c *dynamicIslandController) centeredWinXLocked(screenLeft, screenW int) int {
	u := c.effectiveScaleLocked()
	// Pill horizontal center in DIB coordinates is always (diLogicalW / 2.0) * u.
	return screenLeft + int(math.Round(float64(screenW)/2.0-(float64(diLogicalW)/2.0)*u))
}

func (c *dynamicIslandController) clampWinXLocked(rawX int, withSnap bool) (int, int) {
	u := c.effectiveScaleLocked()
	screenW := c.screenW
	if screenW <= 0 {
		sw, _, _ := diProcGetSystemMetrics.Call(0)
		screenW = int(sw)
		c.screenW = screenW
	}
	screenLeft := c.screenLeft
	nx := (float64(diLogicalW) - c.curW) / 2.0
	marginPx := 0
	if c.pillMode {
		marginPx = int(math.Round(10.0 * u))
	}
	minX := screenLeft + marginPx - int(math.Round(nx*u))
	maxX := screenLeft + screenW - marginPx - int(math.Round((nx+c.curW)*u))
	centerX := c.centeredWinXLocked(screenLeft, screenW)
	if maxX < minX {
		minX = centerX
		maxX = minX
	}
	dock := 0
	if withSnap {
		centerSnapPx := int(math.Round(20.0 * u))
		edgeSnapPx := int(math.Round(5.0 * u))
		if rawX >= centerX-centerSnapPx && rawX <= centerX+centerSnapPx {
			rawX = centerX
		} else if rawX <= minX+edgeSnapPx {
			rawX = minX
			if !c.pillMode {
				dock = -1
			}
		} else if rawX >= maxX-edgeSnapPx {
			rawX = maxX
			if !c.pillMode {
				dock = 1
			}
		}
	}
	if rawX <= minX {
		if withSnap && !c.pillMode {
			dock = -1
		}
		return minX, dock
	}
	if rawX >= maxX {
		if withSnap && !c.pillMode {
			dock = 1
		}
		return maxX, dock
	}
	return rawX, dock
}

func (c *dynamicIslandController) handleMouseMove(mx, my int) {
	c.mu.Lock()
	if c.winDragActive {
		var pt diPoint
		_, _, _ = diProcGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		rawX := c.holdStartWinX + int(pt.X-c.holdStartScreenX)
		newTarget, dock := c.clampWinXLocked(rawX, true)
		c.winTargetX = float64(newTarget)
		c.dockSide = dock
		moveCur := c.cursorMove
		c.mu.Unlock()
		if moveCur != 0 {
			_, _, _ = diProcSetCursor.Call(moveCur)
		}
		return
	}
	if c.holdPending && !c.winDragActive {
		u := c.dpiScale
	if c.sizeScale > 0 {
		u *= c.sizeScale
	}
		if u <= 0 {
			u = 1.0
		}
		var pt diPoint
		_, _, _ = diProcGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		dxScreen := float64(pt.X - c.holdStartScreenX)
		dyScreen := float64(pt.Y - c.holdStartScreenY)
		isVertScroll := (c.activeMode == diModeQueue || (c.activeMode == diModeLyrics && c.lastMouseRx >= 246)) && math.Abs(dyScreen) > math.Abs(dxScreen) && math.Abs(dyScreen) > 5.0*u
		if isVertScroll {
			c.holdPending = false
		} else if math.Abs(dxScreen) > 5.0*u || time.Since(c.holdStart) >= 180*time.Millisecond {
			c.winDragActive = true
			c.holdPending = false
			c.dragActive = false
			c.lyricDragScroll = false
			newTarget, dock := c.clampWinXLocked(c.holdStartWinX+int(pt.X-c.holdStartScreenX), true)
			c.winTargetX = float64(newTarget)
			c.dockSide = dock
			moveCur := c.cursorMove
			c.mu.Unlock()
			if moveCur != 0 {
				_, _, _ = diProcSetCursor.Call(moveCur)
			}
			return
		}
	}
	if !c.tracking {
		var tme diTrackMouseEventStruct
		tme.CbSize = uint32(unsafe.Sizeof(tme))
		tme.DwFlags = diTmeLeave
		tme.HwndTrack = c.hwnd
		_, _, _ = diProcTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
		c.tracking = true
	}
	u := c.dpiScale
	if c.sizeScale > 0 {
		u *= c.sizeScale
	}
	if u <= 0 {
		u = 1.0
	}
	nxCur := (float64(diLogicalW)-c.curW)/2.0 - c.liquidSkewDp*0.35*(1.0-math.Max(c.dockLeftProg, c.dockRightProg))
	if nxCur < 0 {
		nxCur = 0
	} else if nxCur > float64(diLogicalW)-c.curW {
		nxCur = float64(diLogicalW) - c.curW
	}
	rxCur := float64(mx)/u - nxCur
	ryCur := float64(my) / u
	if rxCur >= 0 && rxCur <= c.curW && ryCur >= 0 && ryCur <= c.curH {
		isClosedState := c.activeMode == diModeClosed || c.activeMode == diModeResting || c.activeMode == diModeFullscreenHide
		if c.openTrigger != "click" || !isClosedState {
			c.hovered = true
		}
	}
	c.lastMouseRx = rxCur
	maxScrollDp := math.Max(0.0, float64(len(c.queueItems))*54.0-162.0)
	if c.queueDragScroll && c.activeMode == diModeQueue && maxScrollDp > 0 {
		ry := float64(my) / u
		c.queueTargetScrollDp = clamp01((ry-82.0)/156.0) * maxScrollDp
		c.queueScrollDp = c.queueTargetScrollDp
		c.queueScroll = int(math.Round(c.queueTargetScrollDp / 54.0))
	} else if c.dragActive && c.activeMode == diModeLyrics && len(c.lyrics) > 0 && c.lastMouseRx >= 246 {
		dyDp := float64(my-c.dragStartY) / u
		dxDp := float64(mx-c.dragStartX) / u
		if math.Abs(dyDp) > 6.0 && math.Abs(dyDp) > math.Abs(dxDp) {
			c.lyricDragScroll = true
			maxOff := float64(len(c.lyrics)) * 28.0
			c.lyricTargetOffsetDp = math.Max(-maxOff, math.Min(maxOff, c.lyricDragStartOffset-dyDp))
			c.lyricManualOffsetDp = c.lyricTargetOffsetDp
			c.lyricManualUntil = time.Now().Add(4500 * time.Millisecond)
		}
	} else if c.dragActive && c.activeMode == diModeQueue && maxScrollDp > 0 {
		dyDp := float64(my-c.dragStartY) / u
		dxDp := float64(mx-c.dragStartX) / u
		if math.Abs(dyDp) > 8.0 && math.Abs(dyDp) > math.Abs(dxDp) {
			c.queueTargetScrollDp = math.Max(0.0, math.Min(maxScrollDp, c.queueDragStartScroll-dyDp))
			c.queueScrollDp = c.queueTargetScrollDp
			c.queueScroll = int(math.Round(c.queueTargetScrollDp / 54.0))
		}
	}
	var qIdx int
	c.hoverZone, c.hoverRatio, qIdx = c.hitTestLocked(mx, my)
	if c.seekDragging {
		var seekRatio float64
		nx := (float64(diLogicalW) - c.curW) / 2.0
		rx := float64(mx)/u - nx
		if c.activeMode == diModeLyrics {
			seekRatio = clamp01((rx - 22) / 214.0)
		} else {
			seekRatio = clamp01((rx - 22) / 376.0)
		}
		c.hoverZone = diHitSeek
		c.hoverRatio = seekRatio
		dur := c.durSec
		ctx := c.ctx
		c.mu.Unlock()
		if ctx != nil && dur > 0 {
			wailsRuntime.EventsEmit(ctx, "island:seek", seekRatio*dur)
		}
		return
	}
	if c.activeMode == diModeQueue {
		c.hoverQueueIdx = qIdx
		c.hoverLyric = -1
	} else {
		c.hoverLyric = qIdx
		c.hoverQueueIdx = -1
	}
	c.mu.Unlock()
}

func (c *dynamicIslandController) currentPageIndexLocked() int {
	switch c.activeMode {
	case diModeQueue:
		return 1
	case diModeHeadphones:
		return 2
	default:
		return 0
	}
}

func (c *dynamicIslandController) setPageLocked(page int) {
	cur := c.currentPageIndexLocked()
	page = ((page % 3) + 3) % 3
	if page > cur {
		c.pageSlideDir = 1
	} else if page < cur {
		c.pageSlideDir = -1
	}
	c.pinnedMode = ""
	switch page {
	case 0:
		c.queueOpen = false
		c.headphonesOpen = false
		c.hpListOpen = false
	case 1:
		c.queueOpen = true
		c.headphonesOpen = false
		c.hpListOpen = false
		maxScrollDp := math.Max(0.0, float64(len(c.queueItems))*54.0-162.0)
		if maxScrollDp > 0 && c.queueIndex >= 0 {
			target := math.Max(0.0, math.Min(maxScrollDp, float64(c.queueIndex)*54.0))
			c.queueTargetScrollDp = target
			c.queueScrollDp = target
			c.queueScroll = int(math.Round(target / 54.0))
		} else {
			c.queueScroll = 0
			c.queueScrollDp = 0
			c.queueTargetScrollDp = 0
		}
	case 2:
		c.queueOpen = false
		c.headphonesOpen = true
		go c.pollAudioHardware(false)
	}
}

func (c *dynamicIslandController) switchPageByDeltaLocked(delta int) {
	cur := c.currentPageIndexLocked()
	c.setPageLocked(cur + delta)
	if delta > 0 {
		c.pageSlideDir = 1
	} else if delta < 0 {
		c.pageSlideDir = -1
	}
}

func (c *dynamicIslandController) handleClick(mx, my int) {
	c.mu.Lock()
	if c.openTrigger == "click" && (c.activeMode == diModeClosed || c.activeMode == diModeResting || c.activeMode == diModeFullscreenHide) {
		c.hovered = true
		c.lastActivityAt = time.Now()
		c.mu.Unlock()
		return
	}
	zone, ratio, lyrIdx := c.hitTestLocked(mx, my)
	dur := c.durSec
	ctx := c.ctx
	var lyricSec float64 = -1
	if zone == diHitLyric && lyrIdx >= 0 && lyrIdx < len(c.lyrics) {
		lyricSec = c.lyrics[lyrIdx].TimeSec
	}

	switch zone {
	case diHitBubble:
		c.lyricsOpen = !c.lyricsOpen
		c.queueOpen = false
		c.pinnedMode = ""
		c.mu.Unlock()
		return
	case diHitQueueOpen:
		c.queueOpen = true
		c.lyricsOpen = false
		c.pinnedMode = ""
		maxScrollDp := math.Max(0.0, float64(len(c.queueItems))*54.0-162.0)
		if maxScrollDp > 0 && c.queueIndex >= 0 {
			target := math.Max(0.0, math.Min(maxScrollDp, float64(c.queueIndex)*54.0))
			c.queueTargetScrollDp = target
			c.queueScrollDp = target
			c.queueScroll = int(math.Round(target / 54.0))
		} else {
			c.queueScroll = 0
			c.queueScrollDp = 0
			c.queueTargetScrollDp = 0
		}
		c.mu.Unlock()
		return
	case diHitQueueBack:
		c.queueOpen = false
		c.pinnedMode = ""
		c.mu.Unlock()
		return
	case diHitArrow:
		c.headphonesOpen = !c.headphonesOpen
		c.queueOpen = false
		c.pinnedMode = ""
		c.mu.Unlock()
		go c.pollAudioHardware(false)
		return
	case diHitAudioDev0, diHitAudioDev1, diHitAudioDev2:
		k := zone - diHitAudioDev0
		others := diOtherAudioIndices(c.audioOutputs)
		var devID string
		if k >= 0 && k < len(others) {
			devID = c.audioOutputs[others[k]].ID
			c.hpListOpen = false
		}
		c.mu.Unlock()
		if devID != "" {
			go func(id string) {
				if setDefaultWindowsAudioOutput(id) {
					time.Sleep(150 * time.Millisecond)
					c.pollAudioHardware(false)
				}
			}(devID)
		}
		return
	case diHitDot0:
		c.setPageLocked(0)
		c.mu.Unlock()
		return
	case diHitDot1:
		c.setPageLocked(1)
		c.mu.Unlock()
		return
	case diHitDot2:
		c.setPageLocked(2)
		c.mu.Unlock()
		return
	case diHitDot3:
		c.setPageLocked(3)
		c.mu.Unlock()
		return
	case diHitDot4:
		c.setPageLocked(3)
		c.mu.Unlock()
		return
	case diHitWifi:
		nextState := !c.wifiOn
		c.wifiOn = nextState
		c.mu.Unlock()
		go c.toggleWindowsRadio("WiFi", nextState)
		return
	case diHitGps:
		nextState := !c.gpsOn
		c.gpsOn = nextState
		c.mu.Unlock()
		go c.toggleWindowsLocation(nextState)
		return
	case diHitBt:
		nextState := !c.btOn
		c.btOn = nextState
		c.mu.Unlock()
		go c.toggleWindowsRadio("Bluetooth", nextState)
		return
	case diHitEdgeLeft:
		c.switchPageByDeltaLocked(-1)
		c.mu.Unlock()
		return
	case diHitEdgeRight:
		c.switchPageByDeltaLocked(1)
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	if ctx == nil {
		return
	}
	switch zone {
	case diHitPrev:
		wailsRuntime.EventsEmit(ctx, "taskbar:control", "prev")
	case diHitPlay:
		wailsRuntime.EventsEmit(ctx, "taskbar:control", "toggle")
	case diHitNext:
		wailsRuntime.EventsEmit(ctx, "taskbar:control", "next")
	case diHitFav:
		wailsRuntime.EventsEmit(ctx, "island:action", "toggle-favorite")
	case diHitShuffle:
		wailsRuntime.EventsEmit(ctx, "island:action", "toggle-shuffle")
	case diHitRepeat:
		wailsRuntime.EventsEmit(ctx, "island:action", "cycle-repeat")
	case diHitMute:
		wailsRuntime.EventsEmit(ctx, "island:action", "toggle-mute")
	case diHitEqCycle:
		wailsRuntime.EventsEmit(ctx, "island:action", "cycle-eq")
		c.mu.Lock()
		c.pinnedMode = diModeHeadphones
		c.pinnedUntil = time.Now().Add(5 * time.Second)
		c.mu.Unlock()
	case diHitHpToggle:
		c.mu.Lock()
		c.hpListOpen = !c.hpListOpen
		c.mu.Unlock()
	case diHitSeek:
		if dur > 0 {
			wailsRuntime.EventsEmit(ctx, "island:seek", ratio*dur)
		}
	case diHitLyric:
		if lyricSec >= 0 {
			wailsRuntime.EventsEmit(ctx, "island:seek", lyricSec)
		}
	case diHitArt, diHitQueueMore0, diHitQueueMore1, diHitQueueMore2:
		wailsRuntime.WindowShow(ctx)
		wailsRuntime.WindowUnminimise(ctx)
	case diHitQueueRow:
		c.mu.Lock()
		targetIdx := lyrIdx
		qIdx := c.queueIndex
		qLen := len(c.queueItems)
		c.mu.Unlock()
		if targetIdx >= 0 && targetIdx < qLen {
			if targetIdx == qIdx {
				wailsRuntime.EventsEmit(ctx, "taskbar:control", "toggle")
			} else {
				wailsRuntime.EventsEmit(ctx, "island:play-queue-index", targetIdx)
			}
		}
	case diHitQueueFav:
		c.mu.Lock()
		targetIdx := lyrIdx
		var tid int64
		if targetIdx >= 0 && targetIdx < len(c.queueItems) {
			tid = c.queueItems[targetIdx].ID
		}
		c.mu.Unlock()
		if tid > 0 {
			wailsRuntime.EventsEmit(ctx, "island:toggle-favorite-id", tid)
		}
	case diHitQueueRow0, diHitQueueRow1, diHitQueueRow2:
		row := zone - diHitQueueRow0
		c.mu.Lock()
		targetIdx := c.queueScroll + row
		qIdx := c.queueIndex
		qLen := len(c.queueItems)
		c.mu.Unlock()
		if targetIdx >= 0 && targetIdx < qLen {
			if targetIdx == qIdx {
				wailsRuntime.EventsEmit(ctx, "taskbar:control", "toggle")
			} else {
				wailsRuntime.EventsEmit(ctx, "island:play-queue-index", targetIdx)
			}
		}
	case diHitQueueFav0, diHitQueueFav1, diHitQueueFav2:
		row := zone - diHitQueueFav0
		c.mu.Lock()
		targetIdx := c.queueScroll + row
		var tid int64
		if targetIdx >= 0 && targetIdx < len(c.queueItems) {
			tid = c.queueItems[targetIdx].ID
		}
		c.mu.Unlock()
		if tid > 0 {
			wailsRuntime.EventsEmit(ctx, "island:toggle-favorite-id", tid)
		}
	}
}

func diOtherAudioIndices(outputs []windowsAudioEndpoint) []int {
	var res []int
	for i, d := range outputs {
		if !d.IsActive {
			res = append(res, i)
			if len(res) >= 3 {
				break
			}
		}
	}
	return res
}

func (c *dynamicIslandController) hitTestLocked(mxPhys, myPhys int) (int, float64, int) {
	u := c.dpiScale
	if c.sizeScale > 0 {
		u *= c.sizeScale
	}
	if u <= 0 {
		u = 1.0
	}
	mx := float64(mxPhys) / u
	my := float64(myPhys) / u

	w := c.curW
	h := c.curH
	nx := (float64(diLogicalW)-w)/2.0 - c.liquidSkewDp*0.35*(1.0-math.Max(c.dockLeftProg, c.dockRightProg))
	if nx < 0 {
		nx = 0
	} else if nx > float64(diLogicalW)-w {
		nx = float64(diLogicalW) - w
	}
	rx := mx - nx
	ry := my

	if rx < 0 || rx > w || ry < 0 || ry > h {
		return diHitNone, 0, -1
	}

	mode := c.activeMode
	if mode == diModeMusic || mode == diModeLyrics || mode == diModeQueue || mode == diModeHeadphones {
		leftLim := 20.0
		if c.edgeLeftProg > 0.15 {
			leftLim = 38.0
		}
		rightLim := w - 18.0
		if c.edgeRightProg > 0.15 {
			rightLim = w - 38.0
		}
		if rx >= 0 && rx <= leftLim && ry >= 44 && ry <= h-22 {
			return diHitEdgeLeft, 0, -1
		}
		if rx >= rightLim && rx <= w && ry >= 44 && ry <= h-22 {
			return diHitEdgeRight, 0, -1
		}
	}
	if mode == diModeHeadphones {
		if math.Hypot(rx-(w-76), ry-18) <= 11 {
			return diHitWifi, 0, -1
		}
		if math.Hypot(rx-(w-52), ry-18) <= 11 {
			return diHitGps, 0, -1
		}
		if math.Hypot(rx-(w-28), ry-18) <= 11 {
			return diHitBt, 0, -1
		}
		dotY := h - 11.0
		if ry >= dotY-10 && ry <= dotY+10 && rx >= 186 && rx <= 234 {
			rel := rx - 192
			idx := int(rel / 12.0)
			if idx < 0 {
				idx = 0
			} else if idx > 2 {
				idx = 2
			}
			return diHitDot0 + idx, 0, -1
		}
		if rx >= 18 && rx <= 402 && ry >= 32 && ry <= 84 {
			return diHitHpToggle, 0, -1
		}
		if c.hpListOpen {
			others := diOtherAudioIndices(c.audioOutputs)
			for i := 0; i < len(others); i++ {
				cy := 68.0 + float64(i+1)*56.0
				if ry >= cy-26.0 && ry <= cy+26.0 && rx >= 18 && rx <= 402 {
					return diHitAudioDev0 + i, 0, -1
				}
			}
		}
		return diHitBody, 0, -1
	}

	if mode == diModeQueue {
		if math.Hypot(rx-(w-76), ry-18) <= 11 {
			return diHitWifi, 0, -1
		}
		if math.Hypot(rx-(w-52), ry-18) <= 11 {
			return diHitGps, 0, -1
		}
		if math.Hypot(rx-(w-28), ry-18) <= 11 {
			return diHitBt, 0, -1
		}
		if ry >= 248 && ry <= 272 && rx >= 186 && rx <= 234 {
			rel := rx - 192
			idx := int(rel / 12.0)
			if idx < 0 {
				idx = 0
			} else if idx > 2 {
				idx = 2
			}
			return diHitDot0 + idx, 0, -1
		}
		if math.Hypot(rx-38, ry-54) <= 17 {
			return diHitShuffle, 0, -1
		}
		if math.Hypot(rx-375.5, ry-54) <= 17 {
			return diHitPlay, 0, -1
		}
		if len(c.queueItems) > 3 && rx >= 393 && rx <= 402 && ry >= 78 && ry <= 242 {
			return diHitQueueScroll, 0, -1
		}
		if ry >= 78 && ry <= 242 {
			virtualY := ry + c.queueScrollDp - 80.0
			if virtualY >= 0 {
				itemIdx := int(virtualY / 54.0)
				withinRow := virtualY - float64(itemIdx)*54.0
				if itemIdx >= 0 && itemIdx < len(c.queueItems) && withinRow <= 48.0 {
					if math.Hypot(rx-372, withinRow-24.0) <= 18 {
						return diHitQueueFav, 0, itemIdx
					}
					if rx >= 18 && rx <= 354 {
						return diHitQueueRow, 0, itemIdx
					}
				}
			}
		}
		return diHitBody, 0, -1
	}

	if mode != diModeMusic && mode != diModeLyrics {
		return diHitBody, 0, -1
	}

	if rx >= 20 && rx <= 70 && ry >= 31 && ry <= 81 {
		return diHitArt, 0, -1
	}

	if mode == diModeMusic && ry >= 157 && ry <= 180 && rx >= 186 && rx <= 234 {
		rel := rx - 192
		idx := int(rel / 12.0)
		if idx < 0 {
			idx = 0
		} else if idx > 2 {
			idx = 2
		}
		return diHitDot0 + idx, 0, -1
	}
	if mode == diModeLyrics && ry >= 157 && ry <= 180 && rx >= 108 && rx <= 156 {
		rel := rx - 114
		idx := int(rel / 12.0)
		if idx < 0 {
			idx = 0
		} else if idx > 2 {
			idx = 2
		}
		return diHitDot0 + idx, 0, -1
	}

	if mode == diModeMusic {
		if math.Hypot(rx-344, ry-18) <= 11 {
			return diHitWifi, 0, -1
		}
		if math.Hypot(rx-368, ry-18) <= 11 {
			return diHitGps, 0, -1
		}
		if math.Hypot(rx-392, ry-18) <= 11 {
			return diHitBt, 0, -1
		}
		if math.Hypot(rx-342, ry-56) <= 17 {
			return diHitFav, 0, -1
		}
		if math.Hypot(rx-383, ry-56) <= 17 {
			return diHitArrow, 0, -1
		}
		if ry >= 88 && ry <= 108 && rx >= 22 && rx <= 398 {
			return diHitSeek, clamp01((rx - 22) / 376.0), -1
		}
		if ry >= 114 && ry <= 152 {
			if math.Hypot(rx-41, ry-134) <= 19 {
				return diHitBubble, 0, -1
			}
			if math.Hypot(rx-162, ry-134) <= 19 {
				return diHitPrev, 0, -1
			}
			if math.Hypot(rx-210, ry-134) <= 20 {
				return diHitPlay, 0, -1
			}
			if math.Hypot(rx-258, ry-134) <= 19 {
				return diHitNext, 0, -1
			}
			if math.Hypot(rx-379, ry-134) <= 19 {
				return diHitQueueOpen, 0, -1
			}
		}
	} else if mode == diModeLyrics {
		if math.Hypot(rx-(w-76), ry-18) <= 11 {
			return diHitWifi, 0, -1
		}
		if math.Hypot(rx-(w-52), ry-18) <= 11 {
			return diHitGps, 0, -1
		}
		if math.Hypot(rx-(w-28), ry-18) <= 11 {
			return diHitBt, 0, -1
		}
		if ry >= 88 && ry <= 108 && rx >= 22 && rx <= 236 {
			return diHitSeek, clamp01((rx - 22) / 214.0), -1
		}
		if ry >= 114 && ry <= 152 {
			if math.Hypot(rx-41, ry-134) <= 16 {
				return diHitBubble, 0, -1
			}
			if math.Hypot(rx-96, ry-134) <= 16 {
				return diHitPrev, 0, -1
			}
			if math.Hypot(rx-138, ry-134) <= 17 {
				return diHitPlay, 0, -1
			}
			if math.Hypot(rx-180, ry-134) <= 16 {
				return diHitNext, 0, -1
			}
			if math.Hypot(rx-222, ry-134) <= 16 {
				return diHitFav, 0, -1
			}
		}
		if rx >= 254 && rx <= 502 && ry >= 39 && ry <= 146 && len(c.lyrics) > 0 {
			activeIdx := c.currentLyricIndexLocked()
			if activeIdx < 0 {
				activeIdx = 0
			}
			lineOffset := int(math.Round((ry + c.lyricManualOffsetDp - 92.0) / 28.0))
			targetIdx := activeIdx + lineOffset
			if targetIdx >= 0 && targetIdx < len(c.lyrics) {
				return diHitLyric, 0, targetIdx
			}
		}
	}

	return diHitBody, 0, -1
}

func (c *dynamicIslandController) effectivePosLocked(now time.Time) float64 {
	pos := c.posSec
	if c.isPlaying && !c.posUpdatedAt.IsZero() {
		elapsed := now.Sub(c.posUpdatedAt).Seconds()
		if elapsed > 0 {
			if elapsed > 0.35 {
				elapsed = 0.35
			}
			pos += elapsed
		}
	}
	if c.durSec > 0 && pos > c.durSec {
		pos = c.durSec
	}
	return pos
}

func (c *dynamicIslandController) currentLyricIndexAtLocked(now time.Time) int {
	if len(c.lyrics) == 0 {
		return -1
	}
	effPos := c.effectivePosLocked(now) + 0.05 + c.lyricOffsetSec
	idx := -1
	for i, l := range c.lyrics {
		if effPos >= l.TimeSec {
			idx = i
		} else {
			break
		}
	}
	return idx
}

func (c *dynamicIslandController) currentLyricIndexLocked() int {
	return c.currentLyricIndexAtLocked(time.Now())
}

func (c *dynamicIslandController) isClosedLyricActiveLocked() bool {
	return c.isPlaying && c.showLyric && (c.lyricsOpen || c.compactStyle == "SATU_BARIS") && c.title != ""
}

func (c *dynamicIslandController) isLyricPlaybackModeActiveLocked() bool {
	return c.isPlaying && c.showLyric && (c.lyricsOpen || c.compactStyle == "SATU_BARIS") && c.title != ""
}

func (c *dynamicIslandController) closedLyricMessageLocked(now time.Time, diStr diLocalizedStrings) string {
	if len(c.lyrics) > 0 {
		act := c.currentLyricIndexAtLocked(now)
		if act >= 0 && act < len(c.lyrics) {
			return strings.TrimSpace(c.lyrics[act].Text)
		}
		return "♪  • • •"
	}
	if c.lyricsLoading {
		return diStr.LoadingLyrics
	}
	if c.lyricsHasPlain {
		return diStr.NoSyncLyrics
	}
	return diStr.NoLyrics
}

func (c *dynamicIslandController) resolveTargetModeLocked() string {
	if c.winDragActive {
		return diModeClosed
	}
	now := time.Now()
	if c.pinnedMode != "" && now.Before(c.pinnedUntil) {
		return c.pinnedMode
	}
	c.pinnedMode = ""
	if c.hovered {
		if c.queueOpen {
			return diModeQueue
		}
		if c.headphonesOpen {
			return diModeHeadphones
		}
		if c.lyricsOpen {
			return diModeLyrics
		}
		return diModeMusic
	}
	if c.onlyPlaying && !c.isPlaying {
		return diModeFullscreenHide
	}
	if !c.onlyPlaying && c.pauseHideSec > 0 && !c.isPlaying && c.title != "" && !c.pausedAt.IsZero() && now.Sub(c.pausedAt) >= time.Duration(c.pauseHideSec)*time.Second {
		return diModeFullscreenHide
	}
	if !c.isLyricPlaybackModeActiveLocked() {
		if c.idleHideSec > 0 && !c.lastActivityAt.IsZero() && now.Sub(c.lastActivityAt) >= time.Duration(c.idleHideSec)*time.Second {
			return diModeFullscreenHide
		}
		if c.compactStyle == "TERSEMBUNYI" {
			return diModeFullscreenHide
		}
	}
	return diModeClosed
}

func (c *dynamicIslandController) measureTextWidthDp(fam uintptr, txt string, heightDp float64) float64 {
	if fam == 0 || txt == "" || heightDp <= 0 {
		return 0
	}
	var path uintptr
	st, _, _ := diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&path)))
	if st != 0 || path == 0 {
		return float64(len([]rune(txt))) * heightDp * 0.55
	}
	defer diProcGdipDeletePath.Call(path)

	u16, _ := windows.UTF16FromString(txt)
	if len(u16) <= 1 {
		return 0
	}
	rcOrigin := diRectF{X: 0, Y: 0, Width: 0, Height: 0}
	_, _, _ = diProcGdipAddPathString.Call(
		path,
		uintptr(unsafe.Pointer(&u16[0])),
		uintptr(len(u16)-1),
		fam,
		0,
		uintptr(math.Float32bits(100.0)),
		uintptr(unsafe.Pointer(&rcOrigin)),
		0,
	)
	var bounds diRectF
	stB, _, _ := diProcGdipGetPathWorldBounds.Call(path, uintptr(unsafe.Pointer(&bounds)), 0, 0)
	if stB != 0 || bounds.Height <= 0.1 {
		return float64(len([]rune(txt))) * heightDp * 0.55
	}
	return float64(bounds.Width) * (heightDp / float64(bounds.Height))
}

func (c *dynamicIslandController) resolveClosedWidthLocked() float64 {
	if c.isClosedLyricActiveLocked() {
		lang := c.lang
		if lang == "" {
			lang = detectSystemLanguage()
		}
		txt := c.closedLyricMessageLocked(time.Now(), getDIStrings(lang))
		if txt != "" {
			u := c.effectiveScaleLocked()
			tw := 0.0
			if dwW, ok := c.measureClosedLyricWidthDp(txt, u); ok {
				tw = dwW
			} else {
				fam := c.famSemiBold
				if fam == 0 {
					fam = c.famRegular
				}
				tw = c.measureTextWidthDp(fam, txt, 14.5)
			}
			w := tw + 96.0
			if w < 216.0 {
				w = 216.0
			} else if w > 560.0 {
				w = 560.0
			}
			return w
		}
	}
	return 216.0
}

func (c *dynamicIslandController) tickAndRender() {
	showAllowed := shouldDynamicIslandShow(c.hwnd)
	fgNow, _, _ := diProcGetForegroundWindow.Call()
	transientSwitcher := isTransientTaskSwitcher(fgNow)
	isFullscreen := isForegroundFullscreen(c.hwnd)
	now := time.Now()

	c.mu.Lock()
	if showAllowed && !c.hwPolling && (c.lastHwPoll.IsZero() || now.Sub(c.lastHwPoll) >= 2500*time.Millisecond) {
		go c.pollAudioHardware(true)
	}
	c.wasMinimized = showAllowed
	if !transientSwitcher {
		c.fullscreenActive = isFullscreen
	}
	if c.isPlaying {
		c.pausedAt = time.Time{}
	} else if c.wasPlaying || c.pausedAt.IsZero() {
		c.pausedAt = now
	}
	c.wasPlaying = c.isPlaying
	if c.hovered || c.winDragActive || c.holdPending || c.pinnedMode != "" {
		c.lastActivityAt = now
	}
	targetMode := c.resolveTargetModeLocked()
	hiddenByFullscreen := c.hideInFullscreen && c.fullscreenActive
	if (!showAllowed || hiddenByFullscreen) && c.activeMode == diModeFullscreenHide {
		targetMode = diModeFullscreenHide
	}
	shouldAppear := c.enabled && showAllowed && !hiddenByFullscreen
	if targetMode != c.activeMode {
		tw, th, tr, te := nusaModeDims(targetMode)
		if targetMode == diModeClosed {
			tw = c.resolveClosedWidthLocked()
			c.closedWFrom = tw
			c.closedWTo = tw
			c.closedWStart = now
		} else if targetMode == diModeHeadphones {
			if c.hpListOpen {
				otherCnt := len(diOtherAudioIndices(c.audioOutputs))
				if otherCnt > 0 {
					th = 118.0 + float64(otherCnt)*56.0
				} else {
					th = 106.0
				}
			} else {
				th = 106.0
			}
		}
		oldW, oldH, _, _ := nusaModeDims(c.activeMode)
		if c.activeMode == diModeClosed || c.activeMode == diModeResting || targetMode == diModeClosed || targetMode == diModeResting {
			c.pageSlideDir = 0
		}
		c.shrinking = (tw * th) < (oldW * oldH)
		c.fromW, c.fromH, c.fromR, c.fromE = c.curW, c.curH, c.curR, c.curE
		c.toW, c.toH, c.toR, c.toE = tw, th, tr, te
		c.activeMode = targetMode
		c.transStart = now
		if targetMode == diModeMusic || targetMode == diModeLyrics {
			c.marqueeStart = now
		}
	}
	if c.activeMode == diModeClosed || c.activeMode == diModeResting {
		c.marqueeStart = now
	}
	spd := c.animSpeedScale
	if spd <= 0 {
		spd = 1.0
	}
	dt := 0.016
	if !c.lastTickAt.IsZero() {
		dt = now.Sub(c.lastTickAt).Seconds()
		if dt < 0.004 {
			dt = 0.004
		} else if dt > 0.05 {
			dt = 0.05
		}
	}
	c.lastTickAt = now
	if c.isPlaying {
		const spinOmega = (2.0 * math.Pi) / 9.0
		if c.coverStyle == "circle" && c.coverSpin {
			c.coverSpinRad = math.Mod(c.coverSpinRad+dt*spinOmega, 2.0*math.Pi)
		}
		if c.closedCoverSpin {
			c.closedSpinRad = math.Mod(c.closedSpinRad+dt*spinOmega, 2.0*math.Pi)
		}
	}

	if targetMode == diModeHeadphones {
		wantH := 106.0
		if c.hpListOpen {
			otherCnt := len(diOtherAudioIndices(c.audioOutputs))
			if otherCnt > 0 {
				wantH = 118.0 + float64(otherCnt)*56.0
			}
		}
		if math.Abs(wantH-c.toH) > 0.5 {
			c.fromW, c.fromH, c.fromR, c.fromE = c.curW, c.curH, c.curR, c.curE
			c.toH = wantH
			c.shrinking = wantH < c.curH
			if now.Sub(c.transStart).Seconds() > 0.08*spd {
				c.pageSlideDir = 0
				c.transStart = now
			}
		}
	}
	hpStep := dt / (0.22 * spd)
	if c.activeMode == diModeHeadphones && c.hpListOpen {
		c.hpListProg = math.Min(1.0, c.hpListProg+hpStep)
	} else {
		c.hpListProg = math.Max(0.0, c.hpListProg-hpStep*1.25)
	}

	el := now.Sub(c.transStart).Seconds()
	var whEase float64
	var reEase float64
	if c.shrinking {
		whEase = nusaShrinkEase(clamp01(el / (0.48 * spd)))
		reEase = whEase
	} else {
		whEase = nusaSpring(clamp01(el / (0.62 * spd)))
		reEase = nusaCssEase(clamp01(el / (0.45 * spd)))
	}

	c.curW = c.fromW + (c.toW-c.fromW)*whEase
	if targetMode == diModeClosed && (c.fromH <= 40.0 || el >= 0.30*spd) {
		dynW := c.resolveClosedWidthLocked()
		if math.Abs(dynW-c.closedWTo) > 0.5 {
			c.closedWFrom = c.curW
			c.closedWTo = dynW
			c.closedWStart = now
		}
		if c.closedWTo > 0 {
			wDur := 0.28 * spd
			if c.closedWTo > c.closedWFrom {
				wDur = 0.16 * spd
			}
			wProg := nusaCssEase(clamp01(now.Sub(c.closedWStart).Seconds() / wDur))
			c.curW = c.closedWFrom + (c.closedWTo-c.closedWFrom)*wProg
			c.toW = c.closedWTo
		}
	}
	c.curH = c.fromH + (c.toH-c.fromH)*whEase
	c.curR = c.fromR + (c.toR-c.fromR)*reEase
	c.curE = c.fromE + (c.toE-c.fromE)*reEase

	var promoteMoveCur uintptr
	if c.holdPending && !c.winDragActive {
		u := c.dpiScale
	if c.sizeScale > 0 {
		u *= c.sizeScale
	}
		if u <= 0 {
			u = 1.0
		}
		var pt diPoint
		_, _, _ = diProcGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
		if math.Hypot(float64(pt.X-c.holdStartScreenX), float64(pt.Y-c.holdStartScreenY)) > 6.0*u {
			c.holdPending = false
		} else if now.Sub(c.holdStart) >= 180*time.Millisecond {
			c.winDragActive = true
			c.holdPending = false
			c.dragActive = false
			c.lyricDragScroll = false
			promoteMoveCur = c.cursorMove
		}
	}

	uVal := c.dpiScale
	if c.sizeScale > 0 {
		uVal *= c.sizeScale
	}
	if uVal <= 0 {
		uVal = 1.0
	}
	if c.screenMode == "cursor" && !c.winDragActive && !c.hovered {
		monLeft, monTop, monW := resolveMonitorRect(true)
		if monW > 0 && (monLeft != c.screenLeft || monTop != c.screenTop || monW != c.screenW) {
			c.screenLeft = monLeft
			c.screenTop = monTop
			c.screenW = monW
			if c.pillMode {
				c.winY = monTop + int(math.Round(10.0*uVal))
			} else {
				c.winY = monTop
			}
			c.winTargetX = float64(c.centeredWinXLocked(monLeft, monW))
			c.winXFloat = c.winTargetX
			c.dockSide = 0
		}
	}
	nxBaseCur := (float64(diLogicalW) - c.curW) / 2.0
	minEdgeX := c.screenLeft - int(math.Round(nxBaseCur*uVal))
	maxEdgeX := c.screenLeft + c.screenW - int(math.Round((nxBaseCur+c.curW)*uVal))
	if c.dockSide < 0 {
		c.winTargetX = float64(minEdgeX)
	} else if c.dockSide > 0 {
		c.winTargetX = float64(maxEdgeX)
	} else {
		clampedT, _ := c.clampWinXLocked(int(math.Round(c.winTargetX)), false)
		c.winTargetX = float64(clampedT)
	}
	diffX := c.winTargetX - c.winXFloat
	prevWinXFloat := c.winXFloat
	if math.Abs(diffX) < 0.25 {
		c.winXFloat = c.winTargetX
		c.winVelX = 0
	} else {
		alphaPos := 1.0 - math.Exp((-22.0/spd)*dt)
		c.winXFloat += diffX * alphaPos
		c.winVelX = ((c.winXFloat - prevWinXFloat) / dt) * 0.016
	}
	velSignedDp := c.winVelX / uVal
	targetSkew := math.Max(-14.0, math.Min(14.0, velSignedDp*0.50))
	alphaSkew := 1.0 - math.Exp(-18.0*dt)
	c.liquidSkewDp += (targetSkew - c.liquidSkewDp) * alphaSkew
	if math.Abs(c.liquidSkewDp) < 0.05 && math.Abs(targetSkew) < 0.05 {
		c.liquidSkewDp = 0
	}
	absSkew := math.Abs(c.liquidSkewDp)
	if c.winDragActive {
		c.curW += math.Max(3.0, absSkew)
		c.curR += absSkew * 0.25
		c.curE += absSkew * 0.22
	} else if absSkew > 0.1 {
		c.curW += absSkew
		c.curR += absSkew * 0.25
		c.curE += absSkew * 0.22
	}

	if c.hovered && !c.winDragActive && !c.holdPending && c.winShown {
		var ptHover diPoint
		_, _, _ = diProcGetCursorPos.Call(uintptr(unsafe.Pointer(&ptHover)))
		nxH := (float64(diLogicalW) - c.curW) / 2.0
		pad := 10.0 * uVal
		inX := float64(ptHover.X) >= float64(c.winX)+nxH*uVal-pad && float64(ptHover.X) <= float64(c.winX)+(nxH+c.curW)*uVal+pad
		inY := float64(ptHover.Y) >= float64(c.winY)-pad && float64(ptHover.Y) <= float64(c.winY)+c.curH*uVal+pad
		if !inX || !inY {
			c.hovered = false
			c.hoverZone = diHitNone
			c.lastActivityAt = now
			if c.hideInFullscreen && c.fullscreenActive {
				c.pinnedMode = ""
			}
		}
	} else if !c.hovered && !transientSwitcher && c.openTrigger != "click" && c.activeMode == diModeFullscreenHide && c.winShown && !hiddenByFullscreen {
		var ptHover diPoint
		_, _, _ = diProcGetCursorPos.Call(uintptr(unsafe.Pointer(&ptHover)))
		nxH := (float64(diLogicalW) - c.curW) / 2.0
		padX := 12.0 * uVal
		inX := float64(ptHover.X) >= float64(c.winX)+nxH*uVal-padX && float64(ptHover.X) <= float64(c.winX)+(nxH+c.curW)*uVal+padX
		inY := float64(ptHover.Y) >= float64(c.screenTop) && float64(ptHover.Y) <= float64(c.winY)+10.0*uVal
		if inX && inY {
			c.hovered = true
			c.lastActivityAt = now
		}
	}

	nxBasePost := (float64(diLogicalW) - c.curW) / 2.0
	if c.dockSide < 0 {
		c.winXFloat = float64(c.screenLeft - int(math.Round(nxBasePost*uVal)))
	} else if c.dockSide > 0 {
		c.winXFloat = float64(c.screenLeft + c.screenW - int(math.Round((nxBasePost+c.curW)*uVal)))
	}
	newWinX, _ := c.clampWinXLocked(int(math.Round(c.winXFloat)), false)
	if newWinX != c.winX {
		c.winX = newWinX
	}

	if c.pillMode {
		c.dockLeftProg = 0
		c.dockRightProg = 0
	} else {
		leftScreenPx := float64(newWinX-c.screenLeft) + nxBasePost*uVal
		rightGapPx := float64(c.screenLeft+c.screenW) - (float64(newWinX) + (nxBasePost+c.curW)*uVal)
		targetDockL := clamp01(1.0 - leftScreenPx/(8.0*uVal))
		targetDockR := clamp01(1.0 - rightGapPx/(8.0*uVal))
		if c.dockSide < 0 {
			targetDockL = 1.0
		} else if c.dockSide > 0 {
			targetDockR = 1.0
		}
		dockAlpha := 1.0 - math.Exp(-dt*20.0)
		c.dockLeftProg += (targetDockL - c.dockLeftProg) * dockAlpha
		c.dockRightProg += (targetDockR - c.dockRightProg) * dockAlpha
	}

	if shouldAppear {
		c.appearProg = math.Min(1.0, c.appearProg+dt/(0.26*spd))
	} else {
		hideDur := 0.20 * spd
		if c.activeMode == diModeFullscreenHide {
			hideDur = 0.28 * spd
		}
		c.appearProg = math.Max(0.0, c.appearProg-dt/hideDur)
	}

	if c.activeMode == diModeFullscreenHide && c.curH <= 10.0 {
		c.hideLineProg = math.Min(1.0, c.hideLineProg+dt/(0.22*spd))
	} else {
		c.hideLineProg = math.Max(0.0, c.hideLineProg-dt/(0.18*spd))
	}

	if c.activeMode == diModeLyrics {
		actIdx := c.currentLyricIndexLocked()
		if actIdx != c.lyricIdx {
			c.prevLyricIdx = c.lyricIdx
			c.lyricIdx = actIdx
			c.lyricFromY = c.lyricCurY
			if actIdx >= 0 {
				c.lyricToY = float64(actIdx) * 29.0
			} else {
				c.lyricToY = 0
			}
			c.lyricStart = now
		}
		lel := now.Sub(c.lyricStart).Seconds()
		c.lyricAnimProg = nusaCssEase(clamp01(lel / 0.18))
		c.lyricCurY = c.lyricFromY + (c.lyricToY-c.lyricFromY)*c.lyricAnimProg
		if !c.lyricDragScroll && !c.lyricManualUntil.IsZero() && now.After(c.lyricManualUntil) {
			c.lyricTargetOffsetDp = 0
			c.lyricManualUntil = time.Time{}
		}
		lDiff := c.lyricTargetOffsetDp - c.lyricManualOffsetDp
		if math.Abs(lDiff) < 0.05 {
			c.lyricManualOffsetDp = c.lyricTargetOffsetDp
		} else {
			lAlpha := 1.0 - math.Exp(-dt*18.0)
			c.lyricManualOffsetDp += lDiff * lAlpha
		}
	} else if c.activeMode == diModeClosed {
		curClosedTxt := ""
		if c.isClosedLyricActiveLocked() {
			lCode := c.lang
			if lCode == "" {
				lCode = detectSystemLanguage()
			}
			curClosedTxt = c.closedLyricMessageLocked(now, getDIStrings(lCode))
		}
		if curClosedTxt != c.closedLyricTo {
			c.closedLyricFrom = c.closedLyricTo
			c.closedLyricTo = curClosedTxt
			c.closedLyricStart = now
		}
		cel := now.Sub(c.closedLyricStart).Seconds()
		c.closedLyricProg = nusaCssEase(clamp01(cel / 0.16))
	} else if c.activeMode == diModeQueue {
		diff := c.queueTargetScrollDp - c.queueScrollDp
		if math.Abs(diff) < 0.05 {
			c.queueScrollDp = c.queueTargetScrollDp
		} else {
			qAlpha := 1.0 - math.Exp(-dt*18.0)
			c.queueScrollDp += diff * qAlpha
		}
	}

	targetEdgeL := 0.0
	targetEdgeR := 0.0
	if c.hovered && (c.activeMode == diModeMusic || c.activeMode == diModeLyrics || c.activeMode == diModeQueue || c.activeMode == diModeHeadphones) {
		if c.hoverZone == diHitEdgeLeft {
			targetEdgeL = 1.0
		} else if c.hoverZone == diHitEdgeRight {
			targetEdgeR = 1.0
		}
	}
	edgeAlpha := 1.0 - math.Exp(-dt*18.0)
	if math.Abs(targetEdgeL-c.edgeLeftProg) < 0.01 {
		c.edgeLeftProg = targetEdgeL
	} else {
		c.edgeLeftProg += (targetEdgeL - c.edgeLeftProg) * edgeAlpha
	}
	if math.Abs(targetEdgeR-c.edgeRightProg) < 0.01 {
		c.edgeRightProg = targetEdgeR
	} else {
		c.edgeRightProg += (targetEdgeR - c.edgeRightProg) * edgeAlpha
	}

	winShown := c.winShown
	appearProg := c.appearProg
	physW := c.physW
	physH := c.physH
	opacityPct := c.opacityPct
	if opacityPct < 0 {
		opacityPct = 0
	} else if opacityPct > 100 {
		opacityPct = 100
	}
	curWinX := c.winX
	curWinY := c.winY

	isAnimating := now.Sub(c.transStart).Seconds() < 0.72*spd ||
		now.Sub(c.closedLyricStart).Seconds() < 0.24 ||
		now.Sub(c.closedWStart).Seconds() < 0.34*spd ||
		now.Sub(c.lyricStart).Seconds() < 0.24 ||
		c.hovered ||
		c.winDragActive ||
		c.seekDragging ||
		c.queueDragScroll ||
		c.lyricDragScroll ||
		math.Abs(c.winXFloat-c.winTargetX) > 0.5 ||
		math.Abs(c.winVelX) > 0.5 ||
		(c.isPlaying && (c.closedCoverSpin || c.coverSpin)) ||
		(c.isPlaying && c.lyricSweep && len(c.lyrics) > 0 && (c.activeMode == diModeLyrics || c.isClosedLyricActiveLocked())) ||
		(c.activeMode == diModeHeadphones && c.hpListProg > 0.01 && c.hpListProg < 0.99) ||
		(c.appearProg > 0.01 && c.appearProg < 0.99)
	desiredTimerMs := uint32(250)
	if appearProg <= 0.01 && !winShown {
		desiredTimerMs = 250
	} else if isAnimating {
		desiredTimerMs = 15
	} else if c.isPlaying {
		desiredTimerMs = 120
	}
	updateTimer := desiredTimerMs != c.curTimerMs
	if updateTimer {
		c.curTimerMs = desiredTimerMs
	}
	c.mu.Unlock()

	if updateTimer && c.hwnd != 0 {
		_, _, _ = diProcSetTimer.Call(c.hwnd, diTimerID, uintptr(desiredTimerMs), 0)
	}

	if promoteMoveCur != 0 {
		_, _, _ = diProcSetCursor.Call(promoteMoveCur)
	}

	if appearProg > 0.01 && !winShown {
		_, _, _ = diProcSetWindowPos.Call(c.hwnd, ^uintptr(0), uintptr(curWinX), uintptr(curWinY), uintptr(physW), uintptr(physH), 0x0010|0x0040)
		_, _, _ = diProcShowWindow.Call(c.hwnd, diSwShowNoActivate)
		c.mu.Lock()
		c.winShown = true
		c.mu.Unlock()
	} else if appearProg <= 0.01 && winShown {
		if c.dibBits != nil && physW > 0 && physH > 0 {
			pix := unsafe.Slice((*byte)(c.dibBits), physW*physH*4)
			clear(pix)
		}
		_, _, _ = diProcShowWindow.Call(c.hwnd, diSwHide)
		c.mu.Lock()
		c.winShown = false
		c.hovered = false
		c.hoverZone = diHitNone
		c.mu.Unlock()
		return
	}

	if appearProg > 0.01 {
		c.renderFrame(now)
	}
}

func (c *dynamicIslandController) renderFrame(now time.Time) {
	if c.memDC == 0 || c.dibBits == nil {
		return
	}

	c.mu.Lock()
	u := c.dpiScale
	if c.sizeScale > 0 {
		u *= c.sizeScale
	}
	physW := c.physW
	physH := c.physH
	opacityPct := c.opacityPct
	if opacityPct < 0 {
		opacityPct = 0
	} else if opacityPct > 100 {
		opacityPct = 100
	}
	appear := clamp01(c.appearProg)
	hideLineProg := c.hideLineProg
	mode := c.activeMode
	w := c.curW
	easeAppear := nusaCssEase(appear)
	h := c.curH * easeAppear
	r := math.Min(c.curR, h)
	e := c.curE * clamp01(h/math.Max(1.0, c.curH))
	if h < 1.5 {
		e = 0
	}
	spd := c.animSpeedScale
	if spd <= 0 {
		spd = 1.0
	}
	transEl := now.Sub(c.transStart).Seconds()
	title := c.title
	artist := c.artist
	album := c.album
	playing := c.isPlaying
	trackLoading := c.trackLoading
	isFav := c.isFavorite
	pos := c.effectivePosLocked(now)
	dur := c.durSec
	vol := c.volume
	muted := c.muted
	eqPreset := c.eqPreset
	streakDays := c.streakDays
	lyricsLoading := c.lyricsLoading
	lyricsHasPlain := c.lyricsHasPlain
	lyricsOpen := c.lyricsOpen
	audioDev := c.audioDeviceName
	connType := c.audioConnType
	btBat := c.btBatteryPct
	sysBat := c.sysBatteryPct
	isBt := c.isBtAudio
	audioOutputs := c.audioOutputs
	hpListOpen := c.hpListOpen
	hpListProg := c.hpListProg
	hoverZone := c.hoverZone
	hoverRatio := c.hoverRatio
	hoverLyric := c.hoverLyric
	coverImg := c.coverImg
	coverImgCircle := c.coverImgCircle
	coverImgMini := c.coverImgMini
	coverStyle := c.coverStyle
	coverSpin := c.coverSpin
	closedCoverSpin := c.closedCoverSpin
	coverSpinRad := c.coverSpinRad
	closedSpinRad := c.closedSpinRad
	lazyCoverURL := ""
	if coverImg == nil && c.coverURL != "" && !c.coverLoading && now.Sub(c.lastCoverRetry) > 700*time.Millisecond {
		c.coverLoading = true
		c.lastCoverRetry = now
		lazyCoverURL = c.coverURL
	}
	defaultArt := c.defaultArt
	defaultArtCirc := c.defaultArtCirc
	defaultArtSm := c.defaultArtSm
	defaultArtMini := c.defaultArtMini
	lyrics := c.lyrics
	lyricIdx := c.lyricIdx
	if mode == diModeClosed || mode == diModeResting {
		lyricIdx = c.currentLyricIndexAtLocked(now)
	}
	effLyricPos := pos + c.lyricOffsetSec
	lyricSweep := c.lyricSweep
	prevLyricIdx := c.prevLyricIdx
	lyricAnimProg := c.lyricAnimProg
	lyricCurY := c.lyricCurY
	lyricManualOffsetDp := c.lyricManualOffsetDp
	closedLyricFrom := c.closedLyricFrom
	closedLyricTo := c.closedLyricTo
	closedLyricProg := c.closedLyricProg
	qTitle := c.queueTitle
	qIndex := c.queueIndex
	qScrollDp := c.queueScrollDp
	qScrolling := c.queueDragScroll || math.Abs(c.queueTargetScrollDp-c.queueScrollDp) > 0.05 || c.lyricDragScroll || math.Abs(c.lyricTargetOffsetDp-c.lyricManualOffsetDp) > 0.05
	qItems := c.queueItems
	diLang := c.lang
	if diLang == "" {
		diLang = detectSystemLanguage()
	}
	hoverQueueIdx := c.hoverQueueIdx
	queueDragScroll := c.queueDragScroll
	shuffle := c.shuffle
	wifiOn := c.wifiOn
	gpsOn := c.gpsOn
	btOn := c.btOn
	edgeLeftProg := c.edgeLeftProg
	edgeRightProg := c.edgeRightProg
	pageSlideDir := c.pageSlideDir
	liquidSkewDp := c.liquidSkewDp
	dockLeftProg := c.dockLeftProg
	dockRightProg := c.dockRightProg
	curPage := c.currentPageIndexLocked()
	curWinX := c.winX
	curWinY := c.winY
	pillMode := c.pillMode
	showStatus := c.showStatus
	showCover := c.showCover
	showClosedCover := c.showClosedCover
	showLyric := c.showLyric
	showHeart := c.showHeart
	showDots := c.showDots
	showProgress := c.showProgress
	showClosedProgress := c.showClosedProgress
	progressStyle := c.progressStyle
	if pillMode && easeAppear < 0.999 {
		slideUp := (1.0 - easeAppear) * (float64(c.winY) + c.curH*u*0.55)
		curWinY = int(math.Round(float64(c.winY) - slideUp))
	}
	pal := c.resolvePaletteLocked()
	c.mu.Unlock()

	if lazyCoverURL != "" {
		go c.loadCoverAsync(lazyCoverURL)
	}

	hasTrack := title != ""
	if !hasTrack {
		title = "Velora Music"
		artist = "Putar lagu di Velora untuk memulai"
	} else if album != "" && !strings.EqualFold(album, "YouTube Music") && !strings.Contains(artist, album) {
		artist = artist + " · " + album
	}

	contentOp := nusaCssEase(clamp01((transEl - 0.10*spd) / (0.55 * spd)))
	contentPop := nusaContentPop(clamp01((transEl - 0.05*spd) / (0.50 * spd)))
	contentScaleX := 0.86 + 0.14*contentPop
	contentScaleY := 0.90 + 0.10*contentPop
	pageShift := 0.0
	if pageSlideDir != 0 && mode != diModeResting && mode != diModeClosed {
		slideEase := nusaSpring(clamp01(transEl / (0.44 * spd)))
		contentOp = nusaCssEase(clamp01(transEl / (0.24 * spd)))
		contentScaleX = 0.96 + 0.04*slideEase
		contentScaleY = 0.97 + 0.03*slideEase
		pageShift = float64(pageSlideDir) * (1.0 - slideEase) * 56.0
	}
	if mode == diModeResting || mode == diModeClosed {
		contentOp = 1.0
		contentPop = 1.0
		contentScaleX = 1.0
		contentScaleY = 1.0
		pageShift = 0.0
	}

	nx := (float64(diLogicalW)-w)/2.0 - liquidSkewDp*0.35*(1.0-math.Max(dockLeftProg, dockRightProg))
	if nx < 0 {
		nx = 0
	} else if nx > float64(diLogicalW)-w {
		nx = float64(diLogicalW) - w
	}

	bufLen := physW * physH * 4
	pix := unsafe.Slice((*byte)(c.dibBits), bufLen)
	clear(pix)

	if (mode != diModeFullscreenHide && h < 1.0) || easeAppear < 0.01 {
		sizeDst := diSize{Cx: int32(physW), Cy: int32(physH)}
		ptSrc := diPoint{}
		blend := diBlendFunction{
			BlendOp:             diAcSrcOver,
			SourceConstantAlpha: 255,
			AlphaFormat:         diAcSrcAlpha,
		}
		_, _, _ = diProcUpdateLayeredWindow.Call(
			c.hwnd, 0, 0, uintptr(unsafe.Pointer(&sizeDst)),
			c.memDC, uintptr(unsafe.Pointer(&ptSrc)), 0,
			uintptr(unsafe.Pointer(&blend)), diUlwAlpha,
		)
		return
	}

	rL := r * (1.0 - dockLeftProg)
	eL := e * (1.0 - dockLeftProg)
	eBotL := e * dockLeftProg
	rR := r * (1.0 - dockRightProg)
	eR := e * (1.0 - dockRightProg)
	eBotR := e * dockRightProg

	if mode == diModeFullscreenHide {
		maxAlpha := 255.0 * clamp01(float64(opacityPct)/100.0) * easeAppear
		if h > 10.0 {
			drawVeloraNotchShapePhys(pix, physW, physH, nx*u, w*u, h*u, rL*u, eL*u, eBotL*u, rR*u, eR*u, eBotR*u, pillMode, pal)
			maxAlpha *= clamp01((h - 10.0) / 22.0)
		} else {
			inEase := nusaCssEase(clamp01(hideLineProg))
			visEase := inEase * easeAppear
			lineOp := visEase * 0.90
			lineW := w * (0.18 + 0.82*visEase)
			lineX := nx + (w-lineW)/2.0
			barH := math.Max(0.8, 3.5*visEase) * u
			barY := 1.5 * visEase * u
			if lineOp > 0.005 {
				var gBar uintptr
				_, _, _ = diProcGdipCreateFromHDC.Call(c.memDC, uintptr(unsafe.Pointer(&gBar)))
				if gBar != 0 {
					_, _, _ = diProcGdipSetSmoothingMode.Call(gBar, 2)
					_, _, _ = diProcGdipSetPixelOffsetMode.Call(gBar, 2)
					drawFilledPillGdip(gBar, lineX*u, barY, lineW*u, barH, barH/2.0, rgbaToARGB(pal.Accent, lineOp))
					_, _, _ = diProcGdipDeleteGraphics.Call(gBar)
				}
			}
		}
		sizeDst := diSize{Cx: int32(physW), Cy: int32(physH)}
		ptDst := diPoint{X: int32(curWinX), Y: int32(curWinY)}
		ptSrc := diPoint{}
		constAlpha := uint8(math.Round(maxAlpha))
		blend := diBlendFunction{
			BlendOp:             diAcSrcOver,
			SourceConstantAlpha: constAlpha,
			AlphaFormat:         diAcSrcAlpha,
		}
		_, _, _ = diProcUpdateLayeredWindow.Call(
			c.hwnd, 0, uintptr(unsafe.Pointer(&ptDst)), uintptr(unsafe.Pointer(&sizeDst)),
			c.memDC, uintptr(unsafe.Pointer(&ptSrc)), 0,
			uintptr(unsafe.Pointer(&blend)), diUlwAlpha,
		)
		return
	}
	drawVeloraNotchShapePhys(pix, physW, physH, nx*u, w*u, h*u, rL*u, eL*u, eBotL*u, rR*u, eR*u, eBotR*u, pillMode, pal)

	shiftX := (edgeLeftProg-edgeRightProg)*24.0 + pageShift - liquidSkewDp*0.24
	mapXStatic := func(lx float64) float64 {
		rel := lx - w/2.0
		return (nx + w/2.0 + rel*contentScaleX) * u
	}
	mapX := func(lx float64) float64 {
		rel := lx - w/2.0
		return (nx + w/2.0 + rel*contentScaleX + shiftX) * u
	}
	mapY := func(ly float64) float64 {
		return (ly * contentScaleY) * u
	}
	mapW := func(lw float64) float64 { return lw * contentScaleX * u }
	mapH := func(lh float64) float64 { return lh * contentScaleY * u }

	var g uintptr
	_, _, _ = diProcGdipCreateFromHDC.Call(c.memDC, uintptr(unsafe.Pointer(&g)))
	if g == 0 {
		return
	}
	defer diProcGdipDeleteGraphics.Call(g)

	_, _, _ = diProcGdipSetSmoothingMode.Call(g, 2)   // SmoothingModeHighQuality
	_, _, _ = diProcGdipSetPixelOffsetMode.Call(g, 2) // PixelOffsetModeHighQuality (subpixel AA precision)
	_, _, _ = diProcGdipSetTextRendering.Call(g, 4)   // TextRenderingHintAntiAlias (16-level smooth grayscale AA)

	progRatio := 0.0
	if dur > 0 {
		progRatio = clamp01(pos / dur)
	}

	activeAccent := pal.Accent
	if pal.HasCoverDom {
		activeAccent = lerpColor(pal.Accent, pal.CoverDom, 0.45)
	}

	switch mode {
	case diModeResting, diModeClosed:
		miniPx, topCov, leftCov, ringCx, ringCy, strokeW, ringR := closedPillCoverAndRingGeom(nx, w, h, u)

		miniArt := coverImgMini
		isFallbackMini := false
		if miniArt == nil {
			miniArt = defaultArtMini
			isFallbackMini = true
		}
		if showCover && showClosedCover && miniArt != nil {
			if closedCoverSpin && closedSpinRad != 0 {
				blitRotatedPremulRGBAWithOp(pix, physW, physH, miniArt, leftCov, topCov, closedSpinRad, 1.0)
			} else {
				blitPremulRGBAWithOp(pix, physW, physH, miniArt, leftCov, topCov, 1.0)
			}
			if isFallbackMini {
				leftCx := float64(leftCov) + float64(miniPx-1)/2.0
				c.drawPhosphorGlyphGdip(g, false, 0xE33C, leftCx, ringCy, 11.0*u, rgbaToARGB(pal.TextVariant, 0.55))
			}
		}
		if showProgress && showClosedProgress {
			ringProg := progRatio
			if ringProg < 0.02 && playing {
				ringProg = 0.02
			}
			drawPkRingGdip(g, ringCx, ringCy, ringR, strokeW, ringProg, 1.0,
				rgbaToRGBHex(activeAccent), rgbaToARGB(activeAccent, 0.25))
		}

	case diModeHeadphones:
		if contentOp > 0.02 {
			if showStatus {
				c.drawSystemStatusIconsGdip(g, mapXStatic(w-76), mapXStatic(w-52), mapXStatic(w-28), mapY(18), u, wifiOn, gpsOn, btOn, hoverZone, contentOp, pal)
			}

			outputs := audioOutputs
			if len(outputs) == 0 {
				bat := -1
				if isBt && btBat >= 0 && btBat <= 100 {
					bat = btBat
				}
				outputs = []windowsAudioEndpoint{{
					Name:     audioDev,
					ConnType: connType,
					IsActive: true,
					BatPct:   bat,
					VolPct:   vol,
				}}
			}

			activeDev := outputs[0]
			for _, d := range outputs {
				if d.IsActive {
					activeDev = d
					break
				}
			}

			if hoverZone == diHitHpToggle {
				drawFilledPillGdip(g, mapX(18), mapY(32), mapW(384), mapH(52), 12.0*u, rgbaToARGB(color.RGBA{R: 0x1C, G: 0x20, B: 0x26, A: 255}, 0.60*contentOp))
			}

			activeGlyph := rune(0xE44A)
			if activeDev.ConnType == "bluetooth" {
				activeGlyph = 0xE0DA
			} else if activeDev.ConnType == "wired" {
				activeGlyph = 0xE2A6
			}

			activeVol := activeDev.VolPct
			if activeVol < 0 {
				activeVol = vol
			}
			if muted {
				activeVol = 0
			}
			drawPkRingGdip(g, mapX(40), mapY(58), 14.5*u, 3.2*u, clamp01(float64(activeVol)/100.0), contentOp,
				rgbaToRGBHex(pal.Accent), rgbaToARGB(pal.Accent, 0.22))
			c.drawPhosphorGlyphGdip(g, false, activeGlyph, mapX(40), mapY(58), 14.5*u, rgbaToARGB(pal.TextMain, contentOp))

			hpCol := pal.TextMain
			hpSize := 19.5 + 1.8*hpListProg
			if hoverZone == diHitHpToggle || hpListOpen {
				hpCol = pal.Accent
				if hoverZone == diHitHpToggle {
					hpSize += 1.0
				}
			}
			c.drawPhosphorGlyphGdip(g, hpListOpen, 0xE2A6, mapX(384), mapY(58), hpSize*u, rgbaToARGB(hpCol, contentOp))

			others := diOtherAudioIndices(outputs)
			if hpListProg > 0.01 && len(others) > 0 {
				listEase := nusaCssEase(hpListProg)
				divW := 16.0 * listEase
				drawFilledPillGdip(g, mapX(210.0-divW/2.0), mapY(91.0)-2.0*u, mapW(divW), 4.0*u, 2.0*u, rgbaToARGB(pal.Accent, listEase*contentOp))

				_, _, _ = diProcGdipSetClipRect.Call(
					g,
					uintptr(math.Float32bits(float32(mapX(14)))),
					uintptr(math.Float32bits(float32(mapY(96)))),
					uintptr(math.Float32bits(float32(mapW(392)))),
					uintptr(math.Float32bits(float32(math.Max(0, mapH(h-114))))),
					0,
				)

				for idx, origIdx := range others {
					dev := outputs[origIdx]
					rowDelay := float64(idx) * 0.12
					rowProg := clamp01((hpListProg - rowDelay) / (1.0 - rowDelay))
					rowEase := nusaCssEase(rowProg)
					if rowEase <= 0.01 {
						continue
					}
					slideY := (1.0 - rowEase) * 10.0
					rowOp := contentOp * rowEase
					cy := 68.0 + float64(idx+1)*56.0 + slideY
					rowHovered := hoverZone == diHitAudioDev0+idx

					if rowHovered {
						drawFilledPillGdip(g, mapX(18), mapY(cy-26), mapW(384), mapH(52), 12.0*u, rgbaToARGB(color.RGBA{R: 0x1C, G: 0x20, B: 0x26, A: 255}, 0.80*rowOp))
					}

					iconGlyph := rune(0xE44A)
					if dev.ConnType == "bluetooth" {
						iconGlyph = 0xE0DA
					} else if dev.ConnType == "wired" {
						iconGlyph = 0xE2A6
					}

					drawPkRingGdip(g, mapX(40), mapY(cy), 14.5*u, 2.8*u, 0.0, rowOp,
						rgbaToRGBHex(pal.TextVariant), rgbaToARGB(pal.TextVariant, 0.24))
					iconCol := pal.TextVariant
					iconOp := 0.58 * rowOp
					if rowHovered {
						iconCol = pal.TextMain
						iconOp = 0.85 * rowOp
					}
					c.drawPhosphorGlyphGdip(g, false, iconGlyph, mapX(40), mapY(cy), 14.5*u, rgbaToARGB(iconCol, iconOp))

					radioCol := pal.TextVariant
					radioAlpha := 0.35 * rowOp
					if rowHovered {
						radioCol = pal.Accent
						radioAlpha = 0.85 * rowOp
					}
					drawRoundedRectStrokeGdip(g, mapX(384)-7.5*u, mapY(cy)-7.5*u, 15*u, 15*u, 7.5*u, 1.6*u, rgbaToARGB(radioCol, radioAlpha))
				}
				_, _, _ = diProcGdipResetClip.Call(g)
			}

			if showDots {
				drawVelora5DotsGdip(g, mapX(210), mapY(h-11.0), u, 2, hoverZone, contentOp, pal)
			}
		}

	case diModeMusic:
		if contentOp > 0.02 {
			if showStatus {
				c.drawSystemStatusIconsGdip(g, mapXStatic(344), mapXStatic(368), mapXStatic(392), mapY(18), u, wifiOn, gpsOn, btOn, hoverZone, contentOp, pal)
			}

			isCircleCover := coverStyle == "circle"
			art := coverImg
			if isCircleCover && coverImgCircle != nil {
				art = coverImgCircle
			}
			isFallbackArt := false
			if art == nil {
				if isCircleCover && defaultArtCirc != nil {
					art = defaultArtCirc
				} else {
					art = defaultArt
				}
				isFallbackArt = true
			}
			if showCover && art != nil {
				artLeft := int(math.Round(mapX(22)))
				artTop := int(math.Round(mapY(33)))
				if isCircleCover && coverSpin && coverSpinRad != 0 {
					blitRotatedPremulRGBAWithOp(pix, physW, physH, art, artLeft, artTop, coverSpinRad, contentOp)
				} else {
					blitPremulRGBAWithOp(pix, physW, physH, art, artLeft, artTop, contentOp)
				}
				if isFallbackArt {
					c.drawPhosphorGlyphGdip(g, false, 0xE33C, mapX(45), mapY(56), 20.0*u, rgbaToARGB(pal.TextVariant, 0.50*contentOp))
				}
				borderRad := 8.0 * u
				if isCircleCover {
					borderRad = mapW(23)
				}
				drawRoundedRectStrokeGdip(g, mapX(22), mapY(33), mapW(46), mapH(46), borderRad, 1.0*u, rgbaToARGB(pal.TextMain, 0.14*contentOp))
			}

			if showHeart {
				drawCircleButtonBgGdip(g, mapX(342), mapY(56), 16.5*u, hoverZone == diHitFav, false, contentOp, pal)
				favCol := pal.TextVariant
				if isFav {
					favCol = pal.Primary
				} else if hoverZone == diHitFav {
					favCol = pal.TextMain
				}
				c.drawPhosphorGlyphGdip(g, isFav, 0xE2A8, mapX(342), mapY(56), 16.5*u, rgbaToARGB(favCol, contentOp))
			}

			drawCircleButtonBgGdip(g, mapX(383), mapY(56), 16.5*u, hoverZone == diHitArrow, false, contentOp, pal)
			hpCol := pal.TextMain
			if hoverZone == diHitArrow {
				hpCol = pal.Primary
			}
			c.drawPhosphorGlyphGdip(g, false, 0xE2A6, mapX(383), mapY(56), 16.5*u, rgbaToARGB(hpCol, contentOp))

			if showProgress {
				drawVeloraProgressBarGdip(g, mapX(22), mapY(98), mapW(376), 4.5*u, progRatio, hoverZone == diHitSeek, hoverRatio, contentOp, progressStyle, playing, now, u, pal)
			}

			if showLyric {
				drawCircleButtonBgGdip(g, mapX(41), mapY(134), 17.5*u, hoverZone == diHitBubble, false, contentOp, pal)
				micCol := pal.TextMain
				if hoverZone == diHitBubble {
					micCol = pal.Primary
				}
				c.drawPhosphorGlyphGdip(g, false, 0xE75C, mapX(41), mapY(134), 17.5*u, rgbaToARGB(micCol, contentOp))
			}

			drawCircleButtonBgGdip(g, mapX(162), mapY(134), 17.5*u, hoverZone == diHitPrev, false, contentOp, pal)
			prevCol := pal.TextMain
			if hoverZone == diHitPrev {
				prevCol = pal.Primary
			}
			c.drawPhosphorGlyphGdip(g, true, 0xE5A4, mapX(162), mapY(134), 18.0*u, rgbaToARGB(prevCol, contentOp))

			playBg := pal.Accent
			if hoverZone == diHitPlay {
				playBg = lerpColor(pal.Accent, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 0.14)
			}
			drawFilledPillGdip(g, mapX(210)-20.0*u, mapY(134)-20.0*u, 40.0*u, 40.0*u, 20.0*u, rgbaToARGB(playBg, contentOp))
			onTertiary := color.RGBA{R: 0x34, G: 0x11, B: 0x00, A: 255}
			if trackLoading {
				drawLoadingSpinnerGdip(g, mapX(210), mapY(134), 16.0*u, rgbaToARGB(onTertiary, contentOp), now)
			} else {
				playGlyph := rune(0xE3D0) // play
				if playing {
					playGlyph = 0xE39E // pause
				}
				c.drawPhosphorGlyphGdip(g, true, playGlyph, mapX(210), mapY(134), 18.5*u, rgbaToARGB(onTertiary, contentOp))
			}

			drawCircleButtonBgGdip(g, mapX(258), mapY(134), 17.5*u, hoverZone == diHitNext, false, contentOp, pal)
			nextCol := pal.TextMain
			if hoverZone == diHitNext {
				nextCol = pal.Primary
			}
			c.drawPhosphorGlyphGdip(g, true, 0xE5A6, mapX(258), mapY(134), 18.0*u, rgbaToARGB(nextCol, contentOp))

			drawCircleButtonBgGdip(g, mapX(379), mapY(134), 17.5*u, hoverZone == diHitQueueOpen, false, contentOp, pal)
			qCol := pal.TextVariant
			if hoverZone == diHitQueueOpen {
				qCol = pal.TextMain
			}
			c.drawPhosphorGlyphGdip(g, false, 0xE6AC, mapX(379), mapY(134), 17.5*u, rgbaToARGB(qCol, contentOp))

			if showDots {
				drawVelora5DotsGdip(g, mapX(210), mapY(169), u, 0, hoverZone, contentOp, pal)
			}
		}

	case diModeLyrics:
		if contentOp > 0.02 {
			if showStatus {
				c.drawSystemStatusIconsGdip(g, mapXStatic(w-76), mapXStatic(w-52), mapXStatic(w-28), mapY(18), u, wifiOn, gpsOn, btOn, hoverZone, contentOp, pal)
			}

			isCircleCoverL := coverStyle == "circle"
			art := coverImg
			if isCircleCoverL && coverImgCircle != nil {
				art = coverImgCircle
			}
			isFallbackArtL := false
			if art == nil {
				if isCircleCoverL && defaultArtCirc != nil {
					art = defaultArtCirc
				} else {
					art = defaultArt
				}
				isFallbackArtL = true
			}
			if showCover && art != nil {
				artLeftL := int(math.Round(mapX(22)))
				artTopL := int(math.Round(mapY(33)))
				if isCircleCoverL && coverSpin && coverSpinRad != 0 {
					blitRotatedPremulRGBAWithOp(pix, physW, physH, art, artLeftL, artTopL, coverSpinRad, contentOp)
				} else {
					blitPremulRGBAWithOp(pix, physW, physH, art, artLeftL, artTopL, contentOp)
				}
				if isFallbackArtL {
					c.drawPhosphorGlyphGdip(g, false, 0xE33C, mapX(45), mapY(56), 20.0*u, rgbaToARGB(pal.TextVariant, 0.50*contentOp))
				}
				borderRadL := 8.0 * u
				if isCircleCoverL {
					borderRadL = mapW(23)
				}
				drawRoundedRectStrokeGdip(g, mapX(22), mapY(33), mapW(46), mapH(46), borderRadL, 1.0*u, rgbaToARGB(pal.TextMain, 0.14*contentOp))
			}

			if showProgress {
				drawVeloraProgressBarGdip(g, mapX(22), mapY(98), mapW(214), 4.5*u, progRatio, hoverZone == diHitSeek, hoverRatio, contentOp, progressStyle, playing, now, u, pal)
			}

			drawCircleButtonBgGdip(g, mapX(41), mapY(134), 16.0*u, hoverZone == diHitBubble, true, contentOp, pal)
			c.drawPhosphorGlyphGdip(g, false, 0xE75C, mapX(41), mapY(134), 15.5*u, rgbaToARGB(pal.Primary, contentOp))

			drawCircleButtonBgGdip(g, mapX(96), mapY(134), 16.0*u, hoverZone == diHitPrev, false, contentOp, pal)
			c.drawPhosphorGlyphGdip(g, true, 0xE5A4, mapX(96), mapY(134), 16.0*u, rgbaToARGB(pal.TextMain, contentOp))

			playBgL := pal.Accent
			if hoverZone == diHitPlay {
				playBgL = lerpColor(pal.Accent, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 0.14)
			}
			drawFilledPillGdip(g, mapX(138)-17.5*u, mapY(134)-17.5*u, 35.0*u, 35.0*u, 17.5*u, rgbaToARGB(playBgL, contentOp))
			onTertiaryL := color.RGBA{R: 0x34, G: 0x11, B: 0x00, A: 255}
			if trackLoading {
				drawLoadingSpinnerGdip(g, mapX(138), mapY(134), 14.5*u, rgbaToARGB(onTertiaryL, contentOp), now)
			} else {
				playGlyph := rune(0xE3D0)
				if playing {
					playGlyph = 0xE39E
				}
				c.drawPhosphorGlyphGdip(g, true, playGlyph, mapX(138), mapY(134), 16.5*u, rgbaToARGB(onTertiaryL, contentOp))
			}

			drawCircleButtonBgGdip(g, mapX(180), mapY(134), 16.0*u, hoverZone == diHitNext, false, contentOp, pal)
			c.drawPhosphorGlyphGdip(g, true, 0xE5A6, mapX(180), mapY(134), 16.0*u, rgbaToARGB(pal.TextMain, contentOp))

			if showHeart {
				drawCircleButtonBgGdip(g, mapX(222), mapY(134), 16.0*u, hoverZone == diHitFav, false, contentOp, pal)
				favCol := pal.TextVariant
				if isFav {
					favCol = pal.Primary
				}
				c.drawPhosphorGlyphGdip(g, isFav, 0xE2A8, mapX(222), mapY(134), 16.0*u, rgbaToARGB(favCol, contentOp))
			}

			if showDots {
				drawVelora5DotsGdip(g, mapX(132), mapY(169), u, 0, hoverZone, contentOp, pal)
			}

			drawFilledPillGdip(g, mapX(250), mapY(33), mapW(254), mapH(118.5), 10.0*u, rgbaToARGB(pal.PrimaryDim, 0.94*contentOp))
		}

	case diModeQueue:
		if contentOp > 0.02 {
			if showStatus {
				c.drawSystemStatusIconsGdip(g, mapXStatic(w-76), mapXStatic(w-52), mapXStatic(w-28), mapY(18), u, wifiOn, gpsOn, btOn, hoverZone, contentOp, pal)
			}

			drawCircleButtonBgGdip(g, mapX(38), mapY(54), 16.5*u, hoverZone == diHitShuffle, shuffle, contentOp, pal)
			shufCol := pal.TextMain
			if shuffle {
				shufCol = pal.Accent
			} else if hoverZone == diHitShuffle {
				shufCol = pal.Primary
			}
			c.drawPhosphorGlyphGdip(g, false, 0xE422, mapX(38), mapY(54), 16.0*u, rgbaToARGB(shufCol, contentOp))

			playBgQ := pal.Accent
			if hoverZone == diHitPlay {
				playBgQ = lerpColor(pal.Accent, color.RGBA{R: 255, G: 255, B: 255, A: 255}, 0.14)
			}
			drawFilledPillGdip(g, mapX(375.5)-16.5*u, mapY(54)-16.5*u, 33.0*u, 33.0*u, 16.5*u, rgbaToARGB(playBgQ, contentOp))
			onTertiaryQ := color.RGBA{R: 0x34, G: 0x11, B: 0x00, A: 255}
			if trackLoading {
				drawLoadingSpinnerGdip(g, mapX(375.5), mapY(54), 13.5*u, rgbaToARGB(onTertiaryQ, contentOp), now)
			} else {
				qPlayGlyph := rune(0xE3D0)
				if playing {
					qPlayGlyph = 0xE39E
				}
				c.drawPhosphorGlyphGdip(g, true, qPlayGlyph, mapX(375.5), mapY(54), 15.5*u, rgbaToARGB(onTertiaryQ, contentOp))
			}

			const qViewTop = 78.0
			const qViewBot = 242.0
			clipTopPx := int(math.Round(mapY(qViewTop)))
			clipBotPx := int(math.Round(mapY(qViewBot)))
			_, _, _ = diProcGdipSetClipRect.Call(
				g,
				uintptr(math.Float32bits(float32(mapX(14)))),
				uintptr(math.Float32bits(float32(mapY(qViewTop)))),
				uintptr(math.Float32bits(float32(mapW(382)))),
				uintptr(math.Float32bits(float32(mapH(qViewBot-qViewTop)))),
				0,
			)

			firstIdx := maxInt(0, int(math.Floor(qScrollDp/54.0))-1)
			lastIdx := minInt(len(qItems)-1, int(math.Ceil((qScrollDp+164.0)/54.0)))
			var lazyURLs []string

			if qScrolling {
				_, _, _ = diProcGdipSetTextRendering.Call(g, 5) // AntiAlias (no grid-fit jitter during scroll)
			}
			for itemIdx := firstIdx; itemIdx <= lastIdx; itemIdx++ {
				item := qItems[itemIdx]
				cy := 104.0 + float64(itemIdx)*54.0 - qScrollDp
				isCur := itemIdx == qIndex
				rowHovered := (hoverZone == diHitQueueRow || hoverZone == diHitQueueFav) && hoverQueueIdx == itemIdx

				rowBg := color.RGBA{R: 0x14, G: 0x16, B: 0x18, A: 255}
				rowAlpha := 0.85 * contentOp
				if isCur {
					rowBg = color.RGBA{R: 0x1C, G: 0x20, B: 0x26, A: 255}
					rowAlpha = 0.96 * contentOp
				} else if rowHovered {
					rowBg = color.RGBA{R: 0x22, G: 0x25, B: 0x2A, A: 255}
					rowAlpha = 0.96 * contentOp
				}
				drawFilledPillGdip(g, mapX(18), mapY(cy-24), mapW(374), mapH(48), 10.0*u, rgbaToARGB(rowBg, rowAlpha))

				c.mu.Lock()
				qArt := c.queueArtCache[item.CoverURL]
				if qArt == nil && item.CoverURL != "" && !c.queueArtPend[item.CoverURL] {
					c.queueArtPend[item.CoverURL] = true
					lazyURLs = append(lazyURLs, item.CoverURL)
				}
				c.mu.Unlock()
				isFallbackQ := false
				if qArt == nil {
					if isCur && c.rawCover != nil {
						smPx := maxInt(38, int(math.Round(38.0*u)))
						qArt = resizeCoverBilinear(c.rawCover, smPx, smPx, 6.0*u)
					} else {
						qArt = defaultArtSm
						isFallbackQ = true
					}
				}
				if qArt != nil {
					blitPremulRGBAWithOpClipSubY(pix, physW, physH, qArt, int(math.Round(mapX(25))), mapY(cy-19), clipTopPx, clipBotPx, contentOp)
					if isFallbackQ && !isCur {
						c.drawPhosphorGlyphGdip(g, false, 0xE33C, mapX(44), mapY(cy), 16.0*u, rgbaToARGB(pal.TextVariant, 0.50*contentOp))
					}
				}
				if isCur {
					artX := int(math.Round(mapX(25)))
					ovW := mapW(38)
					ovH := mapH(38)
					if qArt != nil {
						ovW = float64(qArt.Bounds().Dx())
						ovH = float64(qArt.Bounds().Dy())
					}
					drawFilledPillGdip(g, float64(artX)+0.5, mapY(cy-19)+0.5, ovW, ovH, 6.0*u, rgbaToARGB(pal.BgDeep, 0.52*contentOp))
					gRune := rune(0xE3D0)
					if playing {
						gRune = 0xE39E
					}
					c.drawPhosphorGlyphGdip(g, true, gRune, float64(artX)+ovW/2.0, mapY(cy), 15.0*u, rgbaToARGB(pal.TextMain, contentOp))
				}

				qFavCol := pal.TextVariant
				if item.IsFavorite {
					qFavCol = pal.Accent
				} else if hoverZone == diHitQueueFav && hoverQueueIdx == itemIdx {
					qFavCol = pal.Accent
				}
				c.drawPhosphorGlyphGdip(g, item.IsFavorite, 0xE2A8, mapX(372), mapY(cy), 16.5*u, rgbaToARGB(qFavCol, contentOp))
			}
			_, _, _ = diProcGdipResetClip.Call(g)

			for _, lURL := range lazyURLs {
				go c.loadQueueArtAsync(lURL)
			}

			if len(qItems) > 3 {
				trackTop := 82.0
				trackH := 156.0
				totalH := float64(len(qItems)) * 54.0
				maxScrollDp := math.Max(1.0, totalH-162.0)
				thumbH := math.Max(22.0, trackH*(162.0/totalH))
				thumbY := trackTop + clamp01(qScrollDp/maxScrollDp)*(trackH-thumbH)
				drawFilledPillGdip(g, mapX(398.5), mapY(trackTop), mapW(3.5), mapH(trackH), 1.75*u, rgbaToARGB(pal.TextOutline, 0.16*contentOp))
				thumbCol := pal.Primary
				thumbAlpha := 0.45 * contentOp
				if hoverZone == diHitQueueScroll || queueDragScroll {
					thumbAlpha = 0.85 * contentOp
				}
				drawFilledPillGdip(g, mapX(398.5), mapY(thumbY), mapW(3.5), mapH(thumbH), 1.75*u, rgbaToARGB(thumbCol, thumbAlpha))
			}

			if showDots {
				drawVelora5DotsGdip(g, mapX(210), mapY(260), u, 1, hoverZone, contentOp, pal)
			}
		}
	}

	if !c.drawVeloraTextD2D(
		pix, physW, physH, nx, w, h, u, mode, contentOp, mapX, mapXStatic, mapY, mapW, mapH,
		title, artist, pos, dur, playing, lyricsOpen, vol, muted, eqPreset, streakDays,
		audioDev, connType, btBat, sysBat, isBt, audioOutputs, hpListOpen, hpListProg, lyricsLoading, lyricsHasPlain, lyrics, lyricIdx, prevLyricIdx, lyricAnimProg, lyricCurY, lyricManualOffsetDp, hoverLyric,
		closedLyricFrom, closedLyricTo, closedLyricProg,
		qTitle, qIndex, qScrollDp, qItems, pal, now.Sub(c.marqueeStart).Seconds(), diLang, qScrolling, effLyricPos, lyricSweep,
	) {
		c.drawVeloraTextGdip(
			g, nx, w, h, u, mode, contentOp, mapX, mapXStatic, mapY, mapW, mapH,
			title, artist, pos, dur, playing, lyricsOpen, vol, muted, eqPreset, streakDays,
			audioDev, connType, btBat, sysBat, isBt, audioOutputs, hpListOpen, hpListProg, lyricsLoading, lyricsHasPlain, lyrics, lyricIdx, prevLyricIdx, lyricAnimProg, lyricCurY, lyricManualOffsetDp, hoverLyric,
			closedLyricFrom, closedLyricTo, closedLyricProg,
			qTitle, qIndex, qScrollDp, qItems, pal, now.Sub(c.marqueeStart).Seconds(), diLang, effLyricPos, lyricSweep,
		)
	}

	if mode == diModeQueue && contentOp > 0.02 && len(qItems) > 0 {
		maxScrollDp := math.Max(0.0, float64(len(qItems))*54.0-162.0)
		drawQueueScrollFadePhys(pix, physW, physH, mapX(14), mapW(382), mapY(78), mapY(242), u, qScrollDp, maxScrollDp, contentOp, pal)
	}

	if (edgeLeftProg > 0.02 || edgeRightProg > 0.02) && (mode == diModeMusic || mode == diModeLyrics || mode == diModeQueue || mode == diModeHeadphones) {
		drawEdgeGradientPhys(pix, physW, physH, nx*u, w*u, h*u, u, curPage, edgeLeftProg, edgeRightProg, contentOp, pal)
		c.drawEdgePeekGdip(g, nx, w, h, u, curPage, edgeLeftProg, edgeRightProg, contentOp, pal)
	}

	clipOutsideNotchPhys(pix, physW, physH, nx*u, w*u, h*u, rL*u, eL*u, eBotL*u, rR*u, eR*u, eBotR*u, pillMode, pal.TextOutline)

	sizeDst := diSize{Cx: int32(physW), Cy: int32(physH)}
	ptDst := diPoint{X: int32(curWinX), Y: int32(curWinY)}
	ptSrc := diPoint{}
	maxAlpha := 255.0 * clamp01(float64(opacityPct)/100.0)
	constAlpha := uint8(math.Round(maxAlpha))
	if easeAppear < 0.45 {
		constAlpha = uint8(math.Round(maxAlpha * clamp01(easeAppear/0.45)))
	}
	blend := diBlendFunction{
		BlendOp:             diAcSrcOver,
		SourceConstantAlpha: constAlpha,
		AlphaFormat:         diAcSrcAlpha,
	}
	_, _, _ = diProcUpdateLayeredWindow.Call(
		c.hwnd,
		0,
		uintptr(unsafe.Pointer(&ptDst)),
		uintptr(unsafe.Pointer(&sizeDst)),
		c.memDC,
		uintptr(unsafe.Pointer(&ptSrc)),
		0,
		uintptr(unsafe.Pointer(&blend)),
		diUlwAlpha,
	)
}

func argbWithOp(argb uint32, op float64) uint32 {
	a := uint32(math.Round(float64((argb>>24)&0xFF) * clamp01(op)))
	return (a << 24) | (argb & 0x00FFFFFF)
}

func rgbaToARGB(c color.RGBA, op float64) uint32 {
	a := uint32(math.Round(float64(c.A) * clamp01(op)))
	return (a << 24) | (uint32(c.R) << 16) | (uint32(c.G) << 8) | uint32(c.B)
}

func rgbaToRGBHex(c color.RGBA) uint32 {
	return (uint32(c.R) << 16) | (uint32(c.G) << 8) | uint32(c.B)
}

func drawVeloraLogoGdip(g uintptr, cx, cy, size float64, pal veloraBrandPalette, op float64) {
	s := size / 512.0
	ox := cx - size/2.0
	oy := cy - size/2.0

	var penAccent uintptr
	_, _, _ = diProcGdipCreatePen1.Call(uintptr(rgbaToARGB(pal.Accent, op)), uintptr(math.Float32bits(float32(math.Max(1.4, 42.0*s)))), 2, uintptr(unsafe.Pointer(&penAccent)))
	if penAccent != 0 {
		_, _, _ = diProcGdipSetPenStartCap.Call(penAccent, 2)
		_, _, _ = diProcGdipSetPenEndCap.Call(penAccent, 2)
		rArc := 176.0 * s
		_, _, _ = diProcGdipDrawArc.Call(
			g, penAccent,
			uintptr(math.Float32bits(float32(ox+256*s-rArc))),
			uintptr(math.Float32bits(float32(oy+232*s-rArc))),
			uintptr(math.Float32bits(float32(rArc*2))),
			uintptr(math.Float32bits(float32(rArc*2))),
			uintptr(math.Float32bits(160.0)),
			uintptr(math.Float32bits(40.0)),
		)
		_, _, _ = diProcGdipDrawArc.Call(
			g, penAccent,
			uintptr(math.Float32bits(float32(ox+256*s-rArc))),
			uintptr(math.Float32bits(float32(oy+232*s-rArc))),
			uintptr(math.Float32bits(float32(rArc*2))),
			uintptr(math.Float32bits(float32(rArc*2))),
			uintptr(math.Float32bits(-20.0)),
			uintptr(math.Float32bits(40.0)),
		)
		_, _, _ = diProcGdipDeletePen.Call(penAccent)
	}

	var penPrim uintptr
	_, _, _ = diProcGdipCreatePen1.Call(uintptr(rgbaToARGB(pal.Primary, op)), uintptr(math.Float32bits(float32(math.Max(1.5, 42.0*s)))), 2, uintptr(unsafe.Pointer(&penPrim)))
	if penPrim != 0 {
		_, _, _ = diProcGdipSetPenStartCap.Call(penPrim, 2)
		_, _, _ = diProcGdipSetPenEndCap.Call(penPrim, 2)
		_, _, _ = diProcGdipSetPenLineJoin.Call(penPrim, 2)

		var path uintptr
		_, _, _ = diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&path)))
		if path != 0 {
			addBezierF(path, ox+205*s, oy+128*s, ox+162*s, oy+148*s, ox+140*s, oy+188*s, ox+140*s, oy+232*s)
			addBezierF(path, ox+140*s, oy+232*s, ox+140*s, oy+286*s, ox+174*s, oy+328*s, ox+256*s, oy+384*s)
			addBezierF(path, ox+256*s, oy+384*s, ox+338*s, oy+328*s, ox+372*s, oy+286*s, ox+372*s, oy+232*s)
			addBezierF(path, ox+372*s, oy+232*s, ox+372*s, oy+188*s, ox+350*s, oy+148*s, ox+307*s, oy+128*s)
			_, _, _ = diProcGdipDrawPath.Call(g, penPrim, path)
			_, _, _ = diProcGdipDeletePath.Call(path)
		}
		_, _, _ = diProcGdipDeletePen.Call(penPrim)
	}

	var brushAcc uintptr
	_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(pal.Accent, op)), uintptr(unsafe.Pointer(&brushAcc)))
	if brushAcc != 0 {
		fillCircleGdip(g, brushAcc, ox+256*s, oy+232*s, 64.0*s)
		_, _, _ = diProcGdipDeleteBrush.Call(brushAcc)
	}
}

func drawVeloraFlameGdip(g uintptr, cx, cy, size float64, col color.RGBA, op float64) {
	var brush uintptr
	_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(col, op)), uintptr(unsafe.Pointer(&brush)))
	if brush == 0 {
		return
	}
	defer diProcGdipDeleteBrush.Call(brush)

	var path uintptr
	_, _, _ = diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&path)))
	if path == 0 {
		return
	}
	defer diProcGdipDeletePath.Call(path)

	s := size / 24.0
	ox := cx - size/2.0
	oy := cy - size/2.0
	addBezierF(path, ox+12*s, oy+2*s, ox+16*s, oy+7*s, ox+19*s, oy+11*s, ox+19*s, oy+15*s)
	addBezierF(path, ox+19*s, oy+15*s, ox+19*s, oy+19.5*s, ox+15.8*s, oy+22*s, ox+12*s, oy+22*s)
	addBezierF(path, ox+12*s, oy+22*s, ox+8.2*s, oy+22*s, ox+5*s, oy+19.5*s, ox+5*s, oy+15*s)
	addBezierF(path, ox+5*s, oy+15*s, ox+5*s, oy+11*s, ox+8*s, oy+7*s, ox+12*s, oy+2*s)
	_, _, _ = diProcGdipClosePathFigure.Call(path)
	_, _, _ = diProcGdipFillPath.Call(g, brush, path)
}

func addBezierF(path uintptr, x1, y1, x2, y2, x3, y3, x4, y4 float64) {
	_, _, _ = diProcGdipAddPathBezier.Call(
		path,
		uintptr(math.Float32bits(float32(x1))),
		uintptr(math.Float32bits(float32(y1))),
		uintptr(math.Float32bits(float32(x2))),
		uintptr(math.Float32bits(float32(y2))),
		uintptr(math.Float32bits(float32(x3))),
		uintptr(math.Float32bits(float32(y3))),
		uintptr(math.Float32bits(float32(x4))),
		uintptr(math.Float32bits(float32(y4))),
	)
}

func drawRoundedRectStrokeGdip(g uintptr, x, y, w, h, r, strokeW float64, argb uint32) {
	path := createRoundedRectPath(x, y, w, h, r)
	if path == 0 {
		return
	}
	defer diProcGdipDeletePath.Call(path)

	var pen uintptr
	_, _, _ = diProcGdipCreatePen1.Call(uintptr(argb), uintptr(math.Float32bits(float32(strokeW))), 2, uintptr(unsafe.Pointer(&pen)))
	if pen == 0 {
		return
	}
	defer diProcGdipDeletePen.Call(pen)
	_, _, _ = diProcGdipDrawPath.Call(g, pen, path)
}

func drawFilledPillGdip(g uintptr, x, y, w, h, r float64, argb uint32) {
	if w <= 0 || h <= 0 {
		return
	}
	var brush uintptr
	_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(argb), uintptr(unsafe.Pointer(&brush)))
	if brush == 0 {
		return
	}
	defer diProcGdipDeleteBrush.Call(brush)

	if r <= 0.5 {
		var path uintptr
		_, _, _ = diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&path)))
		if path == 0 {
			return
		}
		defer diProcGdipDeletePath.Call(path)
		addLineF(path, x, y, x+w, y)
		addLineF(path, x+w, y, x+w, y+h)
		addLineF(path, x+w, y+h, x, y+h)
		_, _, _ = diProcGdipClosePathFigure.Call(path)
		_, _, _ = diProcGdipFillPath.Call(g, brush, path)
		return
	}
	path := createRoundedRectPath(x, y, w, h, r)
	if path == 0 {
		return
	}
	defer diProcGdipDeletePath.Call(path)
	_, _, _ = diProcGdipFillPath.Call(g, brush, path)
}

func createRoundedRectPath(x, y, w, h, r float64) uintptr {
	if r > w/2 {
		r = w / 2
	}
	if r > h/2 {
		r = h / 2
	}
	var path uintptr
	_, _, _ = diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&path)))
	if path == 0 {
		return 0
	}
	d := r * 2
	if d <= 0.5 {
		addLineF(path, x, y, x+w, y)
		addLineF(path, x+w, y, x+w, y+h)
		addLineF(path, x+w, y+h, x, y+h)
		_, _, _ = diProcGdipClosePathFigure.Call(path)
		return path
	}
	addArcF(path, x, y, d, d, 180, 90)
	addArcF(path, x+w-d, y, d, d, 270, 90)
	addArcF(path, x+w-d, y+h-d, d, d, 0, 90)
	addArcF(path, x, y+h-d, d, d, 90, 90)
	_, _, _ = diProcGdipClosePathFigure.Call(path)
	return path
}

func addLineF(path uintptr, x1, y1, x2, y2 float64) {
	_, _, _ = diProcGdipAddPathLine.Call(
		path,
		uintptr(math.Float32bits(float32(x1))),
		uintptr(math.Float32bits(float32(y1))),
		uintptr(math.Float32bits(float32(x2))),
		uintptr(math.Float32bits(float32(y2))),
	)
}

func addArcF(path uintptr, x, y, w, h, startAng, sweepAng float64) {
	_, _, _ = diProcGdipAddPathArc.Call(
		path,
		uintptr(math.Float32bits(float32(x))),
		uintptr(math.Float32bits(float32(y))),
		uintptr(math.Float32bits(float32(w))),
		uintptr(math.Float32bits(float32(h))),
		uintptr(math.Float32bits(float32(startAng))),
		uintptr(math.Float32bits(float32(sweepAng))),
	)
}

func drawVeloraBarsGdip(g uintptr, cx, cy, u, tSec float64, playing bool, pal veloraBrandPalette, op float64) {
	delays := []float64{0, -0.3, -0.6, -0.15, -0.45}
	bw := 2.3 * u
	gap := 2.0 * u
	maxH := 14.0 * u
	totalW := 5*bw + 4*gap
	startX := cx - totalW/2.0

	cStart := pal.Accent
	if pal.HasCoverDom {
		cStart = lerpColor(pal.Accent, pal.CoverDom, 0.5)
	}
	cEnd := pal.Primary

	for i := 0; i < 5; i++ {
		frac := 0.28
		if playing {
			phase := math.Mod(tSec-delays[i]+10.0, 1.8) / 0.9
			if phase > 1.0 {
				phase = 2.0 - phase
			}
			eased := 0.5 - 0.5*math.Cos(phase*math.Pi)
			frac = 0.28 + 0.72*eased
		}
		barCol := lerpColor(cStart, cEnd, float64(i)/4.0)
		bh := maxH * frac
		bx := startX + float64(i)*(bw+gap)
		by := cy - bh/2.0
		drawFilledPillGdip(g, bx, by, bw, bh, bw/2.0, rgbaToARGB(barCol, op))
	}
}

func drawVelora5DotsGdip(g uintptr, cx, cy, u float64, activeIdx, hoverZone int, op float64, pal veloraBrandPalette) {
	actW := 16.0 * u
	inactW := 4.4 * u
	h := 4.0 * u
	gap := 5.5 * u
	totalW := actW + 2*inactW + 2*gap
	x := cx - totalW/2.0
	for i := 0; i < 3; i++ {
		dw := inactW
		argb := rgbaToARGB(pal.Primary, 0.30*op)
		if i == activeIdx {
			dw = actW
			argb = rgbaToARGB(pal.Accent, op)
		} else if hoverZone == diHitDot0+i {
			argb = rgbaToARGB(pal.Primary, 0.85*op)
		}
		drawFilledPillGdip(g, x, cy-h/2.0, dw, h, h/2.0, argb)
		x += dw + gap
	}
}

func drawVeloraProgressBarGdip(g uintptr, x, cy, w, h, ratio float64, hovered bool, hoverRatio, op float64, style string, playing bool, now time.Time, u float64, pal veloraBrandPalette) {
	trackCol := color.RGBA{R: 0x2C, G: 0x30, B: 0x37, A: 255}
	clampedRatio := clamp01(ratio)

	if style == "wave" {
		barH := h
		if hovered {
			barH = h * 1.25
		}
		y := cy - barH/2.0
		fw := w * clampedRatio
		if fw < w {
			remX := x + fw
			remW := w - fw
			drawFilledPillGdip(g, remX, y, remW, barH, barH/2.0, rgbaToARGB(trackCol, op))
			if hovered && hoverRatio > clampedRatio {
				hx := x + w*clamp01(hoverRatio)
				if hx > remX {
					drawFilledPillGdip(g, remX, y, hx-remX, barH, barH/2.0, rgbaToARGB(pal.Primary, 0.30*op))
				}
			}
		}
		if fw >= 2.0*u {
			var path uintptr
			_, _, _ = diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&path)))
			if path != 0 {
				wavelength := 24.0 * u
				amp := 2.2 * u
				if hovered {
					amp = 2.6 * u
				}
				phase := 0.0
				if playing {
					phase = float64(now.UnixNano()) / 1e9 * (2.0 * math.Pi / 1.2)
				}
				step := 2.0 * u
				if step < 1.5 {
					step = 1.5
				}
				prevX := x
				prevY := cy + math.Sin(phase)*amp
				for dx := step; dx <= fw; dx += step {
					px := x + dx
					py := cy + math.Sin((dx/wavelength)*2.0*math.Pi+phase)*amp
					addLineF(path, prevX, prevY, px, py)
					prevX, prevY = px, py
				}
				if prevX < x+fw {
					endX := x + fw
					endY := cy + math.Sin((fw/wavelength)*2.0*math.Pi+phase)*amp
					addLineF(path, prevX, prevY, endX, endY)
				}
				var pen uintptr
				_, _, _ = diProcGdipCreatePen1.Call(uintptr(rgbaToARGB(pal.Primary, op)), uintptr(math.Float32bits(float32(barH))), 2, uintptr(unsafe.Pointer(&pen)))
				if pen != 0 {
					_, _, _ = diProcGdipSetPenStartCap.Call(pen, 2)
					_, _, _ = diProcGdipSetPenEndCap.Call(pen, 2)
					_, _, _ = diProcGdipSetPenLineJoin.Call(pen, 2)
					_, _, _ = diProcGdipDrawPath.Call(g, pen, path)
					_, _, _ = diProcGdipDeletePen.Call(pen)
				}
				_, _, _ = diProcGdipDeletePath.Call(path)
			}
		}
		knobR := barH * 1.05
		if hovered {
			knobR = barH * 1.30
		}
		var brush uintptr
		_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(pal.Accent, op)), uintptr(unsafe.Pointer(&brush)))
		if brush != 0 {
			fillCircleGdip(g, brush, x+fw, cy, knobR)
			_, _, _ = diProcGdipDeleteBrush.Call(brush)
		}
		return
	}

	if style == "dots" {
		const dotCount = 24
		dotR := h * 0.42
		if hovered {
			dotR = h * 0.54
		}
		var bTrack, bFill, bHover uintptr
		_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(trackCol, op)), uintptr(unsafe.Pointer(&bTrack)))
		_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(pal.Primary, op)), uintptr(unsafe.Pointer(&bFill)))
		if hovered {
			_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(pal.Primary, 0.35*op)), uintptr(unsafe.Pointer(&bHover)))
		}
		for i := 0; i < dotCount; i++ {
			t := float64(i) / float64(dotCount-1)
			cx := x + t*w
			brush := bTrack
			if t <= clampedRatio {
				brush = bFill
			} else if hovered && t <= clamp01(hoverRatio) && bHover != 0 {
				brush = bHover
			}
			if brush != 0 {
				fillCircleGdip(g, brush, cx, cy, dotR)
			}
		}
		if bTrack != 0 {
			_, _, _ = diProcGdipDeleteBrush.Call(bTrack)
		}
		if bFill != 0 {
			_, _, _ = diProcGdipDeleteBrush.Call(bFill)
		}
		if bHover != 0 {
			_, _, _ = diProcGdipDeleteBrush.Call(bHover)
		}
		knobR := h * 1.05
		if hovered {
			knobR = h * 1.30
		}
		var bKnob uintptr
		_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(pal.Accent, op)), uintptr(unsafe.Pointer(&bKnob)))
		if bKnob != 0 {
			fillCircleGdip(g, bKnob, x+w*clampedRatio, cy, knobR)
			_, _, _ = diProcGdipDeleteBrush.Call(bKnob)
		}
		return
	}

	barH := h
	if style == "thick" {
		barH = h * 1.65
	}
	if hovered {
		barH *= 1.25
	}
	y := cy - barH/2.0
	drawFilledPillGdip(g, x, y, w, barH, barH/2.0, rgbaToARGB(trackCol, op))

	fw := w * clampedRatio
	if fw >= barH {
		drawFilledPillGdip(g, x, y, fw, barH, barH/2.0, rgbaToARGB(pal.Primary, op))
	}

	if hovered {
		hx := x + w*clamp01(hoverRatio)
		drawFilledPillGdip(g, x, y, hx-x, barH, barH/2.0, rgbaToARGB(pal.Primary, 0.30*op))
	}

	knobR := barH * 1.05
	if hovered {
		knobR = barH * 1.30
	}
	var brush uintptr
	_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(pal.Accent, op)), uintptr(unsafe.Pointer(&brush)))
	if brush != 0 {
		fillCircleGdip(g, brush, x+fw, cy, knobR)
		_, _, _ = diProcGdipDeleteBrush.Call(brush)
	}
}

func drawCircleButtonBgGdip(g uintptr, cx, cy, r float64, hovered, active bool, op float64, pal veloraBrandPalette) {
	if active {
		col := pal.PrimaryDim
		if hovered {
			col = lerpColor(pal.PrimaryDim, pal.Primary, 0.25)
		}
		drawFilledPillGdip(g, cx-r, cy-r, r*2, r*2, r, rgbaToARGB(col, 0.72*op))
		return
	}
	if hovered {
		drawFilledPillGdip(g, cx-r, cy-r, r*2, r*2, r, rgbaToARGB(pal.TextVariant, 0.14*op))
	}
}

func (c *dynamicIslandController) drawSystemStatusIconsGdip(g uintptr, xWifi, xGps, xBt, cy, u float64, wifiOn, gpsOn, btOn bool, hoverZone int, op float64, pal veloraBrandPalette) {
	size := 15.0 * u
	activeCol := rgbaToARGB(pal.TextMain, 0.95*op)
	inactiveCol := rgbaToARGB(pal.TextOutline, 0.48*op)
	hoverR := 10.5 * u

	if hoverZone == diHitWifi {
		drawFilledPillGdip(g, xWifi-hoverR, cy-hoverR, hoverR*2, hoverR*2, hoverR, rgbaToARGB(pal.TextVariant, 0.16*op))
	} else if hoverZone == diHitGps {
		drawFilledPillGdip(g, xGps-hoverR, cy-hoverR, hoverR*2, hoverR*2, hoverR, rgbaToARGB(pal.TextVariant, 0.16*op))
	} else if hoverZone == diHitBt {
		drawFilledPillGdip(g, xBt-hoverR, cy-hoverR, hoverR*2, hoverR*2, hoverR, rgbaToARGB(pal.TextVariant, 0.16*op))
	}

	wifiGlyph := rune(0xE4EA) // wifi-high
	wifiCol := activeCol
	if !wifiOn {
		wifiGlyph = 0xE4F2 // wifi-slash
		wifiCol = inactiveCol
	}
	if hoverZone == diHitWifi {
		wifiCol = rgbaToARGB(pal.Primary, op)
	}
	c.drawPhosphorGlyphGdip(g, false, wifiGlyph, xWifi, cy, size, wifiCol)

	gpsCol := activeCol
	if !gpsOn {
		gpsCol = inactiveCol
	}
	if hoverZone == diHitGps {
		gpsCol = rgbaToARGB(pal.Primary, op)
	}
	c.drawPhosphorGlyphGdip(g, false, 0xE316, xGps, cy, size, gpsCol)
	if !gpsOn {
		var pen uintptr
		_, _, _ = diProcGdipCreatePen1.Call(uintptr(gpsCol), uintptr(math.Float32bits(float32(1.4*u))), 2, uintptr(unsafe.Pointer(&pen)))
		if pen != 0 {
			_, _, _ = diProcGdipSetPenStartCap.Call(pen, 2)
			_, _, _ = diProcGdipSetPenEndCap.Call(pen, 2)
			_, _, _ = diProcGdipDrawLine.Call(g, pen,
				uintptr(math.Float32bits(float32(xGps-5.5*u))), uintptr(math.Float32bits(float32(cy-5.5*u))),
				uintptr(math.Float32bits(float32(xGps+5.5*u))), uintptr(math.Float32bits(float32(cy+5.5*u))))
			_, _, _ = diProcGdipDeletePen.Call(pen)
		}
	}

	btGlyph := rune(0xE0DA) // bluetooth
	btCol := activeCol
	if !btOn {
		btGlyph = 0xE0DE // bluetooth-slash
		btCol = inactiveCol
	}
	if hoverZone == diHitBt {
		btCol = rgbaToARGB(pal.Primary, op)
	}
	c.drawPhosphorGlyphGdip(g, false, btGlyph, xBt, cy, size, btCol)
}

func drawEdgeGradientPhys(
	pix []byte,
	cw, ch int,
	nxPx, wPx, hPx, u float64,
	curPage int,
	leftProg, rightProg, op float64,
	pal veloraBrandPalette,
) {
	blendGrad := func(xStart, width, maxAlpha float64, fadeToRight bool) {
		if width <= 1.0 || maxAlpha <= 0.01 {
			return
		}
		x0 := maxInt(0, int(math.Floor(xStart)))
		x1 := minInt(cw-1, int(math.Ceil(xStart+width)))
		y1 := minInt(ch-1, int(math.Ceil(hPx)))
		for px := x0; px <= x1; px++ {
			fx := float64(px) + 0.5
			t := (fx - xStart) / width
			if !fadeToRight {
				t = 1.0 - t
			}
			t = clamp01(t)
			alpha := maxAlpha * (t * t)
			if alpha <= 0.005 {
				continue
			}
			for py := 0; py <= y1; py++ {
				blendPremulPixel(pix, cw, px, py, pal.BgDeep, alpha)
			}
		}
	}

	if leftProg > 0.02 {
		blendGrad(nxPx, 64.0*leftProg*u, 0.96*leftProg*op, false)
		blendGrad(nxPx+wPx-48.0*leftProg*u, 48.0*leftProg*u, 0.96*leftProg*op, true)
	}
	if rightProg > 0.02 {
		blendGrad(nxPx+wPx-64.0*rightProg*u, 64.0*rightProg*u, 0.96*rightProg*op, true)
		blendGrad(nxPx, 48.0*rightProg*u, 0.96*rightProg*op, false)
	}
}

func drawQueueScrollFadePhys(
	pix []byte,
	cw, ch int,
	xStartPx, widthPx, yTopPx, yBotPx, u float64,
	qScrollDp, maxScrollDp, op float64,
	pal veloraBrandPalette,
) {
	x0 := maxInt(0, int(math.Floor(xStartPx)))
	x1 := minInt(cw-1, int(math.Ceil(xStartPx+widthPx)))
	if x1 <= x0 {
		return
	}
	blendVert := func(yStart, yEnd, maxAlpha float64, fadeDown bool) {
		h := yEnd - yStart
		if h <= 1.0 || maxAlpha <= 0.01 {
			return
		}
		py0 := maxInt(0, int(math.Floor(yStart)))
		py1 := minInt(ch-1, int(math.Ceil(yEnd)))
		for py := py0; py <= py1; py++ {
			fy := float64(py) + 0.5
			t := (fy - yStart) / h
			if fadeDown {
				t = 1.0 - t
			}
			t = clamp01(t)
			alpha := maxAlpha * (t * t)
			if alpha <= 0.005 {
				continue
			}
			for px := x0; px <= x1; px++ {
				blendPremulPixel(pix, cw, px, py, pal.BgDeep, alpha)
			}
		}
	}

	topFactor := clamp01(qScrollDp / 14.0)
	if topFactor > 0.01 {
		blendVert(yTopPx, yTopPx+20.0*u, 0.98*topFactor*op, true)
	}
	botFactor := clamp01((maxScrollDp - qScrollDp) / 14.0)
	if botFactor > 0.01 {
		blendVert(yBotPx-22.0*u, yBotPx, 0.98*botFactor*op, false)
	}
}

func (c *dynamicIslandController) drawEdgePeekGdip(
	g uintptr,
	nx, w, h, u float64,
	curPage int,
	leftProg, rightProg, op float64,
	pal veloraBrandPalette,
) {
	drawBtn := func(isRight bool, prog float64) {
		if prog <= 0.02 {
			return
		}
		var btnX float64
		var glyph rune
		if isRight {
			btnX = nx + w - 21.0*prog
			glyph = 0xE13A // caret-right
		} else {
			btnX = nx + 21.0*prog
			glyph = 0xE138 // caret-left
		}
		btnR := 14.5 * u
		btnBg := color.RGBA{R: 0x2C, G: 0x30, B: 0x37, A: 255}
		drawFilledPillGdip(g, btnX*u-btnR, (h/2.0)*u-btnR, btnR*2, btnR*2, btnR, rgbaToARGB(btnBg, 0.92*prog*op))
		c.drawPhosphorGlyphGdip(g, false, glyph, btnX*u, (h/2.0)*u, 15.0*u, rgbaToARGB(pal.TextMain, 0.96*prog*op))
	}

	if leftProg > 0.02 {
		drawBtn(false, leftProg)
	}
	if rightProg > 0.02 {
		drawBtn(true, rightProg)
	}
}

func drawCircleStrokeGdip(g uintptr, cx, cy, r, strokeW float64, argb uint32) {
	var pen uintptr
	_, _, _ = diProcGdipCreatePen1.Call(uintptr(argb), uintptr(math.Float32bits(float32(strokeW))), 2, uintptr(unsafe.Pointer(&pen)))
	if pen == 0 {
		return
	}
	defer diProcGdipDeletePen.Call(pen)
	_, _, _ = diProcGdipDrawEllipse.Call(
		g, pen,
		uintptr(math.Float32bits(float32(cx-r))),
		uintptr(math.Float32bits(float32(cy-r))),
		uintptr(math.Float32bits(float32(r*2))),
		uintptr(math.Float32bits(float32(r*2))),
	)
}

func drawPhosphorMicGdip(g uintptr, cx, cy, size float64, active bool, op float64, pal veloraBrandPalette) {
	col := pal.TextMain
	if active {
		col = pal.Accent
	}
	s := size / 24.0
	ox := cx - size/2.0
	oy := cy - size/2.0

	var pen uintptr
	_, _, _ = diProcGdipCreatePen1.Call(uintptr(rgbaToARGB(col, op)), uintptr(math.Float32bits(float32(1.85*s))), 2, uintptr(unsafe.Pointer(&pen)))
	if pen == 0 {
		return
	}
	defer diProcGdipDeletePen.Call(pen)
	_, _, _ = diProcGdipSetPenStartCap.Call(pen, 2)
	_, _, _ = diProcGdipSetPenEndCap.Call(pen, 2)
	_, _, _ = diProcGdipSetPenLineJoin.Call(pen, 2)

	capPath := createRoundedRectPath(ox+9*s, oy+3*s, 6*s, 10*s, 3*s)
	if capPath != 0 {
		_, _, _ = diProcGdipDrawPath.Call(g, pen, capPath)
		_, _, _ = diProcGdipDeletePath.Call(capPath)
	}
	_, _, _ = diProcGdipDrawArc.Call(
		g, pen,
		uintptr(math.Float32bits(float32(ox+6*s))),
		uintptr(math.Float32bits(float32(oy+5*s))),
		uintptr(math.Float32bits(float32(12*s))),
		uintptr(math.Float32bits(float32(11.5*s))),
		uintptr(math.Float32bits(0.0)),
		uintptr(math.Float32bits(180.0)),
	)
	_, _, _ = diProcGdipDrawLine.Call(g, pen,
		uintptr(math.Float32bits(float32(ox+12*s))), uintptr(math.Float32bits(float32(oy+16.5*s))),
		uintptr(math.Float32bits(float32(ox+12*s))), uintptr(math.Float32bits(float32(oy+20.5*s))))
	_, _, _ = diProcGdipDrawLine.Call(g, pen,
		uintptr(math.Float32bits(float32(ox+9*s))), uintptr(math.Float32bits(float32(oy+20.5*s))),
		uintptr(math.Float32bits(float32(ox+15*s))), uintptr(math.Float32bits(float32(oy+20.5*s))))
}

func drawPhosphorHeadphonesGdip(g uintptr, cx, cy, size float64, hovered bool, op float64, pal veloraBrandPalette) {
	col := pal.TextMain
	if hovered {
		col = pal.Primary
	}
	s := size / 24.0
	ox := cx - size/2.0
	oy := cy - size/2.0

	var pen uintptr
	_, _, _ = diProcGdipCreatePen1.Call(uintptr(rgbaToARGB(col, op)), uintptr(math.Float32bits(float32(1.85*s))), 2, uintptr(unsafe.Pointer(&pen)))
	if pen == 0 {
		return
	}
	defer diProcGdipDeletePen.Call(pen)
	_, _, _ = diProcGdipSetPenStartCap.Call(pen, 2)
	_, _, _ = diProcGdipSetPenEndCap.Call(pen, 2)
	_, _, _ = diProcGdipSetPenLineJoin.Call(pen, 2)

	var path uintptr
	_, _, _ = diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&path)))
	if path == 0 {
		return
	}
	defer diProcGdipDeletePath.Call(path)

	_, _, _ = diProcGdipStartPathFigure.Call(path)
	addLineF(path, ox+4*s, oy+14*s, ox+4*s, oy+12*s)
	addArcF(path, ox+4*s, oy+4*s, 16*s, 16*s, 180, 180)
	addLineF(path, ox+20*s, oy+12*s, ox+20*s, oy+14*s)
	_, _, _ = diProcGdipDrawPath.Call(g, pen, path)

	leftCup := createRoundedRectPath(ox+4*s, oy+14*s, 3.5*s, 6*s, 1.5*s)
	if leftCup != 0 {
		_, _, _ = diProcGdipDrawPath.Call(g, pen, leftCup)
		_, _, _ = diProcGdipDeletePath.Call(leftCup)
	}
	rightCup := createRoundedRectPath(ox+16.5*s, oy+14*s, 3.5*s, 6*s, 1.5*s)
	if rightCup != 0 {
		_, _, _ = diProcGdipDrawPath.Call(g, pen, rightCup)
		_, _, _ = diProcGdipDeletePath.Call(rightCup)
	}
}

func drawPhosphorSkipBackGdip(g uintptr, cx, cy, size float64, hovered bool, op float64, pal veloraBrandPalette) {
	col := pal.TextMain
	if hovered {
		col = pal.Primary
	}
	var brush uintptr
	_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(col, op)), uintptr(unsafe.Pointer(&brush)))
	if brush == 0 {
		return
	}
	defer diProcGdipDeleteBrush.Call(brush)

	s := size / 24.0
	ox := cx - size/2.0
	oy := cy - size/2.0

	barPath := createRoundedRectPath(ox+5*s, oy+5*s, 2.4*s, 14*s, 1.2*s)
	if barPath != 0 {
		_, _, _ = diProcGdipFillPath.Call(g, brush, barPath)
		_, _, _ = diProcGdipDeletePath.Call(barPath)
	}
	var tri uintptr
	_, _, _ = diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&tri)))
	if tri != 0 {
		addLineF(tri, ox+18.5*s, oy+5.5*s, ox+8.5*s, oy+12*s)
		addLineF(tri, ox+8.5*s, oy+12*s, ox+18.5*s, oy+18.5*s)
		_, _, _ = diProcGdipClosePathFigure.Call(tri)
		_, _, _ = diProcGdipFillPath.Call(g, brush, tri)
		_, _, _ = diProcGdipDeletePath.Call(tri)
	}
}

func drawPhosphorSkipForwardGdip(g uintptr, cx, cy, size float64, hovered bool, op float64, pal veloraBrandPalette) {
	col := pal.TextMain
	if hovered {
		col = pal.Primary
	}
	var brush uintptr
	_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(col, op)), uintptr(unsafe.Pointer(&brush)))
	if brush == 0 {
		return
	}
	defer diProcGdipDeleteBrush.Call(brush)

	s := size / 24.0
	ox := cx - size/2.0
	oy := cy - size/2.0

	var tri uintptr
	_, _, _ = diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&tri)))
	if tri != 0 {
		addLineF(tri, ox+5.5*s, oy+5.5*s, ox+15.5*s, oy+12*s)
		addLineF(tri, ox+15.5*s, oy+12*s, ox+5.5*s, oy+18.5*s)
		_, _, _ = diProcGdipClosePathFigure.Call(tri)
		_, _, _ = diProcGdipFillPath.Call(g, brush, tri)
		_, _, _ = diProcGdipDeletePath.Call(tri)
	}
	barPath := createRoundedRectPath(ox+16.6*s, oy+5*s, 2.4*s, 14*s, 1.2*s)
	if barPath != 0 {
		_, _, _ = diProcGdipFillPath.Call(g, brush, barPath)
		_, _, _ = diProcGdipDeletePath.Call(barPath)
	}
}

func drawLoadingSpinnerGdip(g uintptr, cx, cy, size float64, argb uint32, now time.Time) {
	if g == 0 || size <= 2 {
		return
	}
	stroke := math.Max(1.8, size*0.16)
	var pen uintptr
	_, _, _ = diProcGdipCreatePen1.Call(uintptr(argb), uintptr(math.Float32bits(float32(stroke))), 2, uintptr(unsafe.Pointer(&pen)))
	if pen == 0 {
		return
	}
	defer diProcGdipDeletePen.Call(pen)
	_, _, _ = diProcGdipSetPenStartCap.Call(pen, 2)
	_, _, _ = diProcGdipSetPenEndCap.Call(pen, 2)
	angle := math.Mod(float64(now.UnixNano())/1e9*360.0*1.35, 360.0)
	r := size / 2.0
	_, _, _ = diProcGdipDrawArc.Call(
		g, pen,
		uintptr(math.Float32bits(float32(cx-r))),
		uintptr(math.Float32bits(float32(cy-r))),
		uintptr(math.Float32bits(float32(size))),
		uintptr(math.Float32bits(float32(size))),
		uintptr(math.Float32bits(float32(angle))),
		uintptr(math.Float32bits(260.0)),
	)
}

func drawVeloraPlayGlyphGdip(g uintptr, cx, cy, size float64, playing, hovered bool, op float64, pal veloraBrandPalette) {
	col := pal.TextMain
	if playing {
		col = pal.Accent
	} else if hovered {
		col = pal.Primary
	}
	var brush uintptr
	_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(col, op)), uintptr(unsafe.Pointer(&brush)))
	if brush == 0 {
		return
	}
	defer diProcGdipDeleteBrush.Call(brush)

	s := size / 24.0
	ox := cx - size/2.0
	oy := cy - size/2.0

	if playing {
		b1 := createRoundedRectPath(ox+6.5*s, oy+5*s, 3.8*s, 14*s, 1.5*s)
		if b1 != 0 {
			_, _, _ = diProcGdipFillPath.Call(g, brush, b1)
			_, _, _ = diProcGdipDeletePath.Call(b1)
		}
		b2 := createRoundedRectPath(ox+13.7*s, oy+5*s, 3.8*s, 14*s, 1.5*s)
		if b2 != 0 {
			_, _, _ = diProcGdipFillPath.Call(g, brush, b2)
			_, _, _ = diProcGdipDeletePath.Call(b2)
		}
	} else {
		var tri uintptr
		_, _, _ = diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&tri)))
		if tri != 0 {
			addLineF(tri, ox+7.5*s, oy+4.5*s, ox+19.5*s, oy+12*s)
			addLineF(tri, ox+19.5*s, oy+12*s, ox+7.5*s, oy+19.5*s)
			_, _, _ = diProcGdipClosePathFigure.Call(tri)
			_, _, _ = diProcGdipFillPath.Call(g, brush, tri)
			_, _, _ = diProcGdipDeletePath.Call(tri)
		}
	}
}

func drawPhosphorQueueGdip(g uintptr, cx, cy, size float64, hovered bool, op float64, pal veloraBrandPalette) {
	col := pal.TextMain
	if hovered {
		col = pal.Primary
	}
	s := size / 24.0
	ox := cx - size/2.0
	oy := cy - size/2.0

	var pen uintptr
	_, _, _ = diProcGdipCreatePen1.Call(uintptr(rgbaToARGB(col, op)), uintptr(math.Float32bits(float32(2.1*s))), 2, uintptr(unsafe.Pointer(&pen)))
	if pen == 0 {
		return
	}
	defer diProcGdipDeletePen.Call(pen)
	_, _, _ = diProcGdipSetPenStartCap.Call(pen, 2)
	_, _, _ = diProcGdipSetPenEndCap.Call(pen, 2)

	for _, yOff := range []float64{6.5, 12.0, 17.5} {
		_, _, _ = diProcGdipDrawLine.Call(g, pen,
			uintptr(math.Float32bits(float32(ox+4.5*s))), uintptr(math.Float32bits(float32(oy+yOff*s))),
			uintptr(math.Float32bits(float32(ox+19.5*s))), uintptr(math.Float32bits(float32(oy+yOff*s))))
	}
}

func drawArrowCircleGdip(g uintptr, cx, cy, r, u, op float64, pal veloraBrandPalette, hovered bool) {
	bgCol := rgbaToARGB(pal.Primary, 0.16*op)
	fgCol := rgbaToARGB(pal.Primary, op)
	if hovered {
		bgCol = rgbaToARGB(pal.Accent, 0.32*op)
		fgCol = rgbaToARGB(pal.Accent, op)
	}
	drawFilledPillGdip(g, cx-r, cy-r, r*2, r*2, r, bgCol)

	var pen uintptr
	_, _, _ = diProcGdipCreatePen1.Call(uintptr(fgCol), uintptr(math.Float32bits(float32(2.1*u))), 2, uintptr(unsafe.Pointer(&pen)))
	if pen == 0 {
		return
	}
	defer diProcGdipDeletePen.Call(pen)
	_, _, _ = diProcGdipSetPenStartCap.Call(pen, 2)
	_, _, _ = diProcGdipSetPenEndCap.Call(pen, 2)
	_, _, _ = diProcGdipSetPenLineJoin.Call(pen, 2)

	var path uintptr
	_, _, _ = diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&path)))
	if path == 0 {
		return
	}
	defer diProcGdipDeletePath.Call(path)
	s := (9.0 / 10.0) * u
	ox := cx - 4.5*u
	oy := cy - 4.5*u
	addLineF(path, ox+3.5*s, oy+1.5*s, ox+7.0*s, oy+5.0*s)
	addLineF(path, ox+7.0*s, oy+5.0*s, ox+3.5*s, oy+8.5*s)
	_, _, _ = diProcGdipDrawPath.Call(g, pen, path)
}

func drawPrevSvgGdip(g uintptr, cx, cy, size float64, hovered bool, op float64, pal veloraBrandPalette) {
	col := pal.TextMain
	if hovered {
		col = pal.Primary
	}
	var brush uintptr
	_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(col, op)), uintptr(unsafe.Pointer(&brush)))
	if brush == 0 {
		return
	}
	defer diProcGdipDeleteBrush.Call(brush)

	var path uintptr
	_, _, _ = diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&path)))
	if path == 0 {
		return
	}
	defer diProcGdipDeletePath.Call(path)

	s := size / 24.0
	ox := cx - size/2.0
	oy := cy - size/2.0

	_, _, _ = diProcGdipStartPathFigure.Call(path)
	addLineF(path, ox+11*s, oy+6*s, ox+3*s, oy+12*s)
	addLineF(path, ox+3*s, oy+12*s, ox+11*s, oy+18*s)
	_, _, _ = diProcGdipClosePathFigure.Call(path)

	_, _, _ = diProcGdipStartPathFigure.Call(path)
	addLineF(path, ox+21*s, oy+6*s, ox+13*s, oy+12*s)
	addLineF(path, ox+13*s, oy+12*s, ox+21*s, oy+18*s)
	_, _, _ = diProcGdipClosePathFigure.Call(path)

	_, _, _ = diProcGdipFillPath.Call(g, brush, path)
}

func drawNextSvgGdip(g uintptr, cx, cy, size float64, hovered bool, op float64, pal veloraBrandPalette) {
	col := pal.TextMain
	if hovered {
		col = pal.Primary
	}
	var brush uintptr
	_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(col, op)), uintptr(unsafe.Pointer(&brush)))
	if brush == 0 {
		return
	}
	defer diProcGdipDeleteBrush.Call(brush)

	var path uintptr
	_, _, _ = diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&path)))
	if path == 0 {
		return
	}
	defer diProcGdipDeletePath.Call(path)

	s := size / 24.0
	ox := cx - size/2.0
	oy := cy - size/2.0

	_, _, _ = diProcGdipStartPathFigure.Call(path)
	addLineF(path, ox+13*s, oy+6*s, ox+21*s, oy+12*s)
	addLineF(path, ox+21*s, oy+12*s, ox+13*s, oy+18*s)
	_, _, _ = diProcGdipClosePathFigure.Call(path)

	_, _, _ = diProcGdipStartPathFigure.Call(path)
	addLineF(path, ox+3*s, oy+6*s, ox+11*s, oy+12*s)
	addLineF(path, ox+11*s, oy+12*s, ox+3*s, oy+18*s)
	_, _, _ = diProcGdipClosePathFigure.Call(path)

	_, _, _ = diProcGdipFillPath.Call(g, brush, path)
}

func drawVeloraPlayButtonGdip(g uintptr, cx, cy, btnSize float64, playing, hovered bool, op float64, pal veloraBrandPalette) {
	bgCol := pal.Accent
	if hovered {
		bgCol = lerpColor(pal.Accent, pal.Primary, 0.35)
	}
	drawFilledPillGdip(g, cx-btnSize/2.0, cy-btnSize/2.0, btnSize, btnSize, btnSize/2.0, rgbaToARGB(bgCol, op))

	iconSize := btnSize * 0.56
	var brush uintptr
	_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(pal.BgDeep, op)), uintptr(unsafe.Pointer(&brush)))
	if brush == 0 {
		return
	}
	defer diProcGdipDeleteBrush.Call(brush)

	var path uintptr
	_, _, _ = diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&path)))
	if path == 0 {
		return
	}
	defer diProcGdipDeletePath.Call(path)

	s := iconSize / 24.0
	ox := cx - iconSize/2.0
	oy := cy - iconSize/2.0

	if playing {
		_, _, _ = diProcGdipStartPathFigure.Call(path)
		addLineF(path, ox+6.5*s, oy+5*s, ox+10.5*s, oy+5*s)
		addLineF(path, ox+10.5*s, oy+5*s, ox+10.5*s, oy+19*s)
		addLineF(path, ox+10.5*s, oy+19*s, ox+6.5*s, oy+19*s)
		_, _, _ = diProcGdipClosePathFigure.Call(path)

		_, _, _ = diProcGdipStartPathFigure.Call(path)
		addLineF(path, ox+13.5*s, oy+5*s, ox+17.5*s, oy+5*s)
		addLineF(path, ox+17.5*s, oy+5*s, ox+17.5*s, oy+19*s)
		addLineF(path, ox+17.5*s, oy+19*s, ox+13.5*s, oy+19*s)
		_, _, _ = diProcGdipClosePathFigure.Call(path)
	} else {
		_, _, _ = diProcGdipStartPathFigure.Call(path)
		addLineF(path, ox+8*s, oy+5*s, ox+19.5*s, oy+12*s)
		addLineF(path, ox+19.5*s, oy+12*s, ox+8*s, oy+19*s)
		_, _, _ = diProcGdipClosePathFigure.Call(path)
	}
	_, _, _ = diProcGdipFillPath.Call(g, brush, path)
}

func drawHeartSvgGdip(g uintptr, cx, cy, size float64, isFav, hovered bool, op float64, pal veloraBrandPalette) {
	s := size / 24.0
	ox := cx - size/2.0
	oy := cy - size/2.0

	var path uintptr
	_, _, _ = diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&path)))
	if path == 0 {
		return
	}
	defer diProcGdipDeletePath.Call(path)

	addBezierF(path, ox+12*s, oy+20.5*s, ox+4*s, oy+15*s, ox+2*s, oy+10.5*s, ox+2*s, oy+7.2*s)
	addBezierF(path, ox+2*s, oy+7.2*s, ox+2*s, oy+4.2*s, ox+4.4*s, oy+2*s, ox+7.3*s, oy+2*s)
	addBezierF(path, ox+7.3*s, oy+2*s, ox+9.4*s, oy+2*s, ox+11.1*s, oy+3.2*s, ox+12*s, oy+4.8*s)
	addBezierF(path, ox+12*s, oy+4.8*s, ox+12.9*s, oy+3.2*s, ox+14.6*s, oy+2*s, ox+16.7*s, oy+2*s)
	addBezierF(path, ox+16.7*s, oy+2*s, ox+19.6*s, oy+2*s, ox+22*s, oy+4.2*s, ox+22*s, oy+7.2*s)
	addBezierF(path, ox+22*s, oy+7.2*s, ox+22*s, oy+10.5*s, ox+20*s, oy+15*s, ox+12*s, oy+20.5*s)
	_, _, _ = diProcGdipClosePathFigure.Call(path)

	if isFav {
		var brush uintptr
		_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(pal.Accent, op)), uintptr(unsafe.Pointer(&brush)))
		if brush != 0 {
			_, _, _ = diProcGdipFillPath.Call(g, brush, path)
			_, _, _ = diProcGdipDeleteBrush.Call(brush)
		}
	} else {
		col := pal.TextVariant
		if hovered {
			col = pal.Accent
		}
		var pen uintptr
		_, _, _ = diProcGdipCreatePen1.Call(uintptr(rgbaToARGB(col, 0.85*op)), uintptr(math.Float32bits(float32(2.0*s))), 2, uintptr(unsafe.Pointer(&pen)))
		if pen != 0 {
			_, _, _ = diProcGdipSetPenLineJoin.Call(pen, 2)
			_, _, _ = diProcGdipDrawPath.Call(g, pen, path)
			_, _, _ = diProcGdipDeletePen.Call(pen)
		}
	}
}

func drawShuffleSvgGdip(g uintptr, cx, cy, size float64, active, hovered bool, op float64, pal veloraBrandPalette) {
	col := pal.TextVariant
	alpha := 0.65 * op
	if active {
		col = pal.Accent
		alpha = op
	} else if hovered {
		col = pal.Primary
		alpha = 0.95 * op
	}
	s := size / 24.0
	ox := cx - size/2.0
	oy := cy - size/2.0

	var pen uintptr
	_, _, _ = diProcGdipCreatePen1.Call(uintptr(rgbaToARGB(col, alpha)), uintptr(math.Float32bits(float32(2.1*s))), 2, uintptr(unsafe.Pointer(&pen)))
	if pen == 0 {
		return
	}
	defer diProcGdipDeletePen.Call(pen)
	_, _, _ = diProcGdipSetPenStartCap.Call(pen, 2)
	_, _, _ = diProcGdipSetPenEndCap.Call(pen, 2)
	_, _, _ = diProcGdipSetPenLineJoin.Call(pen, 2)

	_, _, _ = diProcGdipDrawLine.Call(g, pen,
		uintptr(math.Float32bits(float32(ox+4*s))), uintptr(math.Float32bits(float32(oy+6*s))),
		uintptr(math.Float32bits(float32(ox+20*s))), uintptr(math.Float32bits(float32(oy+18*s))))
	_, _, _ = diProcGdipDrawLine.Call(g, pen,
		uintptr(math.Float32bits(float32(ox+4*s))), uintptr(math.Float32bits(float32(oy+18*s))),
		uintptr(math.Float32bits(float32(ox+20*s))), uintptr(math.Float32bits(float32(oy+6*s))))
	_, _, _ = diProcGdipDrawLine.Call(g, pen,
		uintptr(math.Float32bits(float32(ox+16*s))), uintptr(math.Float32bits(float32(oy+6*s))),
		uintptr(math.Float32bits(float32(ox+20*s))), uintptr(math.Float32bits(float32(oy+6*s))))
	_, _, _ = diProcGdipDrawLine.Call(g, pen,
		uintptr(math.Float32bits(float32(ox+16*s))), uintptr(math.Float32bits(float32(oy+18*s))),
		uintptr(math.Float32bits(float32(ox+20*s))), uintptr(math.Float32bits(float32(oy+18*s))))
}

func drawRepeatSvgGdip(g uintptr, cx, cy, size float64, mode string, hovered bool, op float64, pal veloraBrandPalette) {
	col := pal.TextVariant
	alpha := 0.65 * op
	if mode == "all" {
		col = pal.Primary
		alpha = op
	} else if mode == "one" {
		col = pal.Accent
		alpha = op
	} else if hovered {
		col = pal.Primary
		alpha = 0.95 * op
	}

	s := size / 24.0
	ox := cx - size/2.0
	oy := cy - size/2.0

	var pen uintptr
	_, _, _ = diProcGdipCreatePen1.Call(uintptr(rgbaToARGB(col, alpha)), uintptr(math.Float32bits(float32(2.0*s))), 2, uintptr(unsafe.Pointer(&pen)))
	if pen == 0 {
		return
	}
	defer diProcGdipDeletePen.Call(pen)

	path := createRoundedRectPath(ox+4*s, oy+6*s, 16*s, 12*s, 4*s)
	if path != 0 {
		_, _, _ = diProcGdipDrawPath.Call(g, pen, path)
		_, _, _ = diProcGdipDeletePath.Call(path)
	}
	if mode == "one" {
		var brush uintptr
		_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(pal.Accent, op)), uintptr(unsafe.Pointer(&brush)))
		if brush != 0 {
			fillCircleGdip(g, brush, cx, cy, 2.2*s)
			_, _, _ = diProcGdipDeleteBrush.Call(brush)
		}
	}
}

func drawSpeakerSvgGdip(g uintptr, cx, cy, size float64, muted bool, op float64, argb uint32) {
	s := size / 24.0
	ox := cx - size/2.0
	oy := cy - size/2.0

	var brush uintptr
	_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(argbWithOp(argb, op)), uintptr(unsafe.Pointer(&brush)))
	if brush == 0 {
		return
	}
	defer diProcGdipDeleteBrush.Call(brush)

	var path uintptr
	_, _, _ = diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&path)))
	if path == 0 {
		return
	}
	defer diProcGdipDeletePath.Call(path)

	addLineF(path, ox+3*s, oy+9*s, ox+7*s, oy+9*s)
	addLineF(path, ox+7*s, oy+9*s, ox+12*s, oy+5*s)
	addLineF(path, ox+12*s, oy+5*s, ox+12*s, oy+19*s)
	addLineF(path, ox+12*s, oy+19*s, ox+7*s, oy+15*s)
	addLineF(path, ox+7*s, oy+15*s, ox+3*s, oy+15*s)
	_, _, _ = diProcGdipClosePathFigure.Call(path)
	_, _, _ = diProcGdipFillPath.Call(g, brush, path)

	var pen uintptr
	_, _, _ = diProcGdipCreatePen1.Call(uintptr(argbWithOp(argb, op)), uintptr(math.Float32bits(float32(2.0*s))), 2, uintptr(unsafe.Pointer(&pen)))
	if pen != 0 {
		_, _, _ = diProcGdipSetPenStartCap.Call(pen, 2)
		_, _, _ = diProcGdipSetPenEndCap.Call(pen, 2)
		if muted {
			_, _, _ = diProcGdipDrawLine.Call(g, pen,
				uintptr(math.Float32bits(float32(ox+16*s))), uintptr(math.Float32bits(float32(oy+9*s))),
				uintptr(math.Float32bits(float32(ox+21*s))), uintptr(math.Float32bits(float32(oy+15*s))))
			_, _, _ = diProcGdipDrawLine.Call(g, pen,
				uintptr(math.Float32bits(float32(ox+21*s))), uintptr(math.Float32bits(float32(oy+9*s))),
				uintptr(math.Float32bits(float32(ox+16*s))), uintptr(math.Float32bits(float32(oy+15*s))))
		} else {
			_, _, _ = diProcGdipDrawArc.Call(g, pen,
				uintptr(math.Float32bits(float32(ox+11*s))), uintptr(math.Float32bits(float32(oy+7*s))),
				uintptr(math.Float32bits(float32(8*s))), uintptr(math.Float32bits(float32(10*s))),
				uintptr(math.Float32bits(-45.0)), uintptr(math.Float32bits(90.0)))
		}
		_, _, _ = diProcGdipDeletePen.Call(pen)
	}
}

func drawBubbleSvgGdip(g uintptr, cx, cy, u float64, filled, hovered bool, op float64, pal veloraBrandPalette) {
	sx := (17.0 / 18.0) * u
	sy := (15.0 / 16.0) * u
	ox := cx - 8.5*u
	oy := cy - 7.5*u

	var path uintptr
	_, _, _ = diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&path)))
	if path == 0 {
		return
	}
	defer diProcGdipDeletePath.Call(path)

	addArcF(path, ox+0.8*sx, oy+0.8*sy, 6.4*sx, 6.4*sy, 180, 90)
	addArcF(path, ox+10.8*sx, oy+0.8*sy, 6.4*sx, 6.4*sy, 270, 90)
	addArcF(path, ox+10.8*sx, oy+5.4*sy, 6.4*sx, 6.4*sy, 0, 90)
	addLineF(path, ox+14.0*sx, oy+11.8*sy, ox+8.4*sx, oy+11.8*sy)
	addLineF(path, ox+8.4*sx, oy+11.8*sy, ox+4.6*sx, oy+15.0*sy)
	addLineF(path, ox+4.6*sx, oy+15.0*sy, ox+5.1*sx, oy+11.8*sy)
	addArcF(path, ox+0.8*sx, oy+5.4*sy, 6.4*sx, 6.4*sy, 90, 90)
	_, _, _ = diProcGdipClosePathFigure.Call(path)

	dr := 1.3 * sx
	if filled {
		var wBrush, bBrush uintptr
		_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(pal.Accent, op)), uintptr(unsafe.Pointer(&wBrush)))
		_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(pal.BgDeep, op)), uintptr(unsafe.Pointer(&bBrush)))
		if wBrush != 0 {
			_, _, _ = diProcGdipFillPath.Call(g, wBrush, path)
			_, _, _ = diProcGdipDeleteBrush.Call(wBrush)
		}
		if bBrush != 0 {
			fillCircleGdip(g, bBrush, ox+6.3*sx, oy+5.4*sy, dr)
			fillCircleGdip(g, bBrush, ox+11.3*sx, oy+5.4*sy, dr)
			_, _, _ = diProcGdipDeleteBrush.Call(bBrush)
		}
	} else {
		col := pal.TextVariant
		if hovered {
			col = pal.Accent
		}
		argb := rgbaToARGB(col, 0.85*op)
		var pen, brush uintptr
		_, _, _ = diProcGdipCreatePen1.Call(uintptr(argb), uintptr(math.Float32bits(float32(1.35*u))), 2, uintptr(unsafe.Pointer(&pen)))
		_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(argb), uintptr(unsafe.Pointer(&brush)))
		if pen != 0 {
			_, _, _ = diProcGdipSetPenLineJoin.Call(pen, 2)
			_, _, _ = diProcGdipDrawPath.Call(g, pen, path)
			_, _, _ = diProcGdipDeletePen.Call(pen)
		}
		if brush != 0 {
			fillCircleGdip(g, brush, ox+6.3*sx, oy+5.4*sy, dr)
			fillCircleGdip(g, brush, ox+11.3*sx, oy+5.4*sy, dr)
			_, _, _ = diProcGdipDeleteBrush.Call(brush)
		}
	}
}

func fillCircleGdip(g, brush uintptr, cx, cy, r float64) {
	_, _, _ = diProcGdipFillEllipse.Call(
		g,
		brush,
		uintptr(math.Float32bits(float32(cx-r))),
		uintptr(math.Float32bits(float32(cy-r))),
		uintptr(math.Float32bits(float32(r*2))),
		uintptr(math.Float32bits(float32(r*2))),
	)
}

// closedPillCoverAndRingGeom computes symmetrical geometry for the compact cover disc and progress ring.
func closedPillCoverAndRingGeom(nx, w, h, u float64) (miniPx, topCov, leftCov int, ringCx, ringCy, strokeW, ringR float64) {
	miniPx = maxInt(21, int(math.Round(20.8*u)))
	topCov = int(math.Round((h/2.0)*u - float64(miniPx)/2.0))
	leftCov = int(math.Round((nx+20.0)*u - float64(miniPx)/2.0))
	leftCxIdx := float64(leftCov) + float64(miniPx-1)/2.0
	pillMidIdx := (nx+w/2.0)*u - 0.5
	ringCx = 2.0*pillMidIdx - leftCxIdx
	ringCy = float64(topCov) + float64(miniPx-1)/2.0
	strokeW = 2.8 * u
	ringR = (float64(miniPx) - strokeW) / 2.0
	return miniPx, topCov, leftCov, ringCx, ringCy, strokeW, ringR
}

func drawPkRingGdip(g uintptr, cx, cy, r, strokeW, prog, op float64, fgRGB, bgARGB uint32) {
	var bgPen, fgPen uintptr
	_, _, _ = diProcGdipCreatePen1.Call(uintptr(argbWithOp(bgARGB, op)), uintptr(math.Float32bits(float32(strokeW))), 2, uintptr(unsafe.Pointer(&bgPen)))
	if bgPen != 0 {
		_, _, _ = diProcGdipDrawEllipse.Call(
			g,
			bgPen,
			uintptr(math.Float32bits(float32(cx-r))),
			uintptr(math.Float32bits(float32(cy-r))),
			uintptr(math.Float32bits(float32(r*2))),
			uintptr(math.Float32bits(float32(r*2))),
		)
		_, _, _ = diProcGdipDeletePen.Call(bgPen)
	}
	if prog > 0.005 {
		fgARGB := argbWithOp(0xFF000000|fgRGB, op)
		_, _, _ = diProcGdipCreatePen1.Call(uintptr(fgARGB), uintptr(math.Float32bits(float32(strokeW))), 2, uintptr(unsafe.Pointer(&fgPen)))
		if fgPen != 0 {
			_, _, _ = diProcGdipSetPenStartCap.Call(fgPen, 2)
			_, _, _ = diProcGdipSetPenEndCap.Call(fgPen, 2)
			_, _, _ = diProcGdipDrawArc.Call(
				g,
				fgPen,
				uintptr(math.Float32bits(float32(cx-r))),
				uintptr(math.Float32bits(float32(cy-r))),
				uintptr(math.Float32bits(float32(r*2))),
				uintptr(math.Float32bits(float32(r*2))),
				uintptr(math.Float32bits(-90.0)),
				uintptr(math.Float32bits(float32(360.0*prog))),
			)
			_, _, _ = diProcGdipDeletePen.Call(fgPen)
		}
	}
}

func drawBluetoothSvgGdip(g uintptr, cx, cy, size, op float64, argb uint32) {
	s := size / 24.0
	ox := cx - size/2.0
	oy := cy - size/2.0

	var pen uintptr
	_, _, _ = diProcGdipCreatePen1.Call(uintptr(argbWithOp(argb, op)), uintptr(math.Float32bits(float32(2.4*s))), 2, uintptr(unsafe.Pointer(&pen)))
	if pen == 0 {
		return
	}
	defer diProcGdipDeletePen.Call(pen)
	_, _, _ = diProcGdipSetPenStartCap.Call(pen, 2)
	_, _, _ = diProcGdipSetPenEndCap.Call(pen, 2)
	_, _, _ = diProcGdipSetPenLineJoin.Call(pen, 2)

	var path uintptr
	_, _, _ = diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&path)))
	if path == 0 {
		return
	}
	defer diProcGdipDeletePath.Call(path)

	addLineF(path, ox+7*s, oy+7*s, ox+17*s, oy+17*s)
	addLineF(path, ox+17*s, oy+17*s, ox+12*s, oy+21*s)
	addLineF(path, ox+12*s, oy+21*s, ox+12*s, oy+3*s)
	addLineF(path, ox+12*s, oy+3*s, ox+17*s, oy+7*s)
	addLineF(path, ox+17*s, oy+7*s, ox+7*s, oy+17*s)
	_, _, _ = diProcGdipDrawPath.Call(g, pen, path)
}

func drawVeloraHeadphonesBadgeGdip(g uintptr, cx, cy, u, op float64, pal veloraBrandPalette, hovered bool) {
	ringCol := pal.Primary
	if hovered {
		ringCol = pal.Accent
	}
	drawPkRingGdip(g, cx, cy, 22.0*u, 5.0*u, 0, op, 0, rgbaToARGB(ringCol, 0.10))
	drawPkRingGdip(g, cx, cy, 19.5*u, 1.5*u, 0, op, 0, rgbaToARGB(ringCol, 0.38))
	drawFilledPillGdip(g, cx-15.0*u, cy-15.0*u, 30.0*u, 30.0*u, 15.0*u, rgbaToARGB(pal.PrimaryDim, 0.65*op))
	drawPkRingGdip(g, cx, cy, 14.5*u, 1.5*u, 0, op, 0, rgbaToARGB(pal.Accent, 0.65))

	size := 15.5 * u
	s := size / 24.0
	ox := cx - size/2.0
	oy := cy - size/2.0

	var pen uintptr
	_, _, _ = diProcGdipCreatePen1.Call(uintptr(rgbaToARGB(pal.TextMain, op)), uintptr(math.Float32bits(float32(2.0*s))), 2, uintptr(unsafe.Pointer(&pen)))
	if pen == 0 {
		return
	}
	defer diProcGdipDeletePen.Call(pen)
	_, _, _ = diProcGdipSetPenStartCap.Call(pen, 2)
	_, _, _ = diProcGdipSetPenEndCap.Call(pen, 2)
	_, _, _ = diProcGdipSetPenLineJoin.Call(pen, 2)

	var path uintptr
	_, _, _ = diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&path)))
	if path == 0 {
		return
	}
	defer diProcGdipDeletePath.Call(path)

	_, _, _ = diProcGdipStartPathFigure.Call(path)
	addLineF(path, ox+4*s, oy+15*s, ox+4*s, oy+12*s)
	addArcF(path, ox+4*s, oy+4*s, 16*s, 16*s, 180, 180)
	addLineF(path, ox+20*s, oy+12*s, ox+20*s, oy+15*s)

	_, _, _ = diProcGdipStartPathFigure.Call(path)
	addLineF(path, ox+4*s, oy+15*s, ox+7*s, oy+15*s)
	addLineF(path, ox+7*s, oy+15*s, ox+7*s, oy+20*s)
	addLineF(path, ox+7*s, oy+20*s, ox+4*s, oy+20*s)
	_, _, _ = diProcGdipClosePathFigure.Call(path)

	_, _, _ = diProcGdipStartPathFigure.Call(path)
	addLineF(path, ox+20*s, oy+15*s, ox+17*s, oy+15*s)
	addLineF(path, ox+17*s, oy+15*s, ox+17*s, oy+20*s)
	addLineF(path, ox+17*s, oy+20*s, ox+20*s, oy+20*s)
	_, _, _ = diProcGdipClosePathFigure.Call(path)

	_, _, _ = diProcGdipDrawPath.Call(g, pen, path)
}

func formatEqPresetLabel(eq string) string {
	switch strings.ToLower(strings.TrimSpace(eq)) {
	case "bass":
		return "Deep Bass EQ"
	case "vocal":
		return "Vocal Clarity EQ"
	case "electronic":
		return "Electronic EQ"
	case "flat":
		return "Flat Studio EQ"
	default:
		return "Spotify Studio EQ"
	}
}

func (c *dynamicIslandController) drawPhosphorGlyphGdip(g uintptr, filled bool, codepoint rune, cx, cy, sizePx float64, argb uint32) {
	fam := c.famPhosphorReg
	if filled && c.famPhosphorFill != 0 {
		fam = c.famPhosphorFill
	}
	if fam == 0 {
		return
	}

	var path uintptr
	_, _, _ = diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&path)))
	if path == 0 {
		return
	}
	defer diProcGdipDeletePath.Call(path)

	u16 := [2]uint16{uint16(codepoint), 0}
	rcLayout := diRectF{X: 0, Y: 0, Width: float32(sizePx * 2.0), Height: float32(sizePx * 2.0)}
	st, _, _ := diProcGdipAddPathString.Call(
		path,
		uintptr(unsafe.Pointer(&u16[0])),
		1,
		fam,
		0,
		uintptr(math.Float32bits(float32(sizePx))),
		uintptr(unsafe.Pointer(&rcLayout)),
		0,
	)
	if st != 0 {
		return
	}

	var bounds diRectF
	_, _, _ = diProcGdipGetPathWorldBounds.Call(path, uintptr(unsafe.Pointer(&bounds)), 0, 0)
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return
	}

	pathCenterX := float64(bounds.X) + float64(bounds.Width)/2.0
	pathCenterY := float64(bounds.Y) + float64(bounds.Height)/2.0
	targetX := cx
	if codepoint == 0xE3D0 {
		targetX += sizePx * 0.065
	}
	dx := float32(targetX - pathCenterX)
	dy := float32(cy - pathCenterY)

	var mat uintptr
	_, _, _ = diProcGdipCreateMatrix2.Call(
		uintptr(math.Float32bits(1.0)),
		uintptr(math.Float32bits(0.0)),
		uintptr(math.Float32bits(0.0)),
		uintptr(math.Float32bits(1.0)),
		uintptr(math.Float32bits(dx)),
		uintptr(math.Float32bits(dy)),
		uintptr(unsafe.Pointer(&mat)),
	)
	if mat != 0 {
		_, _, _ = diProcGdipTransformPath.Call(path, mat)
		_, _, _ = diProcGdipDeleteMatrix.Call(mat)
	}

	var brush uintptr
	_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(argb), uintptr(unsafe.Pointer(&brush)))
	if brush == 0 {
		return
	}
	defer diProcGdipDeleteBrush.Call(brush)
	_, _, _ = diProcGdipFillPath.Call(g, brush, path)
}

func computeLineSweepProgressFromLine(line sources.LyricLine, nextStartSec float64, hasNext bool, currentTimeSec float64) float64 {
	if len(line.Words) > 0 {
		eff := currentTimeSec + 0.05
		totalW := 0.0
		litW := 0.0
		for _, w := range line.Words {
			weight := math.Max(1.0, float64(len([]rune(w.Text)))+0.5)
			totalW += weight
			dur := math.Max(0.12, w.DurationSec)
			if w.DurationSec <= 0 {
				dur = 0.35
			}
			litW += weight * clamp01((eff-w.TimeSec)/dur)
		}
		if totalW > 0 {
			return clamp01(litW / totalW)
		}
	}
	return computeLyricSweepProgress(line.Text, line.TimeSec, nextStartSec, hasNext, currentTimeSec)
}

func computeLyricSweepProgress(lineText string, startSec, nextStartSec float64, hasNext bool, currentTimeSec float64) float64 {
	words := strings.Fields(strings.TrimSpace(lineText))
	wordCount := len(words)
	if wordCount < 1 {
		wordCount = 1
	}
	charCount := 0
	for _, w := range words {
		charCount += len([]rune(w))
	}

	var singDur float64
	if hasNext && nextStartSec > startSec {
		gap := math.Max(0.35, nextStartSec-startSec)
		maxVocalDur := math.Max(2.2, float64(wordCount)*0.45+float64(charCount)*0.075)
		singDur = math.Min(gap*0.92, maxVocalDur)
		if gap < 0.8 {
			singDur = math.Max(0.35, gap*0.9)
		}
	} else {
		singDur = math.Max(2.0, math.Min(5.0, float64(wordCount)*0.35+float64(charCount)*0.06))
	}

	elapsed := currentTimeSec + 0.05 - startSec
	return clamp01(elapsed / singDur)
}

func (c *dynamicIslandController) drawCenteredVectorTextGdip(
	g uintptr,
	fam uintptr,
	txt string,
	cx, cy, targetHeightPx, maxWidthPx float64,
	argb uint32,
) {
	c.drawCenteredVectorTextSweepGdip(g, fam, txt, cx, cy, targetHeightPx, maxWidthPx, argb, argb, 1.0, false, 0, 0, 0, 0)
}

func (c *dynamicIslandController) drawCenteredVectorTextSweepGdip(
	g uintptr,
	fam uintptr,
	txt string,
	cx, cy, targetHeightPx, maxWidthPx float64,
	litARGB, dimARGB uint32,
	sweepProg float64,
	restoreClip bool,
	clipX, clipY, clipW, clipH float64,
) {
	if g == 0 || fam == 0 || txt == "" || targetHeightPx <= 1.0 {
		return
	}
	var path uintptr
	st, _, _ := diProcGdipCreatePath.Call(0, uintptr(unsafe.Pointer(&path)))
	if st != 0 || path == 0 {
		return
	}
	defer diProcGdipDeletePath.Call(path)

	u16, _ := windows.UTF16FromString(txt)
	if len(u16) <= 1 {
		return
	}
	rcOrigin := diRectF{X: 0, Y: 0, Width: 0, Height: 0}
	_, _, _ = diProcGdipAddPathString.Call(
		path,
		uintptr(unsafe.Pointer(&u16[0])),
		uintptr(len(u16)-1),
		fam,
		0,
		uintptr(math.Float32bits(100.0)),
		uintptr(unsafe.Pointer(&rcOrigin)),
		0,
	)

	var bounds diRectF
	stB, _, _ := diProcGdipGetPathWorldBounds.Call(path, uintptr(unsafe.Pointer(&bounds)), 0, 0)
	if stB != 0 || bounds.Height <= 0.1 || bounds.Width <= 0.1 {
		return
	}

	scale := targetHeightPx / float64(bounds.Height)
	if maxWidthPx > 0 && float64(bounds.Width)*scale > maxWidthPx {
		scale = maxWidthPx / float64(bounds.Width)
	}

	bcx := float64(bounds.X) + float64(bounds.Width)/2.0
	bcy := float64(bounds.Y) + float64(bounds.Height)/2.0
	tx := cx - bcx*scale
	ty := cy - bcy*scale

	var matrix uintptr
	stM, _, _ := diProcGdipCreateMatrix2.Call(
		uintptr(math.Float32bits(float32(scale))),
		0,
		0,
		uintptr(math.Float32bits(float32(scale))),
		uintptr(math.Float32bits(float32(tx))),
		uintptr(math.Float32bits(float32(ty))),
		uintptr(unsafe.Pointer(&matrix)),
	)
	if stM == 0 && matrix != 0 {
		_, _, _ = diProcGdipTransformPath.Call(path, matrix)
		_, _, _ = diProcGdipDeleteMatrix.Call(matrix)
	}

	if dimARGB == litARGB || sweepProg >= 0.999 {
		var brush uintptr
		_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(litARGB), uintptr(unsafe.Pointer(&brush)))
		if brush != 0 {
			_, _, _ = diProcGdipFillPath.Call(g, brush, path)
			_, _, _ = diProcGdipDeleteBrush.Call(brush)
		}
		return
	}

	var dimBrush uintptr
	_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(dimARGB), uintptr(unsafe.Pointer(&dimBrush)))
	if dimBrush != 0 {
		_, _, _ = diProcGdipFillPath.Call(g, dimBrush, path)
		_, _, _ = diProcGdipDeleteBrush.Call(dimBrush)
	}

	if sweepProg <= 0.001 {
		return
	}

	leftPx := float64(bounds.X)*scale + tx
	topPx := float64(bounds.Y)*scale + ty
	wPx := float64(bounds.Width) * scale
	hPx := float64(bounds.Height) * scale
	_, _, _ = diProcGdipSetClipRect.Call(
		g,
		uintptr(math.Float32bits(float32(leftPx-2))),
		uintptr(math.Float32bits(float32(topPx-2))),
		uintptr(math.Float32bits(float32(wPx*sweepProg+2))),
		uintptr(math.Float32bits(float32(hPx+4))),
		1, // CombineModeIntersect
	)
	var litBrush uintptr
	_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(litARGB), uintptr(unsafe.Pointer(&litBrush)))
	if litBrush != 0 {
		_, _, _ = diProcGdipFillPath.Call(g, litBrush, path)
		_, _, _ = diProcGdipDeleteBrush.Call(litBrush)
	}
	if restoreClip {
		_, _, _ = diProcGdipSetClipRect.Call(
			g,
			uintptr(math.Float32bits(float32(clipX))),
			uintptr(math.Float32bits(float32(clipY))),
			uintptr(math.Float32bits(float32(clipW))),
			uintptr(math.Float32bits(float32(clipH))),
			0,
		)
	} else {
		_, _, _ = diProcGdipResetClip.Call(g)
	}
}

func (c *dynamicIslandController) drawVeloraTextGdip(
	g uintptr,
	nx, w, h, u float64,
	mode string,
	contentOp float64,
	mapX, mapXStatic, mapY, mapW, mapH func(float64) float64,
	title, artist string,
	pos, dur float64,
	playing bool,
	lyricsOpen bool,
	vol int,
	muted bool,
	eqPreset string,
	streakDays int,
	audioDev string,
	connType string,
	btBat int,
	sysBat int,
	isBt bool,
	audioOutputs []windowsAudioEndpoint,
	hpListOpen bool,
	hpListProg float64,
	lyricsLoading bool,
	lyricsHasPlain bool,
	lyrics []sources.LyricLine,
	lyricIdx int,
	prevLyricIdx int,
	lyricAnimProg float64,
	lyricCurY float64,
	lyricManualOffsetDp float64,
	hoverLyric int,
	closedLyricFrom string,
	closedLyricTo string,
	closedLyricProg float64,
	qTitle string,
	qIndex int,
	qScrollDp float64,
	qItems []DynamicIslandQueueItem,
	pal veloraBrandPalette,
	marqueeElapsed float64,
	diLang string,
	effLyricPos float64,
	lyricSweep bool,
) {
	if contentOp <= 0.02 {
		return
	}
	c.mu.Lock()
	showStatus := c.showStatus
	showCover := c.showCover
	showArtist := c.showArtist
	showLyric := c.showLyric
	c.mu.Unlock()
	diStr := getDIStrings(diLang)

	famReg := c.famRegular
	famSemi := c.famSemiBold
	if famReg == 0 {
		return
	}
	if famSemi == 0 {
		famSemi = famReg
	}

	var sfLeft, sfRight, sfCenter uintptr
	_, _, _ = diProcGdipCreateStringFormat.Call(0, 0, uintptr(unsafe.Pointer(&sfLeft)))
	if sfLeft != 0 {
		_, _, _ = diProcGdipSetStringFormatTrim.Call(sfLeft, 3) // StringTrimmingEllipsisCharacter
		_, _, _ = diProcGdipSetStringFormatFlags.Call(sfLeft, 0x00001000)
		defer diProcGdipDeleteStringFormat.Call(sfLeft)
	}
	_, _, _ = diProcGdipCreateStringFormat.Call(0, 0, uintptr(unsafe.Pointer(&sfRight)))
	if sfRight != 0 {
		_, _, _ = diProcGdipSetStringFormatAlign.Call(sfRight, 2)
		_, _, _ = diProcGdipSetStringFormatFlags.Call(sfRight, 0x00001000)
		defer diProcGdipDeleteStringFormat.Call(sfRight)
	}
	_, _, _ = diProcGdipCreateStringFormat.Call(0, 0, uintptr(unsafe.Pointer(&sfCenter)))
	if sfCenter != 0 {
		_, _, _ = diProcGdipSetStringFormatAlign.Call(sfCenter, 1)
		_, _, _ = diProcGdipSetStringFormatFlags.Call(sfCenter, 0x00001000)
		defer diProcGdipDeleteStringFormat.Call(sfCenter)
	}
	var sfNoTrim uintptr
	_, _, _ = diProcGdipCreateStringFormat.Call(0, 0, uintptr(unsafe.Pointer(&sfNoTrim)))
	if sfNoTrim != 0 {
		_, _, _ = diProcGdipSetStringFormatTrim.Call(sfNoTrim, 0) // StringTrimmingNone
		_, _, _ = diProcGdipSetStringFormatFlags.Call(sfNoTrim, 0x00001000)
		defer diProcGdipDeleteStringFormat.Call(sfNoTrim)
	} else {
		sfNoTrim = sfLeft
	}

	var brushMain, brushVar, brushPrim, brushAcc uintptr
	_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(pal.TextMain, contentOp)), uintptr(unsafe.Pointer(&brushMain)))
	_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(pal.TextVariant, 0.85*contentOp)), uintptr(unsafe.Pointer(&brushVar)))
	_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(pal.Primary, contentOp)), uintptr(unsafe.Pointer(&brushPrim)))
	_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(pal.Accent, contentOp)), uintptr(unsafe.Pointer(&brushAcc)))
	for _, br := range []uintptr{brushMain, brushVar, brushPrim, brushAcc} {
		if br != 0 {
			defer diProcGdipDeleteBrush.Call(br)
		}
	}

	var f14Bold, f14Semi, f13Semi, f115Reg, f115Semi, f115Bold, f11Var, f10Bold, fLyrOn, fLyrOff uintptr
	_, _, _ = diProcGdipCreateFont.Call(famReg, uintptr(math.Float32bits(float32(14.5*u))), 1, 2, uintptr(unsafe.Pointer(&f14Bold)))
	_, _, _ = diProcGdipCreateFont.Call(famSemi, uintptr(math.Float32bits(float32(14.2*u))), 0, 2, uintptr(unsafe.Pointer(&f14Semi)))
	_, _, _ = diProcGdipCreateFont.Call(famSemi, uintptr(math.Float32bits(float32(13.2*u))), 0, 2, uintptr(unsafe.Pointer(&f13Semi)))
	_, _, _ = diProcGdipCreateFont.Call(famReg, uintptr(math.Float32bits(float32(11.5*u))), 0, 2, uintptr(unsafe.Pointer(&f115Reg)))
	_, _, _ = diProcGdipCreateFont.Call(famSemi, uintptr(math.Float32bits(float32(11.5*u))), 0, 2, uintptr(unsafe.Pointer(&f115Semi)))
	_, _, _ = diProcGdipCreateFont.Call(famReg, uintptr(math.Float32bits(float32(11.5*u))), 1, 2, uintptr(unsafe.Pointer(&f115Bold)))
	_, _, _ = diProcGdipCreateFont.Call(famReg, uintptr(math.Float32bits(float32(11.0*u))), 0, 2, uintptr(unsafe.Pointer(&f11Var)))
	_, _, _ = diProcGdipCreateFont.Call(famReg, uintptr(math.Float32bits(float32(10.0*u))), 1, 2, uintptr(unsafe.Pointer(&f10Bold)))
	_, _, _ = diProcGdipCreateFont.Call(famReg, uintptr(math.Float32bits(float32(14.0*u))), 1, 2, uintptr(unsafe.Pointer(&fLyrOn)))
	_, _, _ = diProcGdipCreateFont.Call(famSemi, uintptr(math.Float32bits(float32(12.5*u))), 0, 2, uintptr(unsafe.Pointer(&fLyrOff)))
	for _, fn := range []uintptr{f14Bold, f14Semi, f13Semi, f115Reg, f115Semi, f115Bold, f11Var, f10Bold, fLyrOn, fLyrOff} {
		if fn != 0 {
			defer diProcGdipDeleteFont.Call(fn)
		}
	}

	drawStrPhys := func(txt string, font, brush, sf uintptr, px, py, pw, ph float64) {
		if txt == "" || font == 0 || brush == 0 {
			return
		}
		u16, _ := windows.UTF16FromString(txt)
		rc := diRectF{X: float32(px), Y: float32(py), Width: float32(pw), Height: float32(ph)}
		_, _, _ = diProcGdipDrawString.Call(g, uintptr(unsafe.Pointer(&u16[0])), uintptr(len(u16)-1), font, uintptr(unsafe.Pointer(&rc)), sf, brush)
	}

	drawMarqueePhys := func(txt string, font, brush uintptr, col color.RGBA, colAlpha, px, py, pw, ph float64) {
		if txt == "" || font == 0 || brush == 0 || pw <= 2 {
			return
		}
		u16, _ := windows.UTF16FromString(txt)
		if len(u16) <= 1 {
			return
		}
		layoutRc := diRectF{X: 0, Y: 0, Width: float32(4000.0 * u), Height: float32(ph)}
		var boundRc diRectF
		_, _, _ = diProcGdipMeasureString.Call(
			g,
			uintptr(unsafe.Pointer(&u16[0])),
			uintptr(len(u16)-1),
			font,
			uintptr(unsafe.Pointer(&layoutRc)),
			sfNoTrim,
			uintptr(unsafe.Pointer(&boundRc)),
			0, 0,
		)
		if float64(boundRc.Width) <= pw {
			drawStrPhys(txt, font, brush, sfNoTrim, px, py, pw, ph)
			return
		}

		overflowPx := float64(boundRc.Width) - pw + 4.0*u
		scrollDur := math.Max(1.4, overflowPx/(28.0*u))
		const pauseStart = 2.2
		const pauseEnd = 2.0
		const fadeOutDur = 0.24
		const fadeInDur = 0.24
		cycle := pauseStart + scrollDur + pauseEnd + fadeOutDur + fadeInDur
		tCycle := math.Mod(math.Max(0, marqueeElapsed), cycle)

		offPx := 0.0
		fadeMult := 1.0
		if tCycle < pauseStart {
			offPx = 0.0
		} else if tCycle < pauseStart+scrollDur {
			p := (tCycle - pauseStart) / scrollDur
			offPx = overflowPx * p
		} else if tCycle < pauseStart+scrollDur+pauseEnd {
			offPx = overflowPx
		} else if tCycle < pauseStart+scrollDur+pauseEnd+fadeOutDur {
			offPx = overflowPx
			p := (tCycle - pauseStart - scrollDur - pauseEnd) / fadeOutDur
			fadeMult = 1.0 - nusaCssEase(p)
		} else {
			offPx = 0.0
			p := (tCycle - pauseStart - scrollDur - pauseEnd - fadeOutDur) / fadeInDur
			fadeMult = nusaCssEase(p)
		}

		useBrush := brush
		var tempBrush uintptr
		if fadeMult < 0.99 {
			_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(col, colAlpha*fadeMult)), uintptr(unsafe.Pointer(&tempBrush)))
			if tempBrush != 0 {
				useBrush = tempBrush
				defer diProcGdipDeleteBrush.Call(tempBrush)
			}
		}

		_, _, _ = diProcGdipSetClipRect.Call(
			g,
			uintptr(math.Float32bits(float32(px))),
			uintptr(math.Float32bits(float32(py))),
			uintptr(math.Float32bits(float32(pw))),
			uintptr(math.Float32bits(float32(ph))),
			0,
		)
		drawStrPhys(txt, font, useBrush, sfNoTrim, px-offPx, py, float64(boundRc.Width)+12.0*u, ph)
		_, _, _ = diProcGdipResetClip.Call(g)
	}

	timeStr := getWindowsFormattedTime()

	switch mode {
	case diModeResting, diModeClosed:
		miniPx := maxInt(21, int(math.Round(20.8*u)))
		topCov := int(math.Round((h/2.0)*u - float64(miniPx)/2.0))
		cyExact := float64(topCov) + float64(miniPx)/2.0
		if showLyric && playing && (lyricsOpen || c.compactStyle == "SATU_BARIS") && title != "" {
			txtHeightPx := float64(miniPx) * 0.60
			const closedClipMargin = 38.0
			_, _, _ = diProcGdipSetClipRect.Call(
				g,
				uintptr(math.Float32bits(float32((nx+closedClipMargin)*u))),
				uintptr(math.Float32bits(0)),
				uintptr(math.Float32bits(float32(math.Max(1.0, (w-closedClipMargin*2.0)*u)))),
				uintptr(math.Float32bits(float32(h*u))),
				0,
			)
			maxClosedW := math.Max(12.0*u, (w-84.0)*u)
			if closedLyricProg < 0.99 && closedLyricFrom != "" && closedLyricFrom != closedLyricTo {
				outOp := contentOp * (1.0 - closedLyricProg)
				outY := cyExact - 3.0*u*closedLyricProg
				if outOp > 0.02 {
					c.drawCenteredVectorTextGdip(
						g, famSemi, closedLyricFrom,
						(nx+w/2.0)*u, outY, txtHeightPx, maxClosedW,
						rgbaToARGB(pal.TextMain, outOp),
					)
				}
			}
			inTxt := closedLyricTo
			if inTxt == "" {
				inTxt = fallbackClosedLyricText(lyrics, lyricIdx, lyricsLoading, lyricsHasPlain, diStr)
			}
			if inTxt != "" {
				inOp := contentOp
				inY := cyExact
				if closedLyricProg < 0.99 && closedLyricFrom != "" && closedLyricFrom != closedLyricTo {
					inOp = contentOp * closedLyricProg
					inY = cyExact + 3.0*u*(1.0-closedLyricProg)
				}
				if lyricSweep && len(lyrics) > 0 && lyricIdx >= 0 && lyricIdx < len(lyrics) && inTxt == strings.TrimSpace(lyrics[lyricIdx].Text) {
					hasNext := lyricIdx+1 < len(lyrics)
					nextStart := 0.0
					if hasNext {
						nextStart = lyrics[lyricIdx+1].TimeSec
					}
					swProg := computeLineSweepProgressFromLine(lyrics[lyricIdx], nextStart, hasNext, effLyricPos)
					c.drawCenteredVectorTextSweepGdip(
						g, famSemi, inTxt,
						(nx+w/2.0)*u, inY, txtHeightPx, maxClosedW,
						rgbaToARGB(pal.TextMain, inOp),
						rgbaToARGB(pal.TextMain, 0.48*inOp),
						swProg,
						true,
						(nx+closedClipMargin)*u, 0, math.Max(1.0, (w-closedClipMargin*2.0)*u), h*u,
					)
				} else {
					c.drawCenteredVectorTextGdip(
						g, famSemi, inTxt,
						(nx+w/2.0)*u, inY, txtHeightPx, maxClosedW,
						rgbaToARGB(pal.TextMain, inOp),
					)
				}
			}
			_, _, _ = diProcGdipResetClip.Call(g)
		} else {
			c.drawCenteredVectorTextGdip(
				g,
				famSemi,
				timeStr,
				(nx+w/2.0)*u,
				cyExact,
				float64(miniPx)*0.62,
				(w-80.0)*u,
				rgbaToARGB(pal.TextMain, contentOp),
			)
		}

	case diModeHeadphones:
		drawStrPhys(timeStr, f13Semi, brushMain, sfLeft, mapXStatic(22), mapY(8), mapW(140), 22*u)

		outputs := audioOutputs
		if len(outputs) == 0 {
			bat := -1
			if isBt && btBat >= 0 && btBat <= 100 {
				bat = btBat
			}
			outputs = []windowsAudioEndpoint{{
				Name:     audioDev,
				ConnType: connType,
				IsActive: true,
				BatPct:   bat,
			}}
		}

		activeDev := outputs[0]
		for _, d := range outputs {
			if d.IsActive {
				activeDev = d
				break
			}
		}
		drawStrPhys(activeDev.Name, f14Semi, brushMain, sfLeft, mapX(68), mapY(38), mapW(290), 22*u)
		subTxt := diStr.Connected
		if activeDev.BatPct >= 0 && activeDev.BatPct <= 100 {
			subTxt = fmt.Sprintf("%s · %d%%", diStr.Connected, activeDev.BatPct)
		}
		drawStrPhys(subTxt, f115Reg, brushVar, sfLeft, mapX(68), mapY(60), mapW(290), 18*u)

		others := diOtherAudioIndices(outputs)
		if hpListProg > 0.01 && len(others) > 0 {
			_, _, _ = diProcGdipSetClipRect.Call(
				g,
				uintptr(math.Float32bits(float32(mapX(14)))),
				uintptr(math.Float32bits(float32(mapY(96)))),
				uintptr(math.Float32bits(float32(mapW(392)))),
				uintptr(math.Float32bits(float32(math.Max(0, mapH(h-114))))),
				0,
			)

			for idx, origIdx := range others {
				dev := outputs[origIdx]
				rowDelay := float64(idx) * 0.12
				rowProg := clamp01((hpListProg - rowDelay) / (1.0 - rowDelay))
				rowEase := nusaCssEase(rowProg)
				if rowEase <= 0.01 {
					continue
				}
				slideY := (1.0 - rowEase) * 10.0
				rowOp := contentOp * rowEase
				cy := 68.0 + float64(idx+1)*56.0 + slideY

				var bName uintptr
				_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(pal.TextVariant, 0.75*rowOp)), uintptr(unsafe.Pointer(&bName)))
				if bName != 0 {
					drawStrPhys(dev.Name, f14Semi, bName, sfLeft, mapX(68), mapY(cy-20), mapW(290), 22*u)
					_, _, _ = diProcGdipDeleteBrush.Call(bName)
				}

				subTxt := diStr.Available
				if dev.BatPct >= 0 && dev.BatPct <= 100 {
					subTxt = fmt.Sprintf("%s · %d%%", diStr.Available, dev.BatPct)
				}
				var bSub uintptr
				_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(pal.TextVariant, 0.52*rowOp)), uintptr(unsafe.Pointer(&bSub)))
				if bSub != 0 {
					drawStrPhys(subTxt, f115Reg, bSub, sfLeft, mapX(68), mapY(cy+2), mapW(290), 18*u)
					_, _, _ = diProcGdipDeleteBrush.Call(bSub)
				}
			}
			_, _, _ = diProcGdipResetClip.Call(g)
		}

	case diModeMusic:
		if showStatus {
			drawStrPhys(timeStr, f13Semi, brushMain, sfLeft, mapXStatic(22), mapY(8), mapW(140), 22*u)
		}

		mTextX := 78.0
		mTextW := 238.0
		if !showCover {
			mTextX = 22.0
			mTextW = 294.0
		}
		drawMarqueePhys(title, f14Semi, brushMain, pal.TextMain, contentOp, mapX(mTextX), mapY(34), mapW(mTextW), 22*u)
		if showArtist {
			drawMarqueePhys(artist, f115Reg, brushVar, pal.TextVariant, 0.85*contentOp, mapX(mTextX), mapY(56), mapW(mTextW), 18*u)
		}

	case diModeLyrics:
		if showStatus {
			drawStrPhys(timeStr, f13Semi, brushMain, sfLeft, mapXStatic(22), mapY(8), mapW(140), 22*u)
		}

		lTextX := 76.0
		lTextW := 164.0
		if !showCover {
			lTextX = 22.0
			lTextW = 218.0
		}
		drawMarqueePhys(title, f13Semi, brushMain, pal.TextMain, contentOp, mapX(lTextX), mapY(34), mapW(lTextW), 20*u)
		if showArtist {
			drawMarqueePhys(artist, f11Var, brushVar, pal.TextVariant, 0.85*contentOp, mapX(lTextX), mapY(56), mapW(lTextW), 18*u)
		}

		onPrimCont := color.RGBA{R: 0xD8, G: 0xF3, B: 0xFF, A: 255}
		var sfWrap uintptr
		_, _, _ = diProcGdipCreateStringFormat.Call(0, 0, uintptr(unsafe.Pointer(&sfWrap)))
		if sfWrap != 0 {
			_, _, _ = diProcGdipSetStringFormatTrim.Call(sfWrap, 1) // StringTrimmingWord (wrap, no "...")
			defer diProcGdipDeleteStringFormat.Call(sfWrap)
		} else {
			sfWrap = sfLeft
		}

		var fLyrUniform uintptr
		_, _, _ = diProcGdipCreateFont.Call(famReg, uintptr(math.Float32bits(float32(13.2*u))), 1, 2, uintptr(unsafe.Pointer(&fLyrUniform)))
		if fLyrUniform != 0 {
			defer diProcGdipDeleteFont.Call(fLyrUniform)
		} else {
			fLyrUniform = fLyrOn
		}

		wrapW := mapW(234)
		if len(lyrics) == 0 {
			msgTitle := diStr.NoLyrics
			if lyricsLoading {
				msgTitle = diStr.LoadingLyrics
			} else if lyricsHasPlain {
				msgTitle = diStr.NoSyncLyrics
			}
			msgHDp := 22.0
			if fLyrUniform != 0 && msgTitle != "" {
				u16, _ := windows.UTF16FromString(msgTitle)
				layoutRc := diRectF{X: 0, Y: 0, Width: float32(wrapW), Height: float32(106.5 * u)}
				var boundRc diRectF
				_, _, _ = diProcGdipMeasureString.Call(
					g,
					uintptr(unsafe.Pointer(&u16[0])),
					uintptr(len(u16)-1),
					fLyrUniform,
					uintptr(unsafe.Pointer(&layoutRc)),
					sfWrap,
					uintptr(unsafe.Pointer(&boundRc)),
					0, 0,
				)
				if boundRc.Height > 1.0 {
					msgHDp = float64(boundRc.Height)/u + 2.0
				}
			}
			yMsg := math.Max(42.0, 92.0-msgHDp/2.0)
			var bEmpty uintptr
			_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(onPrimCont, contentOp)), uintptr(unsafe.Pointer(&bEmpty)))
			if bEmpty != 0 {
				drawStrPhys(msgTitle, fLyrUniform, bEmpty, sfWrap, mapX(260), mapY(yMsg), wrapW, msgHDp*u+4*u)
				_, _, _ = diProcGdipDeleteBrush.Call(bEmpty)
			}
			return
		}
		lineTops := make([]float64, len(lyrics))
		lineHeights := make([]float64, len(lyrics))
		curTopDp := 0.0
		for i, line := range lyrics {
			hDp := 22.0
			if fLyrUniform != 0 && line.Text != "" {
				u16, _ := windows.UTF16FromString(line.Text)
				layoutRc := diRectF{X: 0, Y: 0, Width: float32(wrapW), Height: float32(400.0 * u)}
				var boundRc diRectF
				_, _, _ = diProcGdipMeasureString.Call(
					g,
					uintptr(unsafe.Pointer(&u16[0])),
					uintptr(len(u16)-1),
					fLyrUniform,
					uintptr(unsafe.Pointer(&layoutRc)),
					sfWrap,
					uintptr(unsafe.Pointer(&boundRc)),
					0, 0,
				)
				if boundRc.Height > 1.0 {
					hDp = float64(boundRc.Height)/u + 2.0
				}
			}
			lineTops[i] = curTopDp
			lineHeights[i] = hDp
			curTopDp += hDp + 6.0
		}

		activeTargetDp := 0.0
		if lyricIdx >= 0 && lyricIdx < len(lyrics) {
			activeTargetDp = lineTops[lyricIdx] + lineHeights[lyricIdx]/2.0
		} else if len(lyrics) > 0 {
			activeTargetDp = lineHeights[0] / 2.0
		}
		scrollCenterDp := activeTargetDp
		if prevLyricIdx >= 0 && prevLyricIdx < len(lyrics) && prevLyricIdx != lyricIdx && lyricAnimProg < 0.99 {
			prevCenter := lineTops[prevLyricIdx] + lineHeights[prevLyricIdx]/2.0
			scrollCenterDp = prevCenter + (activeTargetDp-prevCenter)*lyricAnimProg
		}
		maxScrollOff := math.Max(0.0, curTopDp-activeTargetDp)
		minScrollOff := -activeTargetDp
		clampedManual := math.Max(minScrollOff, math.Min(maxScrollOff, lyricManualOffsetDp))
		scrollCenterDp += clampedManual

		_, _, _ = diProcGdipSetClipRect.Call(
			g,
			uintptr(math.Float32bits(float32(mapX(256)))),
			uintptr(math.Float32bits(float32(mapY(39)))),
			uintptr(math.Float32bits(float32(mapW(242)))),
			uintptr(math.Float32bits(float32(mapH(106.5)))),
			0,
		)

		for i, line := range lyrics {
			yTop := 92.0 + lineTops[i] - scrollCenterDp
			yBot := yTop + lineHeights[i]
			if yBot < 38 || yTop > 148 {
				continue
			}
			lineCenterY := (yTop + yBot) / 2.0
			relMask := (lineCenterY - 39.0) / 106.5
			maskAlpha := 1.0
			if relMask < 0 {
				maskAlpha = 0.25
			} else if relMask < 0.14 {
				maskAlpha = 0.25 + 0.75*(relMask/0.14)
			} else if relMask > 1.0 {
				maskAlpha = 0.25
			} else if relMask > 0.86 {
				maskAlpha = 0.25 + 0.75*((1.0-relMask)/0.14)
			}

			opFactor := 0.35
			if i == lyricIdx {
				if prevLyricIdx >= 0 && prevLyricIdx != lyricIdx && lyricAnimProg < 0.99 {
					opFactor = 0.35 + 0.65*lyricAnimProg
				} else {
					opFactor = 1.0
				}
			} else if i == prevLyricIdx && prevLyricIdx != lyricIdx && lyricAnimProg < 0.99 {
				opFactor = 1.0 - 0.65*lyricAnimProg
			}
			if i == hoverLyric && opFactor < 0.95 {
				opFactor = 0.95
			}

			var bLine uintptr
			_, _, _ = diProcGdipCreateSolidFill.Call(uintptr(rgbaToARGB(onPrimCont, opFactor*contentOp*maskAlpha)), uintptr(unsafe.Pointer(&bLine)))
			if bLine != 0 {
				drawStrPhys(line.Text, fLyrUniform, bLine, sfWrap, mapX(260), mapY(yTop), wrapW, lineHeights[i]*u+4*u)
				_, _, _ = diProcGdipDeleteBrush.Call(bLine)
			}
		}
		_, _, _ = diProcGdipResetClip.Call(g)

	case diModeQueue:
		drawStrPhys(timeStr, f13Semi, brushMain, sfLeft, mapXStatic(22), mapY(8), mapW(140), 22*u)

		hdrTitle := qTitle
		if hdrTitle == "" || strings.EqualFold(hdrTitle, "Queue") || strings.EqualFold(hdrTitle, "Antrean") {
			hdrTitle = diStr.QueueTitle
		}
		drawStrPhys(hdrTitle, f14Semi, brushMain, sfCenter, mapX(72), mapY(43), mapW(276), 22*u)

		if len(qItems) == 0 {
			drawStrPhys(diStr.EmptyQueue, f13Semi, brushVar, sfCenter, mapX(40), mapY(136), mapW(340), 22*u)
			return
		}

		const qViewTop = 78.0
		const qViewBot = 242.0
		_, _, _ = diProcGdipSetClipRect.Call(
			g,
			uintptr(math.Float32bits(float32(mapX(14)))),
			uintptr(math.Float32bits(float32(mapY(qViewTop)))),
			uintptr(math.Float32bits(float32(mapW(382)))),
			uintptr(math.Float32bits(float32(mapH(qViewBot-qViewTop)))),
			0,
		)

		firstIdx := maxInt(0, int(math.Floor(qScrollDp/54.0))-1)
		lastIdx := minInt(len(qItems)-1, int(math.Ceil((qScrollDp+164.0)/54.0)))
		for itemIdx := firstIdx; itemIdx <= lastIdx; itemIdx++ {
			item := qItems[itemIdx]
			cy := 104.0 + float64(itemIdx)*54.0 - qScrollDp
			isCur := itemIdx == qIndex

			titleBrush := brushMain
			if isCur {
				titleBrush = brushAcc
			}
			drawStrPhys(item.Title, f13Semi, titleBrush, sfLeft, mapX(74), mapY(cy-18), mapW(278), 22*u)
			drawStrPhys(item.Artist, f11Var, brushVar, sfLeft, mapX(74), mapY(cy+2), mapW(278), 18*u)
		}
		_, _, _ = diProcGdipResetClip.Call(g)
	}
}

func fallbackClosedLyricText(lyrics []sources.LyricLine, lyricIdx int, lyricsLoading, lyricsHasPlain bool, diStr diLocalizedStrings) string {
	if len(lyrics) > 0 {
		if lyricIdx >= 0 && lyricIdx < len(lyrics) {
			return strings.TrimSpace(lyrics[lyricIdx].Text)
		}
		return "♪  • • •"
	}
	if lyricsLoading {
		return diStr.LoadingLyrics
	}
	if lyricsHasPlain {
		return diStr.NoSyncLyrics
	}
	return diStr.NoLyrics
}

func formatMMSS(sec float64) string {
	if sec < 0 || math.IsNaN(sec) || math.IsInf(sec, 0) {
		sec = 0
	}
	total := int(math.Floor(sec))
	return fmt.Sprintf("%02d:%02d", total/60, total%60)
}

func drawVeloraNotchShapePhys(pix []byte, cw, ch int, nx, w, h, rL, eL, eBotL, rR, eR, eBotR float64, pillMode bool, pal veloraBrandPalette) {
	if w <= 0 || h <= 0 {
		return
	}
	if rL > w/2 {
		rL = w / 2
	}
	if rL > h {
		rL = h
	}
	if rR > w/2 {
		rR = w / 2
	}
	if rR > h {
		rR = h
	}
	if pillMode && rL > h/2 {
		rL = h / 2
	}
	xStart := maxInt(0, int(math.Floor(nx-eL-2)))
	xEnd := minInt(cw-1, int(math.Ceil(nx+w+eR+2)))
	yEnd := minInt(ch-1, int(math.Ceil(h+math.Max(eBotL, eBotR)+2)))
	if pillMode {
		xStart = maxInt(0, int(math.Floor(nx-2)))
		xEnd = minInt(cw-1, int(math.Ceil(nx+w+2)))
		yEnd = minInt(ch-1, int(math.Ceil(h+2)))
	}

	for py := 0; py <= yEnd; py++ {
		fy := float64(py) + 0.5
		for px := xStart; px <= xEnd; px++ {
			fx := float64(px) + 0.5
			var dist float64
			if pillMode {
				dist = pillSignedDist(fx, fy, nx, w, h, rL)
			} else {
				dist = nusaNotchSignedDistLR(fx, fy, nx, w, h, rL, eL, eBotL, rR, eR, eBotR)
			}
			if dist > 0.75 {
				continue
			}
			cov := clamp01(0.5 - dist/1.5)
			if cov <= 0 {
				continue
			}
			blendPremulPixel(pix, cw, px, py, pal.BgDeep, cov)
		}
	}
}

func pillSignedDist(fx, fy, nx, w, h, r float64) float64 {
	cx := nx + w/2.0
	cy := h / 2.0
	qx := math.Abs(fx-cx) - (w/2.0 - r)
	qy := math.Abs(fy-cy) - (h/2.0 - r)
	ox := math.Max(qx, 0)
	oy := math.Max(qy, 0)
	return math.Hypot(ox, oy) + math.Min(math.Max(qx, qy), 0) - r
}

func nusaNotchSignedDistLR(fx, fy, nx, w, h, rL, eL, eBotL, rR, eR, eBotR float64) float64 {
	if fy > h {
		if fx < nx || fx > nx+w {
			return 10.0
		}
		if eBotL > 0.1 && fx >= nx && fx <= nx+eBotL && fy <= h+eBotL {
			return eBotL - math.Hypot(fx-(nx+eBotL), fy-(h+eBotL))
		}
		if eBotR > 0.1 && fx <= nx+w && fx >= nx+w-eBotR && fy <= h+eBotR {
			return eBotR - math.Hypot(fx-(nx+w-eBotR), fy-(h+eBotR))
		}
		if rL > 0.1 && fx < nx+rL {
			return math.Hypot(fx-(nx+rL), fy-(h-rL)) - rL
		}
		if rR > 0.1 && fx > nx+w-rR {
			return math.Hypot(fx-(nx+w-rR), fy-(h-rR)) - rR
		}
		return fy - h
	}
	if fx < nx {
		if eL <= 0.1 || fy > eL {
			return 10.0
		}
		return eL - math.Hypot(fx-(nx-eL), fy-eL)
	}
	if fx > nx+w {
		if eR <= 0.1 || fy > eR {
			return 10.0
		}
		return eR - math.Hypot(fx-(nx+w+eR), fy-eR)
	}
	if rL > 0.1 && fx < nx+rL && fy > h-rL {
		return math.Hypot(fx-(nx+rL), fy-(h-rL)) - rL
	}
	if rR > 0.1 && fx > nx+w-rR && fy > h-rR {
		return math.Hypot(fx-(nx+w-rR), fy-(h-rR)) - rR
	}
	return fy - h
}

func clipOutsideNotchPhys(pix []byte, cw, ch int, nx, w, h, rL, eL, eBotL, rR, eR, eBotR float64, pillMode bool, borderCol color.RGBA) {
	if pillMode && rL > h/2 {
		rL = h / 2
	}
	yEnd := minInt(ch-1, int(math.Ceil(h+math.Max(eBotL, eBotR)+4)))
	if pillMode {
		yEnd = minInt(ch-1, int(math.Ceil(h+4)))
	}
	for py := 0; py <= yEnd; py++ {
		fy := float64(py) + 0.5
		for px := 0; px < cw; px++ {
			idx := (py*cw + px) * 4
			if pix[idx+3] == 0 {
				continue
			}
			fx := float64(px) + 0.5
			var dist float64
			if pillMode {
				dist = pillSignedDist(fx, fy, nx, w, h, rL)
			} else {
				dist = nusaNotchSignedDistLR(fx, fy, nx, w, h, rL, eL, eBotL, rR, eR, eBotR)
			}
			if dist > 0.75 {
				pix[idx+0] = 0
				pix[idx+1] = 0
				pix[idx+2] = 0
				pix[idx+3] = 0
			} else if dist > -0.75 {
				cov := clamp01(0.5 - dist/1.5)
				a := uint8(math.Round(255.0 * cov))
				pix[idx+0] = uint8((uint16(pix[idx+0]) * uint16(a)) / 255)
				pix[idx+1] = uint8((uint16(pix[idx+1]) * uint16(a)) / 255)
				pix[idx+2] = uint8((uint16(pix[idx+2]) * uint16(a)) / 255)
				pix[idx+3] = a
			} else {
				pix[idx+3] = 255
			}
		}
	}
}

func nusaVerticalMask(t float64) float64 {
	if t <= 0 {
		return 1.0
	}
	if t < 0.30 {
		return 1.0 - (t/0.30)*0.45
	}
	if t < 0.70 {
		return 0.55 - ((t-0.30)/0.40)*0.43
	}
	if t < 1.0 {
		return 0.12 * (1.0 - (t-0.70)/0.30)
	}
	return 0
}

func sampleVeloraGradient(u float64, c1, c2, c3 color.RGBA) color.RGBA {
	if u < 0.52 {
		return lerpColor(c1, c2, u/0.52)
	}
	return lerpColor(c2, c3, (u-0.52)/0.48)
}

func lerpColor(a, b color.RGBA, k float64) color.RGBA {
	k = clamp01(k)
	return color.RGBA{
		R: uint8(float64(a.R)*(1-k) + float64(b.R)*k),
		G: uint8(float64(a.G)*(1-k) + float64(b.G)*k),
		B: uint8(float64(a.B)*(1-k) + float64(b.B)*k),
		A: 255,
	}
}

func blendRGBAWithAlpha(dst, src color.RGBA, alpha float64) color.RGBA {
	alpha = clamp01(alpha)
	return color.RGBA{
		R: uint8(float64(dst.R)*(1-alpha) + float64(src.R)*alpha),
		G: uint8(float64(dst.G)*(1-alpha) + float64(src.G)*alpha),
		B: uint8(float64(dst.B)*(1-alpha) + float64(src.B)*alpha),
		A: 255,
	}
}

func drawVeloraGlowingEyePhys(pix []byte, cw, ch int, cx, cy, rx, ry, u float64, haloCol color.RGBA) {
	pad := int(math.Ceil(13.0 * u))
	x0 := maxInt(0, int(cx-rx)-pad)
	x1 := minInt(cw-1, int(cx+rx)+pad)
	y0 := maxInt(0, int(cy-ry)-pad)
	y1 := minInt(ch-1, int(cy+ry)+pad)
	for py := y0; py <= y1; py++ {
		dy := (float64(py) + 0.5 - cy) / ry
		for px := x0; px <= x1; px++ {
			dx := (float64(px) + 0.5 - cx) / rx
			d := math.Hypot(dx, dy)
			if d <= 1.08 {
				cov := clamp01((1.08 - d) / 0.22)
				blendPremulPixel(pix, cw, px, py, color.RGBA{R: 245, G: 250, B: 255, A: 255}, cov)
			} else if d <= 2.5 {
				glow := math.Pow(1.0-(d-1.08)/1.42, 2) * 0.58
				blendPremulPixel(pix, cw, px, py, haloCol, glow)
			}
		}
	}
}

func drawConicRingPhys(pix []byte, cw, ch int, cx, cy, rOuter, rInner, progress float64, fg, bg color.RGBA) {
	x0 := maxInt(0, int(cx-rOuter-2))
	x1 := minInt(cw-1, int(cx+rOuter+2))
	y0 := maxInt(0, int(cy-rOuter-2))
	y1 := minInt(ch-1, int(cy+rOuter+2))
	for py := y0; py <= y1; py++ {
		dy := float64(py) + 0.5 - cy
		for px := x0; px <= x1; px++ {
			dx := float64(px) + 0.5 - cx
			d := math.Hypot(dx, dy)
			if d < rInner-0.75 || d > rOuter+0.75 {
				continue
			}
			covOuter := clamp01(0.5 - (d-rOuter)/1.5)
			covInner := clamp01(0.5 + (d-rInner)/1.5)
			cov := covOuter * covInner
			if cov <= 0 {
				continue
			}
			ang := math.Atan2(dx, -dy)
			if ang < 0 {
				ang += 2 * math.Pi
			}
			col := bg
			if ang/(2*math.Pi) <= progress {
				col = fg
			}
			blendPremulPixel(pix, cw, px, py, col, cov)
		}
	}
}

func buildVeloraFallbackArt(w, h int, radius float64, pal veloraBrandPalette) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	cx := float64(w) / 2.0
	cy := float64(h) / 2.0
	hw := cx - radius
	hh := cy - radius

	for y := 0; y < h; y++ {
		dy := math.Abs(float64(y)+0.5-cy) - hh
		if dy < 0 {
			dy = 0
		}
		for x := 0; x < w; x++ {
			dx := math.Abs(float64(x)+0.5-cx) - hw
			if dx < 0 {
				dx = 0
			}
			dist := math.Hypot(dx, dy) - radius
			if dist > 0.75 {
				continue
			}
			cov := clamp01(0.5 - dist/1.5)
			col := color.RGBA{R: 0x2C, G: 0x30, B: 0x37, A: uint8(255.0 * cov)}
			dst.SetRGBA(x, y, col)
		}
	}
	return dst
}

func blendPremulPixel(pix []byte, cw, x, y int, col color.RGBA, cov float64) {
	sa := (float64(col.A) / 255.0) * cov
	if sa <= 0 {
		return
	}
	idx := (y*cw + x) * 4
	inv := 1.0 - sa
	b := float64(col.B)*sa + float64(pix[idx+0])*inv
	g := float64(col.G)*sa + float64(pix[idx+1])*inv
	r := float64(col.R)*sa + float64(pix[idx+2])*inv
	a := sa*255.0 + float64(pix[idx+3])*inv
	if b > 255 {
		b = 255
	}
	if g > 255 {
		g = 255
	}
	if r > 255 {
		r = 255
	}
	if a > 255 {
		a = 255
	}
	pix[idx+0] = uint8(b)
	pix[idx+1] = uint8(g)
	pix[idx+2] = uint8(r)
	pix[idx+3] = uint8(a)
}

func blitPremulRGBAWithOp(pix []byte, cw, ch int, src *image.RGBA, dx, dy int, op float64) {
	blitPremulRGBAWithOpClip(pix, cw, ch, src, dx, dy, 0, ch, op)
}

func blitRotatedPremulRGBAWithOp(pix []byte, cw, ch int, src *image.RGBA, dx, dy int, angleRad, op float64) {
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	if w <= 0 || h <= 0 {
		return
	}
	cx := float64(w) / 2.0
	cy := float64(h) / 2.0
	rad := math.Min(cx, cy) - 0.5
	cosA := math.Cos(-angleRad)
	sinA := math.Sin(-angleRad)

	for sy := 0; sy < h; sy++ {
		ty := dy + sy
		if ty < 0 || ty >= ch {
			continue
		}
		vy := float64(sy) + 0.5 - cy
		for sx := 0; sx < w; sx++ {
			tx := dx + sx
			if tx < 0 || tx >= cw {
				continue
			}
			vx := float64(sx) + 0.5 - cx
			dist := math.Hypot(vx, vy) - rad
			if dist > 0.75 {
				continue
			}
			maskCov := clamp01(0.5 - dist/1.5)
			if maskCov <= 0 {
				continue
			}
			rx := vx*cosA - vy*sinA + cx - 0.5
			ry := vx*sinA + vy*cosA + cy - 0.5
			x0 := int(math.Floor(rx))
			y0 := int(math.Floor(ry))
			wx := rx - float64(x0)
			wy := ry - float64(y0)
			x1 := x0 + 1
			y1 := y0 + 1
			x0 = maxInt(0, minInt(w-1, x0))
			x1 = maxInt(0, minInt(w-1, x1))
			y0 = maxInt(0, minInt(h-1, y0))
			y1 = maxInt(0, minInt(h-1, y1))

			c00 := src.RGBAAt(b.Min.X+x0, b.Min.Y+y0)
			c10 := src.RGBAAt(b.Min.X+x1, b.Min.Y+y0)
			c01 := src.RGBAAt(b.Min.X+x0, b.Min.Y+y1)
			c11 := src.RGBAAt(b.Min.X+x1, b.Min.Y+y1)

			rF := (float64(c00.R)*(1-wx)+float64(c10.R)*wx)*(1-wy) + (float64(c01.R)*(1-wx)+float64(c11.R)*wx)*wy
			gF := (float64(c00.G)*(1-wx)+float64(c10.G)*wx)*(1-wy) + (float64(c01.G)*(1-wx)+float64(c11.G)*wx)*wy
			bF := (float64(c00.B)*(1-wx)+float64(c10.B)*wx)*(1-wy) + (float64(c01.B)*(1-wx)+float64(c11.B)*wx)*wy
			aF := (float64(c00.A)*(1-wx)+float64(c10.A)*wx)*(1-wy) + (float64(c01.A)*(1-wx)+float64(c11.A)*wx)*wy
			if aF < 1 {
				continue
			}
			col := color.RGBA{
				R: uint8(minInt(255, int(math.Round(rF)))),
				G: uint8(minInt(255, int(math.Round(gF)))),
				B: uint8(minInt(255, int(math.Round(bF)))),
				A: uint8(minInt(255, int(math.Round(aF*maskCov)))),
			}
			blendPremulPixel(pix, cw, tx, ty, col, op)
		}
	}
}

func blitPremulRGBAWithOpClip(pix []byte, cw, ch int, src *image.RGBA, dx, dy, clipTop, clipBot int, op float64) {
	b := src.Bounds()
	topLim := maxInt(0, clipTop)
	botLim := minInt(ch, clipBot)
	for sy := b.Min.Y; sy < b.Max.Y; sy++ {
		ty := dy + sy - b.Min.Y
		if ty < topLim || ty >= botLim {
			continue
		}
		for sx := b.Min.X; sx < b.Max.X; sx++ {
			tx := dx + sx - b.Min.X
			if tx < 0 || tx >= cw {
				continue
			}
			c := src.RGBAAt(sx, sy)
			if c.A == 0 {
				continue
			}
			blendPremulPixel(pix, cw, tx, ty, c, op)
		}
	}
}

func blitPremulRGBAWithOpClipSubY(pix []byte, cw, ch int, src *image.RGBA, dx int, dyFloat float64, clipTop, clipBot int, op float64) {
	b := src.Bounds()
	srcH := b.Dy()
	srcW := b.Dx()
	if srcH <= 0 || srcW <= 0 {
		return
	}
	baseY := int(math.Floor(dyFloat))
	frac := dyFloat - float64(baseY)
	if frac < 0.001 {
		blitPremulRGBAWithOpClip(pix, cw, ch, src, dx, baseY, clipTop, clipBot, op)
		return
	}
	topLim := maxInt(0, clipTop)
	botLim := minInt(ch, clipBot)
	invFrac := 1.0 - frac
	for ry := 0; ry <= srcH; ry++ {
		ty := baseY + ry
		if ty < topLim || ty >= botLim {
			continue
		}
		for sx := 0; sx < srcW; sx++ {
			tx := dx + sx
			if tx < 0 || tx >= cw {
				continue
			}
			var rF, gF, bF, aF float64
			if ry < srcH {
				c0 := src.RGBAAt(b.Min.X+sx, b.Min.Y+ry)
				rF += float64(c0.R) * invFrac
				gF += float64(c0.G) * invFrac
				bF += float64(c0.B) * invFrac
				aF += float64(c0.A) * invFrac
			}
			if ry > 0 {
				c1 := src.RGBAAt(b.Min.X+sx, b.Min.Y+ry-1)
				rF += float64(c1.R) * frac
				gF += float64(c1.G) * frac
				bF += float64(c1.B) * frac
				aF += float64(c1.A) * frac
			}
			if aF < 0.5 {
				continue
			}
			cBlend := color.RGBA{
				R: uint8(clamp01(rF/255.0) * 255.0),
				G: uint8(clamp01(gF/255.0) * 255.0),
				B: uint8(clamp01(bF/255.0) * 255.0),
				A: uint8(clamp01(aF/255.0) * 255.0),
			}
			blendPremulPixel(pix, cw, tx, ty, cBlend, op)
		}
	}
}

func resizeCoverBilinear(src image.Image, w, h int, radius float64) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw <= 0 || sh <= 0 {
		return dst
	}
	srcRGBA := image.NewRGBA(sb)
	draw.Draw(srcRGBA, sb, src, sb.Min, draw.Src)

	side := minInt(sw, sh)
	offX := sb.Min.X + (sw-side)/2
	offY := sb.Min.Y + (sh-side)/2

	cx := float64(w) / 2.0
	cy := float64(h) / 2.0
	hw := cx - radius
	hh := cy - radius

	for y := 0; y < h; y++ {
		dy := math.Abs(float64(y)+0.5-cy) - hh
		if dy < 0 {
			dy = 0
		}
		for x := 0; x < w; x++ {
			dx := math.Abs(float64(x)+0.5-cx) - hw
			if dx < 0 {
				dx = 0
			}
			dist := math.Hypot(dx, dy) - radius
			if dist > 0.75 {
				continue
			}
			cov := clamp01(0.5 - dist/1.5)
			if cov <= 0 {
				continue
			}

			var rAcc, gAcc, bAcc float64
			for sySub := 0; sySub < 2; sySub++ {
				v := (float64(y) + (float64(sySub)+0.5)*0.5) / float64(h)
				fy := float64(offY) + v*float64(side-1)
				y0 := int(fy)
				y1 := minInt(offY+side-1, y0+1)
				wy := fy - float64(y0)
				for sxSub := 0; sxSub < 2; sxSub++ {
					u := (float64(x) + (float64(sxSub)+0.5)*0.5) / float64(w)
					fx := float64(offX) + u*float64(side-1)
					x0 := int(fx)
					x1 := minInt(offX+side-1, x0+1)
					wx := fx - float64(x0)

					c00 := srcRGBA.RGBAAt(x0, y0)
					c10 := srcRGBA.RGBAAt(x1, y0)
					c01 := srcRGBA.RGBAAt(x0, y1)
					c11 := srcRGBA.RGBAAt(x1, y1)

					rAcc += (float64(c00.R)*(1-wx)+float64(c10.R)*wx)*(1-wy) + (float64(c01.R)*(1-wx)+float64(c11.R)*wx)*wy
					gAcc += (float64(c00.G)*(1-wx)+float64(c10.G)*wx)*(1-wy) + (float64(c01.G)*(1-wx)+float64(c11.G)*wx)*wy
					bAcc += (float64(c00.B)*(1-wx)+float64(c10.B)*wx)*(1-wy) + (float64(c01.B)*(1-wx)+float64(c11.B)*wx)*wy
				}
			}
			col := color.RGBA{
				R: uint8(minInt(255, int(rAcc*0.25))),
				G: uint8(minInt(255, int(gAcc*0.25))),
				B: uint8(minInt(255, int(bAcc*0.25))),
				A: uint8(255.0 * cov),
			}
			dst.SetRGBA(x, y, col)
		}
	}
	return dst
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
