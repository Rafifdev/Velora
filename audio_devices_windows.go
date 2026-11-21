//go:build windows

package main

import (
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type windowsAudioEndpoint struct {
	ID       string
	Name     string
	ConnType string // "bluetooth", "wired", "speaker"
	IsActive bool
	BatPct   int // -1 if no battery (wired/speaker), 0..100 if Bluetooth battery reported
	VolPct   int // 0..100 endpoint master volume (-1 if unavailable)
}

var (
	diOle32              = windows.NewLazySystemDLL("ole32.dll")
	diProcCoInitializeEx = diOle32.NewProc("CoInitializeEx")
	diProcCoUninitialize = diOle32.NewProc("CoUninitialize")
	diProcCoCreateInst   = diOle32.NewProc("CoCreateInstance")
	diProcCoTaskMemFree  = diOle32.NewProc("CoTaskMemFree")
	diProcPropVarClear   = diOle32.NewProc("PropVariantClear")

	clsidMMDeviceEnumerator = windows.GUID{Data1: 0xBCDE0395, Data2: 0xE52F, Data3: 0x467C, Data4: [8]byte{0x8E, 0x3D, 0xC4, 0x57, 0x92, 0x91, 0x69, 0x2E}}
	iidIMMDeviceEnumerator  = windows.GUID{Data1: 0xA95664D2, Data2: 0x9614, Data3: 0x4F35, Data4: [8]byte{0xA7, 0x46, 0xDE, 0x8D, 0xB6, 0x36, 0x17, 0xE6}}

	clsidPolicyConfig = windows.GUID{Data1: 0x870AF99C, Data2: 0x171D, Data3: 0x4F9E, Data4: [8]byte{0xAF, 0x0D, 0xE6, 0x3D, 0xF4, 0x0C, 0x2B, 0xC9}}
	iidIPolicyConfig  = windows.GUID{Data1: 0xF8679F50, Data2: 0x850A, Data3: 0x41CF, Data4: [8]byte{0x9C, 0x72, 0x43, 0x0F, 0x29, 0x02, 0x90, 0xC8}}

	iidIAudioEndpointVolume = windows.GUID{Data1: 0x5CDF2C82, Data2: 0x841E, Data3: 0x4546, Data4: [8]byte{0x97, 0x22, 0x0C, 0xF7, 0x40, 0x78, 0x22, 0x9A}}

	pkeyDeviceFriendlyName = struct {
		fmtid windows.GUID
		pid   uint32
	}{
		fmtid: windows.GUID{Data1: 0xA45C254E, Data2: 0xDF1C, Data3: 0x4EFD, Data4: [8]byte{0x80, 0x20, 0x67, 0xD1, 0x46, 0xA8, 0x50, 0xE0}},
		pid:   14,
	}
)

type comObj struct {
	vtbl *[32]uintptr
}

func comRelease(p uintptr) {
	if p == 0 {
		return
	}
	obj := (*comObj)(unsafe.Pointer(p))
	_, _, _ = syscall.SyscallN(obj.vtbl[2], p)
}

func classifyAudioEndpoint(rawName string, btFriendly string, btBat int) (connType string, batPct int) {
	low := strings.ToLower(rawName)
	batPct = -1
	if strings.Contains(low, "bluetooth") || strings.Contains(low, "bth") || strings.Contains(low, "hands-free") ||
		strings.Contains(low, "buds") || strings.Contains(low, "airpods") || strings.Contains(low, "soundcore") ||
		strings.Contains(low, "tws") || (btFriendly != "" && strings.Contains(low, strings.ToLower(btFriendly))) {
		connType = "bluetooth"
		if btBat >= 0 && btBat <= 100 {
			batPct = btBat
		}
		return connType, batPct
	}
	if strings.Contains(low, "headphone") || strings.Contains(low, "headset") || strings.Contains(low, "earphone") ||
		strings.Contains(low, "usb") || strings.Contains(low, "dac") || strings.Contains(low, "jack") {
		return "wired", -1
	}
	return "speaker", -1
}

func listWindowsAudioOutputs(btBat int, btFriendly string) []windowsAudioEndpoint {
	hrInit, _, _ := diProcCoInitializeEx.Call(0, 0x0) // COINIT_MULTITHREADED
	if hrInit == 0 || hrInit == 1 {
		defer diProcCoUninitialize.Call()
	}

	var enumPtr uintptr
	hr, _, _ := diProcCoCreateInst.Call(
		uintptr(unsafe.Pointer(&clsidMMDeviceEnumerator)),
		0,
		1, // CLSCTX_INPROC_SERVER
		uintptr(unsafe.Pointer(&iidIMMDeviceEnumerator)),
		uintptr(unsafe.Pointer(&enumPtr)),
	)
	if hr != 0 || enumPtr == 0 {
		return nil
	}
	defer comRelease(enumPtr)
	enumObj := (*comObj)(unsafe.Pointer(enumPtr))

	var defID string
	var defPtr uintptr
	hrDef, _, _ := syscall.SyscallN(enumObj.vtbl[4], enumPtr, 0, 0, uintptr(unsafe.Pointer(&defPtr)))
	if hrDef == 0 && defPtr != 0 {
		defObj := (*comObj)(unsafe.Pointer(defPtr))
		var idPtr *uint16
		if r, _, _ := syscall.SyscallN(defObj.vtbl[5], defPtr, uintptr(unsafe.Pointer(&idPtr))); r == 0 && idPtr != nil {
			defID = windows.UTF16PtrToString(idPtr)
			_, _, _ = diProcCoTaskMemFree.Call(uintptr(unsafe.Pointer(idPtr)))
		}
		comRelease(defPtr)
	}

	var colPtr uintptr
	hrCol, _, _ := syscall.SyscallN(enumObj.vtbl[3], enumPtr, 0, 1, uintptr(unsafe.Pointer(&colPtr)))
	if hrCol != 0 || colPtr == 0 {
		return nil
	}
	defer comRelease(colPtr)
	colObj := (*comObj)(unsafe.Pointer(colPtr))

	var count uint32
	if r, _, _ := syscall.SyscallN(colObj.vtbl[3], colPtr, uintptr(unsafe.Pointer(&count))); r != 0 {
		return nil
	}

	var out []windowsAudioEndpoint
	for i := uint32(0); i < count && i < 12; i++ {
		var devPtr uintptr
		if r, _, _ := syscall.SyscallN(colObj.vtbl[4], colPtr, uintptr(i), uintptr(unsafe.Pointer(&devPtr))); r != 0 || devPtr == 0 {
			continue
		}
		devObj := (*comObj)(unsafe.Pointer(devPtr))

		var idStr string
		var idPtr *uint16
		if r, _, _ := syscall.SyscallN(devObj.vtbl[5], devPtr, uintptr(unsafe.Pointer(&idPtr))); r == 0 && idPtr != nil {
			idStr = windows.UTF16PtrToString(idPtr)
			_, _, _ = diProcCoTaskMemFree.Call(uintptr(unsafe.Pointer(idPtr)))
		}

		friendly := ""
		var psPtr uintptr
		if r, _, _ := syscall.SyscallN(devObj.vtbl[4], devPtr, 0, uintptr(unsafe.Pointer(&psPtr))); r == 0 && psPtr != 0 {
			psObj := (*comObj)(unsafe.Pointer(psPtr))
			var pv [24]byte // PROPVARIANT
			if r2, _, _ := syscall.SyscallN(psObj.vtbl[5], psPtr, uintptr(unsafe.Pointer(&pkeyDeviceFriendlyName)), uintptr(unsafe.Pointer(&pv[0]))); r2 == 0 {
				vt := *(*uint16)(unsafe.Pointer(&pv[0]))
				if vt == 31 { // VT_LPWSTR
					valPtr := *(*uintptr)(unsafe.Pointer(&pv[8]))
					if valPtr != 0 {
						friendly = windows.UTF16PtrToString((*uint16)(unsafe.Pointer(valPtr)))
					}
				}
				_, _, _ = diProcPropVarClear.Call(uintptr(unsafe.Pointer(&pv[0])))
			}
			comRelease(psPtr)
		}

		volPct := -1
		var epVolPtr uintptr
		if rVol, _, _ := syscall.SyscallN(devObj.vtbl[3], devPtr, uintptr(unsafe.Pointer(&iidIAudioEndpointVolume)), 1, 0, uintptr(unsafe.Pointer(&epVolPtr))); rVol == 0 && epVolPtr != 0 {
			epVolObj := (*comObj)(unsafe.Pointer(epVolPtr))
			var scalar float32
			if rGet, _, _ := syscall.SyscallN(epVolObj.vtbl[9], epVolPtr, uintptr(unsafe.Pointer(&scalar))); rGet == 0 {
				v := int(scalar*100.0 + 0.5)
				if v < 0 {
					v = 0
				} else if v > 100 {
					v = 100
				}
				volPct = v
			}
			var isMuted int32
			if rMute, _, _ := syscall.SyscallN(epVolObj.vtbl[15], epVolPtr, uintptr(unsafe.Pointer(&isMuted))); rMute == 0 && isMuted != 0 {
				volPct = 0
			}
			comRelease(epVolPtr)
		}
		comRelease(devPtr)

		if idStr == "" || friendly == "" {
			continue
		}
		connType, batPct := classifyAudioEndpoint(friendly, btFriendly, btBat)
		out = append(out, windowsAudioEndpoint{
			ID:       idStr,
			Name:     cleanAudioDeviceName(friendly),
			ConnType: connType,
			IsActive: idStr == defID,
			BatPct:   batPct,
			VolPct:   volPct,
		})
	}
	return out
}

func setDefaultWindowsAudioOutput(deviceID string) bool {
	if deviceID == "" {
		return false
	}
	hrInit, _, _ := diProcCoInitializeEx.Call(0, 0x0)
	if hrInit == 0 || hrInit == 1 {
		defer diProcCoUninitialize.Call()
	}

	var pcPtr uintptr
	hr, _, _ := diProcCoCreateInst.Call(
		uintptr(unsafe.Pointer(&clsidPolicyConfig)),
		0,
		1,
		uintptr(unsafe.Pointer(&iidIPolicyConfig)),
		uintptr(unsafe.Pointer(&pcPtr)),
	)
	if hr != 0 || pcPtr == 0 {
		return false
	}
	defer comRelease(pcPtr)
	pcObj := (*comObj)(unsafe.Pointer(pcPtr))

	id16, err := windows.UTF16PtrFromString(deviceID)
	if err != nil {
		return false
	}
	ok := false
	for role := uintptr(0); role <= 2; role++ { // eConsole, eMultimedia, eCommunications
		r, _, _ := syscall.SyscallN(pcObj.vtbl[13], pcPtr, uintptr(unsafe.Pointer(id16)), role)
		if r == 0 {
			ok = true
		}
	}
	return ok
}
