package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	goruntime "runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"velora/internal/audio"
	"velora/internal/db"
	"velora/internal/discord"
	"velora/internal/jam"
	"velora/internal/sources"
	"velora/internal/stats"
	"velora/internal/tray"
)

type BootstrapData struct {
	Tracks         []db.Track             `json:"tracks"`
	Albums         []db.Album             `json:"albums"`
	Artists        []db.Artist            `json:"artists"`
	Playlists      []db.Playlist          `json:"playlists"`
	Sources        []db.Source            `json:"sources"`
	History        []db.PlayHistoryItem   `json:"history"`
	Preferences    map[string]string      `json:"preferences"`
	FireStreak     stats.FireStreakStatus `json:"fireStreak"`
	ListeningStats stats.ListeningStats   `json:"listeningStats"`
	MediaBaseURL   string                 `json:"mediaBaseUrl"`
}

type App struct {
	ctx          context.Context
	dataDir      string
	uploadsDir   string
	artworkDir   string
	downloadsDir string
	store        *db.Store
	statsSvc     *stats.Service
	localSrc      *sources.LocalScanner
	ytClient      *sources.YTMusicClient
	spotifyClient *sources.SpotifyClient
	deezerClient  *sources.DeezerClient
	downloader    *sources.DownloadManager
	jamSrv       *jam.Server
	discordRPC   *discord.Client
	frontendFS   fs.FS
	trayIcon     []byte
	trayCtrl     *tray.Controller
	quitting     atomic.Bool
	closeToTray  atomic.Bool
	closingOnce  sync.Once
	player               *audio.PlayerController
	ytHomeMu             sync.Mutex
	ytHomeDone           chan struct{}
	ytHomeFeed           sources.YTMusicHomeFeed
	ytHomeErr            error
	ytHomeCached         bool
	ytHomeAt             time.Time
	lyricUpgradeMu       sync.Mutex
	lyricUpgradeTriedMap map[int64]bool
	lyricsSeqMu          sync.Mutex
	lyricsSeqMap         map[int64]uint64
}

func NewApp() *App {
	return &App{}
}

func (a *App) SetFrontendFS(fsys fs.FS) {
	a.frontendFS = fsys
	if a.jamSrv != nil && fsys != nil {
		a.jamSrv.SetFrontendFS(fsys)
	}
}

func (a *App) SetTrayIcon(icon []byte) {
	a.trayIcon = icon
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	cfgDir, err := os.UserConfigDir()
	if err != nil || cfgDir == "" {
		cfgDir = os.TempDir()
	}
	a.dataDir = filepath.Join(cfgDir, "Velora")
	a.uploadsDir = filepath.Join(a.dataDir, "uploads")
	a.artworkDir = filepath.Join(a.dataDir, "artwork_cache")
	a.downloadsDir = filepath.Join(a.dataDir, "downloads")
	_ = os.MkdirAll(a.dataDir, 0o755)
	_ = os.MkdirAll(a.uploadsDir, 0o755)
	_ = os.MkdirAll(a.artworkDir, 0o755)
	_ = os.MkdirAll(a.downloadsDir, 0o755)

	store, err := db.Open(filepath.Join(a.dataDir, "velora.db"))
	if err != nil {
		panic(fmt.Errorf("failed to open velora.db: %w", err))
	}
	_ = store.SeedDefaults()

	a.store = store
	a.statsSvc = stats.NewService(store)
	a.localSrc = sources.NewLocalScanner(store, a.uploadsDir, a.artworkDir)
	a.ytClient = sources.NewYTMusicClient()
	a.spotifyClient = sources.NewSpotifyClient()
	a.deezerClient = sources.NewDeezerClient()
	discordEnabled := true
	discordClientID := ""
	if prefs, pErr := a.store.GetPreferences(); pErr == nil {
		a.closeToTray.Store(prefs["system.closeToTray"] == "true")
		a.spotifyClient.SetCredentials(prefs["integrations.spotify.clientId"], prefs["integrations.spotify.clientSecret"])
		discordEnabled = prefs["integrations.discord.enabled"] != "false"
		discordClientID = prefs["integrations.discord.clientId"]
		lang := prefs["content.language"]
		if strings.TrimSpace(lang) == "" {
			lang = detectSystemLanguage()
		}
		a.applyContentLocale(prefs["content.country"], lang)
	}
	a.discordRPC = discord.NewClient(discordEnabled, discordClientID)
	a.downloader = sources.NewDownloadManager(store, a.ytClient, a.downloadsDir, func(eventName string, data any) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, eventName, data)
		}
	})
	a.downloader.SetTrackResolver(a.resolveTrackForDownload)
	a.startYTMusicHomePrefetch()
	go a.ensureStoreDesktopShortcut()
	a.player = audio.NewPlayerController(store, a.ytClient, a.artworkDir)
	_, _ = a.player.StartMediaServer()
	a.jamSrv = jam.NewServer(
		a.player.BaseURL(),
		func(cmd jam.Command) {
			if a.ctx != nil {
				runtime.EventsEmit(a.ctx, "jam:command", cmd)
			}
		},
		func(_ int) {
			if a.ctx != nil && a.jamSrv != nil {
				runtime.EventsEmit(a.ctx, "jam:status", a.jamSrv.Status())
			}
		},
	)
	a.jamSrv.SetSearchHandler(func(ctx context.Context, q string) ([]jam.SearchResult, error) {
		var out []jam.SearchResult
		seen := make(map[string]struct{})
		if local, err := a.store.GetTracks("all", q); err == nil {
			for _, tr := range local {
				if len(out) >= 15 {
					break
				}
				if tr.ExternalID != "" {
					seen[tr.ExternalID] = struct{}{}
				}
				out = append(out, jam.SearchResult{
					TrackID:     tr.ID,
					ExternalID:  tr.ExternalID,
					Title:       tr.Title,
					Artist:      tr.ArtistName,
					Album:       tr.AlbumTitle,
					DurationSec: tr.DurationSec,
					CoverURL:    tr.CoverURL,
					SourceType:  tr.SourceType,
				})
			}
		}
		if hits, err := a.SearchYouTubeMusic(q); err == nil {
			for _, yt := range hits {
				if len(out) >= 25 {
					break
				}
				if yt.VideoID != "" {
					if _, dup := seen[yt.VideoID]; dup {
						continue
					}
					seen[yt.VideoID] = struct{}{}
				}
				out = append(out, jam.SearchResult{
					ExternalID:  yt.VideoID,
					Title:       yt.Title,
					Artist:      yt.Artist,
					Album:       yt.Album,
					DurationSec: yt.DurationSec,
					CoverURL:    yt.CoverURL,
					SourceType:  "ytmusic",
				})
			}
		}
		return out, nil
	})
	a.jamSrv.SetLyricsHandler(func(_ context.Context, snap jam.PlaybackSnapshot) (jam.LyricsPayload, error) {
		lyr := a.GetTrackLyrics(snap.TrackID, snap.Title, snap.Artist, snap.Album, int(snap.DurationSec))
		synced := make([]jam.LyricLine, 0, len(lyr.SyncedLyrics))
		for _, line := range lyr.SyncedLyrics {
			synced = append(synced, jam.LyricLine{TimeSec: line.TimeSec, Text: line.Text})
		}
		return jam.LyricsPayload{
			PlainLyrics:  lyr.PlainLyrics,
			SyncedLyrics: synced,
		}, nil
	})
	if a.frontendFS != nil {
		a.jamSrv.SetFrontendFS(a.frontendFS)
	}
	a.jamSrv.SetRPCHandler(a.handleJamRPC)
	a.trayCtrl = tray.NewController(
		a.trayIcon,
		func() {
			if a.ctx != nil {
				runtime.WindowShow(a.ctx)
				runtime.WindowUnminimise(a.ctx)
			}
		},
		func() {
			a.StopJam()
		},
		func() {
			a.forceQuitApp()
		},
	)
	if prefs, pErr := a.store.GetPreferences(); pErr == nil {
		if d, err := strconv.Atoi(prefs["privacy.historyDays"]); err == nil && d > 0 {
			_ = a.store.PrunePlayHistory(d)
		}
	}
}

func (a *App) startYTMusicHomePrefetch() {
	a.ytHomeMu.Lock()
	if a.ytHomeDone != nil {
		a.ytHomeMu.Unlock()
		return
	}
	done := make(chan struct{})
	a.ytHomeDone = done
	a.ytHomeMu.Unlock()

	go func() {
		defer close(done)
		reqCtx, cancel := context.WithTimeout(context.Background(), 18*time.Second)
		defer cancel()
		feed, err := a.buildHomeFeed(reqCtx)
		a.ytHomeMu.Lock()
		a.ytHomeFeed = feed
		a.ytHomeErr = err
		a.ytHomeCached = err == nil && (len(feed.Shelves) > 0 || len(feed.Tracks) > 0)
		if a.ytHomeCached {
			a.ytHomeAt = time.Now()
		}
		a.ytHomeMu.Unlock()
	}()
}

func (a *App) buildHomeFeed(ctx context.Context) (sources.YTMusicHomeFeed, error) {
	// Primary: direct YouTube Music InnerTube (WEB_REMIX with country/language locale, matching Auralys).
	feed, err := a.ytClient.GetHomeFeed(ctx)
	if err == nil && (len(feed.Shelves) > 0 || len(feed.Tracks) > 0) {
		return feed, nil
	}
	// Graceful fallbacks if YouTube Music browse is temporarily unavailable.
	if a.useSpotifyMetadata() {
		if spFeed, spErr := a.spotifyClient.GetHomeFeed(ctx); spErr == nil && (len(spFeed.Shelves) > 0 || len(spFeed.Tracks) > 0) {
			return spFeed, nil
		}
	}
	if a.useDeezerMetadata() {
		if dzFeed, dzErr := a.deezerClient.GetHomeFeed(ctx); dzErr == nil && (len(dzFeed.Shelves) > 0 || len(dzFeed.Tracks) > 0) {
			return dzFeed, nil
		}
	}
	return feed, err
}

func (a *App) useSpotifyMetadata() bool {
	if a.spotifyClient == nil || !a.spotifyClient.IsConfigured() {
		return false
	}
	if a.store != nil {
		if prefs, err := a.store.GetPreferences(); err == nil {
			if v, ok := prefs["integrations.spotify.enabled"]; ok && strings.EqualFold(strings.TrimSpace(v), "false") {
				return false
			}
		}
	}
	return true
}

func (a *App) useDeezerMetadata() bool {
	if a.deezerClient == nil {
		return false
	}
	if a.store != nil {
		if prefs, err := a.store.GetPreferences(); err == nil {
			if v, ok := prefs["integrations.deezer.enabled"]; ok && strings.EqualFold(strings.TrimSpace(v), "false") {
				return false
			}
		}
	}
	return true
}

func isHybridTrackID(extID string) bool {
	id := strings.TrimSpace(extID)
	return strings.HasPrefix(id, "sp:track:") || strings.HasPrefix(id, "dz:track:")
}

func (a *App) invalidateHomeCache() {
	a.ytHomeMu.Lock()
	a.ytHomeCached = false
	a.ytHomeAt = time.Time{}
	a.ytHomeFeed = sources.YTMusicHomeFeed{}
	a.ytHomeErr = nil
	a.ytHomeDone = nil
	a.ytHomeMu.Unlock()
}

var atvUpgradeCache sync.Map

func (a *App) resolveTrackForDownload(ctx context.Context, tr db.Track) (db.Track, error) {
	if !isHybridTrackID(strings.TrimSpace(tr.ExternalID)) {
		return tr, nil
	}
	return a.ensureTrackPlayableExternalID(ctx, tr)
}

func (a *App) ensureTrackPlayableExternalID(ctx context.Context, tr db.Track) (db.Track, error) {
	extID := strings.TrimSpace(tr.ExternalID)
	if !isHybridTrackID(extID) {
		if tr.SourceType == "ytmusic" && !tr.IsDownloaded && extID != "" && a.ytClient != nil && sources.ShouldPreferStudioATV(tr.Title, tr.CoverURL, "") {
			var atv sources.YTMusicTrack
			if cached, ok := atvUpgradeCache.Load(extID); ok {
				if c, okType := cached.(sources.YTMusicTrack); okType {
					atv = c
				}
			} else {
				atvCtx, cancel := context.WithTimeout(ctx, 4500*time.Millisecond)
				found, okFound := a.ytClient.FindStudioATVTrack(atvCtx, tr.Title, tr.ArtistName)
				cancel()
				if okFound {
					atv = found
				}
				atvUpgradeCache.Store(extID, atv)
			}
			if atv.VideoID != "" && atv.VideoID != extID {
				if a.store != nil && tr.ID > 0 {
					if upgraded, uErr := a.store.UpgradeTrackToStudio(tr.ID, atv.VideoID, atv.CoverURL, atv.DurationSec); uErr == nil {
						return upgraded, nil
					}
				}
				tr.ExternalID = atv.VideoID
				if atv.DurationSec > 0 {
					tr.DurationSec = atv.DurationSec
				}
				if atv.CoverURL != "" {
					tr.CoverURL = atv.CoverURL
				}
			}
		}
		return tr, nil
	}
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var (
		vid string
		err error
	)
	if strings.HasPrefix(extID, "dz:track:") {
		if a.deezerClient == nil {
			return tr, fmt.Errorf("deezer client not initialized")
		}
		vid, err = a.deezerClient.ResolveTrackToYTMusic(reqCtx, a.ytClient, extID, tr.Title, tr.ArtistName, tr.DurationSec)
	} else if !strings.HasPrefix(extID, "sp:import:") && a.spotifyClient != nil {
		vid, err = a.spotifyClient.ResolveTrackToYTMusic(reqCtx, a.ytClient, extID, tr.Title, tr.ArtistName, tr.DurationSec)
	} else {
		err = fmt.Errorf("fallback yt search")
	}
	if err != nil || strings.TrimSpace(vid) == "" {
		q := strings.TrimSpace(tr.Title + " " + tr.ArtistName)
		if hits, sErr := a.ytClient.Search(reqCtx, q); sErr == nil && len(hits) > 0 {
			if picked := sources.PickBestYTMusicVideoID(hits, tr.Title, tr.ArtistName, tr.DurationSec); picked != "" {
				vid = picked
				err = nil
			}
		}
	}
	if err != nil {
		return tr, err
	}
	tr.ExternalID = vid
	if a.store != nil && tr.ID > 0 {
		_ = a.store.UpdateTrackExternalID(tr.ID, vid)
	}
	return tr, nil
}

func (a *App) domReady(ctx context.Context) {
	a.initTaskbar(ctx)
	a.initDynamicIsland(ctx)
	startMemoryTrimMonitor(ctx)
}

func (a *App) TrimMemory() {
	go trimAppMemoryWorkingSet()
}

func (a *App) forceQuitApp() {
	a.quitting.Store(true)
	a.closingOnce.Do(func() {
		if a.ctx != nil {
			runtime.WindowHide(a.ctx)
		}
		time.AfterFunc(2*time.Second, func() {
			os.Exit(0)
		})
		if a.ctx != nil {
			go runtime.Quit(a.ctx)
		} else {
			os.Exit(0)
		}
	})
}

func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	if a.quitting.Load() {
		return false
	}
	if a.closeToTray.Load() {
		go func() {
			if a.trayCtrl != nil {
				remoteURL := ""
				running := false
				if a.jamSrv != nil {
					st := a.jamSrv.Status()
					running = st.Running
					remoteURL = st.RemoteURL
				}
				a.trayCtrl.EnsureStarted(running, remoteURL)
			}
		}()
		runtime.WindowHide(ctx)
		return true
	}
	a.forceQuitApp()
	return false
}

func (a *App) shutdown(ctx context.Context) {
	setSystemKeepAwake(false)
	var wg sync.WaitGroup
	wg.Add(5)
	go func() {
		defer wg.Done()
		if a.discordRPC != nil {
			a.discordRPC.Shutdown()
		}
	}()
	go func() {
		defer wg.Done()
		a.shutdownTaskbar()
	}()
	go func() {
		defer wg.Done()
		a.shutdownDynamicIsland()
	}()
	go func() {
		defer wg.Done()
		if a.trayCtrl != nil {
			a.trayCtrl.Shutdown()
		}
	}()
	go func() {
		defer wg.Done()
		if a.jamSrv != nil {
			a.jamSrv.Stop()
		}
		if a.player != nil {
			a.player.Shutdown()
		}
	}()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(1200 * time.Millisecond):
	}
	if a.store != nil {
		_ = a.store.Close()
	}
}

func (a *App) GetBootstrapData() (BootstrapData, error) {
	tracks, _ := a.store.GetTracks("all", "")
	albums, _ := a.store.GetAlbums(false)
	artists, _ := a.store.GetArtists(false)
	playlists, _ := a.store.GetPlaylists(false)
	srcs, _ := a.store.GetSources()
	history, _ := a.store.GetPlayHistory(100)
	prefs, _ := a.store.GetPreferences()
	if prefs == nil {
		prefs = map[string]string{}
	}
	if prefs["content.language"] == "" {
		prefs["content.language"] = detectSystemLanguage()
	}
	a.SetDynamicIslandLanguage(prefs["content.language"])
	streak, _ := a.statsSvc.GetFireStreak(time.Now())
	lStats, _ := a.statsSvc.GetListeningStats()

	return BootstrapData{
		Tracks:         tracks,
		Albums:         albums,
		Artists:        artists,
		Playlists:      playlists,
		Sources:        srcs,
		History:        history,
		Preferences:    prefs,
		FireStreak:     streak,
		ListeningStats: lStats,
		MediaBaseURL:   a.player.BaseURL(),
	}, nil
}

func (a *App) GetTracks(filter, query string) ([]db.Track, error) {
	return a.store.GetTracks(filter, query)
}

func (a *App) GetAlbums(favoritesOnly bool) ([]db.Album, error) {
	return a.store.GetAlbums(favoritesOnly)
}

func (a *App) GetAlbumTracks(albumID int64) ([]db.Track, error) {
	return a.store.GetAlbumTracks(albumID)
}

func (a *App) GetArtists(favoritesOnly bool) ([]db.Artist, error) {
	return a.store.GetArtists(favoritesOnly)
}

func (a *App) GetArtistTracks(artistID int64) ([]db.Track, error) {
	return a.store.GetArtistTracks(artistID)
}

func (a *App) ToggleFavoriteTrack(id int64) (bool, error) {
	return a.store.ToggleFavoriteTrack(id)
}

func (a *App) ToggleFavoriteAlbum(id int64) (bool, error) {
	return a.store.ToggleFavoriteAlbum(id)
}

func (a *App) ToggleFavoriteYTMusicAlbum(browseID, title, artistName, coverURL string, releaseYear int) (db.Album, error) {
	return a.store.ToggleFavoriteYTMusicAlbum(browseID, title, artistName, coverURL, releaseYear)
}

func (a *App) ToggleSaveYTMusicPlaylist(browseID, title, subtitle, coverURL string, trackIDs []int64) (bool, error) {
	return a.store.ToggleSaveYTMusicPlaylist(browseID, title, subtitle, coverURL, "#1DB954", trackIDs)
}

func (a *App) ToggleFavoriteArtist(id int64) (bool, error) {
	return a.store.ToggleFavoriteArtist(id)
}

func (a *App) ToggleFavoriteYTMusicArtist(externalID, name, avatarURL string) (db.Artist, error) {
	return a.store.ToggleFavoriteYTMusicArtist(externalID, name, avatarURL)
}

func (a *App) GetPlaylists(localOnly bool) ([]db.Playlist, error) {
	return a.store.GetPlaylists(localOnly)
}

func (a *App) GetPlaylistDetail(id int64) (db.PlaylistDetail, error) {
	return a.store.GetPlaylistDetail(id)
}

func (a *App) GetAudioOutputs() []AudioOutputDevice {
	list := listWindowsAudioOutputs(-1, "")
	if list == nil {
		return []AudioOutputDevice{}
	}
	return list
}

func (a *App) SetAudioOutput(deviceID string) bool {
	ok := setDefaultWindowsAudioOutput(strings.TrimSpace(deviceID))
	if ok {
		a.NotifyDynamicIslandAudioDevice("")
	}
	return ok
}

func (a *App) CreatePlaylist(name, description string, isLocal bool, coverColor string) (db.Playlist, error) {
	return a.store.CreatePlaylist(name, description, isLocal, coverColor)
}

func (a *App) UpdatePlaylist(id int64, name, description, coverColor string) (db.Playlist, error) {
	return a.store.UpdatePlaylist(id, name, description, coverColor)
}

func (a *App) DeletePlaylist(id int64) error {
	return a.store.DeletePlaylist(id)
}

func (a *App) AddTrackToPlaylist(playlistID, trackID int64) error {
	return a.store.AddTrackToPlaylist(playlistID, trackID)
}

func (a *App) RemoveTrackFromPlaylist(playlistID, trackID int64) error {
	return a.store.RemoveTrackFromPlaylist(playlistID, trackID)
}

func (a *App) ReorderPlaylistTracks(playlistID int64, trackIDs []int64) error {
	return a.store.ReorderPlaylistTracks(playlistID, trackIDs)
}

func (a *App) ExportLocalPlaylistM3U(playlistID int64) (string, error) {
	detail, err := a.store.GetPlaylistDetail(playlistID)
	if err != nil {
		return "", err
	}
	savePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export Local Playlist (.m3u)",
		DefaultFilename: detail.Playlist.Name + ".m3u",
		Filters: []runtime.FileFilter{
			{DisplayName: "M3U Playlist (*.m3u)", Pattern: "*.m3u;*.m3u8"},
		},
	})
	if err != nil || savePath == "" {
		return "", err
	}
	if err := sources.ExportPlaylistM3U(detail, savePath); err != nil {
		return "", err
	}
	return savePath, nil
}

func (a *App) ImportLocalPlaylistM3U() (db.Playlist, error) {
	openPath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Import Local Playlist (.m3u / .m3u8)",
		Filters: []runtime.FileFilter{
			{DisplayName: "M3U Playlists", Pattern: "*.m3u;*.m3u8"},
		},
	})
	if err != nil || openPath == "" {
		return db.Playlist{}, err
	}
	pl, _, err := a.localSrc.ImportPlaylistM3U(openPath)
	return pl, err
}

func (a *App) ImportSpotifyPlaylist(spotifyURL string) (db.Playlist, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	sp, err := sources.FetchSpotifyPlaylist(reqCtx, spotifyURL)
	if err != nil {
		return db.Playlist{}, err
	}

	pl, err := a.store.CreatePlaylist(sp.Name, "Imported from Spotify", true, "#1DB954")
	if err != nil {
		return db.Playlist{}, err
	}
	if sp.CoverURL != "" {
		a.store.UpdatePlaylistCover(pl.ID, sp.CoverURL)
	}

	ytTracks := sources.ResolveSpotifyTracksToYTMusic(reqCtx, a.ytClient, sp.Tracks)
	for _, yt := range ytTracks {
		if saved, err := a.store.UpsertTrack(db.Track{
			Title:       yt.Title,
			ArtistName:  yt.Artist,
			AlbumTitle:  yt.Album,
			DurationSec: yt.DurationSec,
			Format:      yt.Format,
			Bitrate:     160,
			SourceType:  "ytmusic",
			ExternalID:  yt.VideoID,
			CoverURL:    yt.CoverURL,
		}); err == nil {
			_ = a.store.AddTrackToPlaylist(pl.ID, saved.ID)
		}
	}

	detail, err := a.store.GetPlaylistDetail(pl.ID)
	if err != nil {
		return pl, nil
	}
	return detail.Playlist, nil
}

func (a *App) UploadLocalFiles() ([]db.Track, error) {
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Upload Audio Files ke Velora",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "All Supported Audio Files",
				Pattern:     "*.mp3;*.mp2;*.mp1;*.mpa;*.flac;*.m4a;*.m4b;*.m4p;*.mp4;*.alac;*.aac;*.ogg;*.oga;*.opus;*.spx;*.wav;*.wave;*.aiff;*.aif;*.aifc;*.caf;*.au;*.snd;*.wma;*.asf;*.webm;*.mka;*.ape;*.wv;*.mpc;*.tak;*.tta;*.dsf;*.dff;*.ac3;*.dts;*.amr;*.awb;*.3gp;*.mid;*.midi",
			},
			{
				DisplayName: "All Files (*.*)",
				Pattern:     "*.*",
			},
		},
	})
	if err != nil || len(files) == 0 {
		return []db.Track{}, err
	}
	return a.localSrc.ImportFiles(files, true)
}

func (a *App) ImportLocalPaths(paths []string) ([]db.Track, error) {
	return a.localSrc.ImportFiles(paths, true)
}

func (a *App) AddLocalFolder() ([]db.Track, error) {
	folder, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Pilih Folder Musik Lokal",
	})
	if err != nil || folder == "" {
		return []db.Track{}, err
	}
	return a.localSrc.ScanFolder(folder)
}

func (a *App) RescanSource(sourceID int64) ([]db.Track, error) {
	src, err := a.store.GetSourceByID(sourceID)
	if err != nil {
		return nil, err
	}
	if src.Path == "" {
		return []db.Track{}, nil
	}
	return a.localSrc.ScanFolder(src.Path)
}

func (a *App) RemoveSource(sourceID int64) error {
	return a.store.RemoveSource(sourceID)
}

func (a *App) GetSources() ([]db.Source, error) {
	return a.store.GetSources()
}

func (a *App) SearchYouTubeMusic(query string) ([]sources.YTMusicTrack, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	if tracks, err := a.ytClient.Search(reqCtx, query); err == nil && len(tracks) > 0 {
		return tracks, nil
	}
	if a.useSpotifyMetadata() {
		if tracks, err := a.spotifyClient.SearchTracks(reqCtx, query); err == nil && len(tracks) > 0 {
			return tracks, nil
		}
	}
	if a.useDeezerMetadata() {
		if tracks, err := a.deezerClient.SearchTracks(reqCtx, query); err == nil && len(tracks) > 0 {
			return tracks, nil
		}
	}
	return []sources.YTMusicTrack{}, nil
}

func (a *App) SearchYouTubeMusicPlaylists(query string) ([]sources.YTMusicCard, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 14*time.Second)
	defer cancel()
	if cards, err := a.ytClient.SearchPlaylists(reqCtx, query); err == nil && len(cards) > 0 {
		return cards, nil
	}
	if a.useSpotifyMetadata() {
		if cards, err := a.spotifyClient.SearchPlaylists(reqCtx, query); err == nil && len(cards) > 0 {
			return cards, nil
		}
	}
	if a.useDeezerMetadata() {
		if cards, err := a.deezerClient.SearchPlaylists(reqCtx, query); err == nil && len(cards) > 0 {
			return cards, nil
		}
	}
	return []sources.YTMusicCard{}, nil
}

func (a *App) SearchYouTubeMusicAlbums(query string) ([]sources.YTMusicCard, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if cards, err := a.ytClient.SearchAlbums(reqCtx, query); err == nil && len(cards) > 0 {
		return cards, nil
	}
	if a.useSpotifyMetadata() {
		if cards, err := a.spotifyClient.SearchAlbums(reqCtx, query); err == nil && len(cards) > 0 {
			return cards, nil
		}
	}
	if a.useDeezerMetadata() {
		if cards, err := a.deezerClient.SearchAlbums(reqCtx, query); err == nil && len(cards) > 0 {
			return cards, nil
		}
	}
	return []sources.YTMusicCard{}, nil
}

func (a *App) SearchYouTubeMusicArtists(query string) ([]sources.YTMusicCard, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if cards, err := a.ytClient.SearchArtists(reqCtx, query); err == nil && len(cards) > 0 {
		return cards, nil
	}
	if a.useSpotifyMetadata() {
		if cards, err := a.spotifyClient.SearchArtists(reqCtx, query); err == nil && len(cards) > 0 {
			return cards, nil
		}
	}
	if a.useDeezerMetadata() {
		if cards, err := a.deezerClient.SearchArtists(reqCtx, query); err == nil && len(cards) > 0 {
			return cards, nil
		}
	}
	return []sources.YTMusicCard{}, nil
}

func (a *App) GetMoreArtists(offset int, excludeBrowseIDs []string) ([]sources.YTMusicCard, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	queries := []string{
		"",
		"Top Pop Artists",
		"Popular Indonesian Singers",
		"Trending Music Artists",
		"Best Indie Artists Indonesia",
		"Popular Rock Bands",
		"Top R&B Soul Artists",
		"Acoustic Singer Songwriter",
	}
	excl := map[string]struct{}{}
	for _, id := range excludeBrowseIDs {
		excl[strings.ToLower(strings.TrimSpace(id))] = struct{}{}
	}
	startIdx := 0
	if offset > 0 {
		startIdx = (offset / 12) % len(queries)
	}
	for step := 0; step < len(queries); step++ {
		q := queries[(startIdx+step)%len(queries)]
		if cards, err := a.ytClient.SearchArtists(reqCtx, q); err == nil && len(cards) > 0 {
			out := make([]sources.YTMusicCard, 0, len(cards))
			for _, c := range cards {
				k := strings.ToLower(strings.TrimSpace(c.BrowseID))
				if k == "" {
					k = strings.ToLower(strings.TrimSpace(c.Title))
				}
				if _, skip := excl[k]; !skip {
					out = append(out, c)
				}
			}
			if len(out) > 0 {
				return out, nil
			}
		}
	}
	if a.useDeezerMetadata() {
		if cards, err := a.deezerClient.GetMoreArtists(reqCtx, offset, excludeBrowseIDs); err == nil && len(cards) > 0 {
			return cards, nil
		}
	}
	return []sources.YTMusicCard{}, nil
}

func (a *App) GetYTMusicTop100() ([]sources.YTMusicTrack, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return a.ytClient.GetTop100Tracks(reqCtx)
}

func (a *App) GetYTMusicTop100Recommended(excludeVideoIDs []string) ([]sources.YTMusicTrack, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var seeds []sources.HistorySeed
	if recent, err := a.store.GetRecentSeedTracks(12); err == nil {
		for _, tr := range recent {
			seeds = append(seeds, sources.HistorySeed{
				VideoID: strings.TrimSpace(tr.ExternalID),
				Title:   tr.Title,
				Artist:  tr.ArtistName,
			})
		}
	}
	if tracks, err := a.ytClient.GetHistoryBasedRecommended(reqCtx, seeds, excludeVideoIDs); err == nil && len(tracks) > 0 {
		return tracks, nil
	}
	if a.useDeezerMetadata() {
		if tracks, dErr := a.deezerClient.GetRecommendedTracks(reqCtx, seeds, excludeVideoIDs, 28); dErr == nil && len(tracks) > 0 {
			return tracks, nil
		}
	}
	return []sources.YTMusicTrack{}, nil
}

func (a *App) GetMoreTopSongs(offset int, excludeVideoIDs []string) ([]sources.YTMusicTrack, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if tracks, err := a.ytClient.GetTop100Tracks(reqCtx); err == nil && len(tracks) > 0 {
		excl := map[string]struct{}{}
		for _, id := range excludeVideoIDs {
			excl[strings.TrimSpace(id)] = struct{}{}
		}
		out := make([]sources.YTMusicTrack, 0, len(tracks))
		for _, tr := range tracks {
			if _, skip := excl[tr.VideoID]; !skip {
				out = append(out, tr)
			}
		}
		if len(out) > 0 {
			return out, nil
		}
	}
	if tracks, err := a.ytClient.GetHistoryBasedRecommended(reqCtx, nil, excludeVideoIDs); err == nil && len(tracks) > 0 {
		return tracks, nil
	}
	if a.useDeezerMetadata() {
		if tracks, err := a.deezerClient.GetMoreTopSongs(reqCtx, offset, excludeVideoIDs, 35); err == nil && len(tracks) > 0 {
			return tracks, nil
		}
	}
	return []sources.YTMusicTrack{}, nil
}

func (a *App) GetYTMusicHome() (sources.YTMusicHomeFeed, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	a.ytHomeMu.Lock()
	if a.ytHomeCached && time.Since(a.ytHomeAt) < 10*time.Minute {
		feed := a.ytHomeFeed
		a.ytHomeMu.Unlock()
		return feed, nil
	}
	done := a.ytHomeDone
	a.ytHomeMu.Unlock()

	if done != nil {
		select {
		case <-done:
			a.ytHomeMu.Lock()
			cached := a.ytHomeCached && time.Since(a.ytHomeAt) < 10*time.Minute
			feed := a.ytHomeFeed
			a.ytHomeMu.Unlock()
			if cached {
				return feed, nil
			}
		case <-ctx.Done():
			return sources.YTMusicHomeFeed{}, ctx.Err()
		}
	}

	reqCtx, cancel := context.WithTimeout(ctx, 18*time.Second)
	defer cancel()
	feed, err := a.buildHomeFeed(reqCtx)
	if err == nil && (len(feed.Shelves) > 0 || len(feed.Tracks) > 0) {
		a.ytHomeMu.Lock()
		a.ytHomeFeed = feed
		a.ytHomeErr = nil
		a.ytHomeCached = true
		a.ytHomeAt = time.Now()
		a.ytHomeMu.Unlock()
	}
	return feed, err
}

func (a *App) GetYTMusicHomeContinuation(continuation string) (sources.YTMusicHomeFeed, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return a.ytClient.GetHomeContinuation(reqCtx, continuation)
}

func (a *App) PlayYTMusicCollection(browseID, playlistID, title, subtitle, coverURL string) ([]db.Track, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 14*time.Second)
	defer cancel()
	var ytTracks []sources.YTMusicTrack
	var err error
	colID := strings.TrimSpace(browseID)
	if !strings.HasPrefix(colID, "sp:") && !strings.HasPrefix(colID, "dz:") &&
		(strings.HasPrefix(strings.TrimSpace(playlistID), "sp:") || strings.HasPrefix(strings.TrimSpace(playlistID), "dz:")) {
		colID = strings.TrimSpace(playlistID)
	}
	if strings.HasPrefix(colID, "sp:") && a.spotifyClient != nil && a.spotifyClient.IsConfigured() {
		ytTracks, err = a.spotifyClient.GetCollectionTracks(reqCtx, colID)
	} else if strings.HasPrefix(colID, "dz:") && a.deezerClient != nil {
		ytTracks, err = a.deezerClient.GetCollectionTracks(reqCtx, colID)
	} else {
		ytTracks, err = a.ytClient.GetCollectionTracks(reqCtx, browseID, playlistID, subtitle, title, coverURL)
	}
	if err != nil || len(ytTracks) == 0 {
		return nil, fmt.Errorf("tidak ada lagu ditemukan untuk koleksi ini")
	}
	limit := len(ytTracks)
	if limit > 25 {
		limit = 25
	}
	var saved []db.Track
	for i := 0; i < limit; i++ {
		tr, err := a.SaveYTMusicTrack(ytTracks[i])
		if err == nil {
			saved = append(saved, tr)
		}
	}
	if len(saved) > 0 {
		if resolved, rErr := a.ensureTrackPlayableExternalID(reqCtx, saved[0]); rErr == nil {
			saved[0] = resolved
		}
		a.player.SetCurrentTrack(saved[0])
	}
	return saved, nil
}

type YTMusicAlbumView struct {
	BrowseID       string                `json:"browseId"`
	Kind           string                `json:"kind"`
	Title          string                `json:"title"`
	Artist         string                `json:"artist"`
	ArtistBrowseID string                `json:"artistBrowseId"`
	Year           string                `json:"year"`
	CoverURL       string                `json:"coverUrl"`
	Subtitle       string                `json:"subtitle"`
	Tracks         []db.Track            `json:"tracks"`
	ArtistAlbums   []sources.YTMusicCard `json:"artistAlbums"`
}

func (a *App) GetYTMusicTrackAlbum(videoID, browseID, title, artist, albumTitle, coverURL string) (YTMusicAlbumView, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 16*time.Second)
	defer cancel()

	var detail sources.YTMusicAlbumDetail
	var err error
	bID := strings.TrimSpace(browseID)
	vID := strings.TrimSpace(videoID)
	if strings.HasPrefix(bID, "dz:album:") && a.deezerClient != nil {
		detail, err = a.deezerClient.GetAlbumDetail(reqCtx, bID)
	} else if strings.HasPrefix(vID, "dz:track:") && a.deezerClient != nil {
		if albID, aErr := a.deezerClient.GetTrackAlbumID(reqCtx, vID); aErr == nil && albID != "" {
			detail, err = a.deezerClient.GetAlbumDetail(reqCtx, albID)
		} else {
			err = aErr
		}
	} else if a.spotifyClient != nil && a.spotifyClient.IsConfigured() && strings.HasPrefix(bID, "sp:album:") {
		detail, err = a.spotifyClient.GetAlbumDetail(reqCtx, bID)
	} else if a.spotifyClient != nil && a.spotifyClient.IsConfigured() && strings.HasPrefix(vID, "sp:track:") {
		if albID, aErr := a.spotifyClient.GetTrackAlbumID(reqCtx, vID); aErr == nil && albID != "" {
			detail, err = a.spotifyClient.GetAlbumDetail(reqCtx, albID)
		} else {
			err = aErr
		}
	} else {
		detail, err = a.ytClient.GetTrackAlbumDetail(reqCtx, videoID, browseID, title, artist, albumTitle, coverURL)
	}
	if err != nil {
		return YTMusicAlbumView{}, err
	}

	limit := len(detail.Tracks)
	if limit > 200 {
		limit = 200
	}
	saved := make([]db.Track, 0, limit)
	for i := 0; i < limit; i++ {
		tr, err := a.SaveYTMusicTrack(detail.Tracks[i])
		if err == nil {
			tr.GlobalPlays = detail.Tracks[i].PlayCountText
			saved = append(saved, tr)
		}
	}

	kind := strings.TrimSpace(detail.Kind)
	if kind == "" {
		kind = "album"
	}

	return YTMusicAlbumView{
		BrowseID:       detail.BrowseID,
		Kind:           kind,
		Title:          detail.Title,
		Artist:         detail.Artist,
		ArtistBrowseID: detail.ArtistBrowseID,
		Year:           detail.Year,
		CoverURL:       detail.CoverURL,
		Subtitle:       detail.Subtitle,
		Tracks:         saved,
		ArtistAlbums:   detail.ArtistAlbums,
	}, nil
}

type YTMusicArtistView struct {
	BrowseID       string                `json:"browseId"`
	Name           string                `json:"name"`
	AvatarURL      string                `json:"avatarUrl"`
	FansCount      int64                 `json:"fansCount"`
	TopTracks      []db.Track            `json:"topTracks"`
	Albums         []sources.YTMusicCard `json:"albums"`
	RelatedArtists []sources.YTMusicCard `json:"relatedArtists"`
}

func (a *App) GetYTMusicArtistDetail(browseID, name, avatarURL string) (YTMusicArtistView, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 16*time.Second)
	defer cancel()

	bID := strings.TrimSpace(browseID)
	qName := strings.TrimSpace(name)

	var detail sources.YTMusicArtistDetail
	var err error

	if strings.HasPrefix(bID, "dz:artist:") && a.deezerClient != nil {
		detail, err = a.deezerClient.GetArtistDetail(reqCtx, bID)
	} else if strings.HasPrefix(bID, "sp:artist:") && a.spotifyClient != nil && a.spotifyClient.IsConfigured() {
		detail, err = a.spotifyClient.GetArtistDetail(reqCtx, bID)
	} else if qName != "" {
		if a.useSpotifyMetadata() {
			if hits, sErr := a.spotifyClient.SearchArtists(reqCtx, qName); sErr == nil && len(hits) > 0 {
				detail, err = a.spotifyClient.GetArtistDetail(reqCtx, hits[0].BrowseID)
			}
		}
		if len(detail.TopTracks) == 0 && a.useDeezerMetadata() {
			if hits, dErr := a.deezerClient.SearchArtists(reqCtx, qName); dErr == nil && len(hits) > 0 {
				detail, err = a.deezerClient.GetArtistDetail(reqCtx, hits[0].BrowseID)
			}
		}
		if len(detail.TopTracks) == 0 && a.ytClient != nil {
			trs, _ := a.ytClient.Search(reqCtx, qName)
			albs, _ := a.ytClient.SearchAlbums(reqCtx, qName)
			detail = sources.YTMusicArtistDetail{
				BrowseID:  bID,
				Name:      qName,
				AvatarURL: avatarURL,
				TopTracks: trs,
				Albums:    albs,
			}
		}
	}
	if err != nil && len(detail.TopTracks) == 0 && len(detail.Albums) == 0 {
		return YTMusicArtistView{}, err
	}

	if detail.Name == "" {
		detail.Name = qName
	}
	if detail.AvatarURL == "" {
		detail.AvatarURL = avatarURL
	}

	limit := len(detail.TopTracks)
	if limit > 15 {
		limit = 15
	}
	saved := make([]db.Track, 0, limit)
	for i := 0; i < limit; i++ {
		tr, sErr := a.SaveYTMusicTrack(detail.TopTracks[i])
		if sErr == nil {
			saved = append(saved, tr)
		}
	}

	return YTMusicArtistView{
		BrowseID:       detail.BrowseID,
		Name:           detail.Name,
		AvatarURL:      detail.AvatarURL,
		FansCount:      detail.FansCount,
		TopTracks:      saved,
		Albums:         detail.Albums,
		RelatedArtists: detail.RelatedArtists,
	}, nil
}

func (a *App) tryUpgradeUnsyncedOnce(trackID int64) bool {
	if trackID <= 0 {
		return false
	}
	a.lyricUpgradeMu.Lock()
	defer a.lyricUpgradeMu.Unlock()
	if a.lyricUpgradeTriedMap == nil {
		a.lyricUpgradeTriedMap = make(map[int64]bool)
	}
	if a.lyricUpgradeTriedMap[trackID] {
		return false
	}
	a.lyricUpgradeTriedMap[trackID] = true
	return true
}

func (a *App) nextLyricsReqSeq(trackID int64) uint64 {
	a.lyricsSeqMu.Lock()
	defer a.lyricsSeqMu.Unlock()
	if a.lyricsSeqMap == nil {
		a.lyricsSeqMap = map[int64]uint64{}
	}
	a.lyricsSeqMap[trackID]++
	return a.lyricsSeqMap[trackID]
}

func (a *App) isLatestLyricsReqSeq(trackID int64, seq uint64) bool {
	a.lyricsSeqMu.Lock()
	defer a.lyricsSeqMu.Unlock()
	return a.lyricsSeqMap != nil && a.lyricsSeqMap[trackID] == seq
}

func (a *App) RefreshTrackLyrics(trackID int64, title, artist, album string, durationSec int) sources.TrackLyrics {
	return a.resolveTrackLyrics(trackID, title, artist, album, durationSec, true)
}

func (a *App) GetTrackLyrics(trackID int64, title, artist, album string, durationSec int) sources.TrackLyrics {
	return a.resolveTrackLyrics(trackID, title, artist, album, durationSec, false)
}

func (a *App) resolveTrackLyrics(trackID int64, title, artist, album string, durationSec int, forceRefresh bool) sources.TrackLyrics {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	dbDur := 0
	extID := ""
	if trackID > 0 {
		if tr, trErr := a.store.GetTrackByID(trackID); trErr == nil {
			dbDur = tr.DurationSec
			extID = strings.TrimSpace(tr.ExternalID)
			if strings.TrimSpace(title) == "" {
				title = tr.Title
			}
			if strings.TrimSpace(artist) == "" {
				artist = tr.ArtistName
			}
			if strings.TrimSpace(album) == "" {
				album = tr.AlbumTitle
			}
			if durationSec <= 0 && dbDur > 0 {
				durationSec = dbDur
			}
			if durationSec > 0 && (dbDur <= 0 || math.Abs(float64(durationSec-dbDur)) >= 4.0) && durationSec != dbDur {
				_ = a.store.UpdateTrackDuration(trackID, durationSec)
			}
		}
		if forceRefresh {
			a.store.DeleteCachedLyrics(trackID)
		} else if plain, syncedJSON, source, syncedDurSec, ok := a.store.GetCachedLyrics(trackID); ok {
			var lines []sources.LyricLine
			_ = json.Unmarshal([]byte(syncedJSON), &lines)
			if lines == nil {
				lines = []sources.LyricLine{}
			}
			staleByDuration := durationSec > 0 && (syncedDurSec <= 0 || math.Abs(float64(durationSec-syncedDurSec)) >= 3.0)
			if !staleByDuration && len(lines) > 0 && durationSec > 0 && lines[len(lines)-1].TimeSec > float64(durationSec)+6.0 {
				staleByDuration = true
			}
			needSyncUpgrade := !staleByDuration && len(lines) == 0 && source != "instrumental" && a.tryUpgradeUnsyncedOnce(trackID)
			if !staleByDuration && !needSyncUpgrade {
				a.updateDynamicIslandLyrics(lines, plain, source)
				return sources.TrackLyrics{
					TrackID:      trackID,
					Title:        title,
					Artist:       artist,
					PlainLyrics:  plain,
					SyncedLyrics: lines,
					Source:       source,
				}
			}
		}
	}

	reqSeq := a.nextLyricsReqSeq(trackID)
	res := sources.FetchLyricsWithVideoOpts(ctx, trackID, extID, title, artist, album, durationSec, forceRefresh)

	if res.Source != "instrumental" && extID != "" && (len(res.SyncedLyrics) == 0 || res.NeedsVideoVerify) {
		ytCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
		defer cancel()
		if ytSynced, ytPlain, ok := a.ytClient.GetSyncedOrPlainLyricsByVideoID(ytCtx, extID); ok {
			if len(ytSynced) > 0 {
				ytAligned := sources.AlignLyricsWithVideo(ctx, extID, ytSynced, 0, durationSec)
				if len(res.SyncedLyrics) == 0 {
					res.SyncedLyrics = ytAligned
					res.PlainLyrics = ytPlain
					res.Source = "ytmusic-synced"
				} else if res.NeedsVideoVerify && durationSec > 0 && ytAligned[len(ytAligned)-1].TimeSec <= float64(durationSec)+3.0 {
					res.SyncedLyrics = ytAligned
					res.PlainLyrics = ytPlain
					res.Source = "ytmusic-synced"
				}
			} else if len(res.SyncedLyrics) == 0 && (res.Source == "fallback" || strings.TrimSpace(res.PlainLyrics) == "") && ytPlain != "" {
				res.PlainLyrics = ytPlain
				res.Source = "ytmusic"
			}
		}
	}

	if trackID > 0 && !a.isLatestLyricsReqSeq(trackID, reqSeq) {
		return res
	}
	if trackID > 0 && res.Source != "fallback" {
		if durationSec <= 0 {
			if _, _, _, existingDur, ok := a.store.GetCachedLyrics(trackID); ok && existingDur > 0 {
				a.updateDynamicIslandLyrics(res.SyncedLyrics, res.PlainLyrics, res.Source)
				return res
			}
		}
		b, _ := json.Marshal(res.SyncedLyrics)
		a.store.SaveCachedLyrics(trackID, res.PlainLyrics, string(b), res.Source, durationSec)
	}
	a.updateDynamicIslandLyrics(res.SyncedLyrics, res.PlainLyrics, res.Source)
	return res
}

func (a *App) SaveYTMusicTrack(yt sources.YTMusicTrack) (db.Track, error) {
	isHybridTrack := isHybridTrackID(yt.VideoID)
	if !isHybridTrack && (strings.TrimSpace(yt.Album) == "" || strings.EqualFold(strings.TrimSpace(yt.Album), "YouTube Music") || yt.DurationSec <= 0) {
		ctx := a.ctx
		if ctx == nil {
			ctx = context.Background()
		}
		reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		q := strings.TrimSpace(yt.Title + " " + yt.Artist)
		if q != "" {
			if hits, err := a.ytClient.Search(reqCtx, q); err == nil {
				for _, h := range hits {
					if h.VideoID == yt.VideoID && yt.DurationSec <= 0 && h.DurationSec > 0 {
						yt.DurationSec = h.DurationSec
					}
					if h.Album != "" && !strings.EqualFold(h.Album, "YouTube Music") {
						yt.Album = h.Album
						if yt.CoverURL == "" && h.CoverURL != "" {
							yt.CoverURL = h.CoverURL
						}
						if yt.DurationSec <= 0 && h.DurationSec > 0 {
							yt.DurationSec = h.DurationSec
						}
						break
					}
				}
			}
		}
		if strings.TrimSpace(yt.Album) == "" || strings.EqualFold(strings.TrimSpace(yt.Album), "YouTube Music") {
			yt.Album = "Single"
		}
	} else if strings.TrimSpace(yt.Album) == "" {
		yt.Album = "Single"
	}
	return a.store.UpsertTrack(db.Track{
		Title:       yt.Title,
		ArtistName:  yt.Artist,
		AlbumTitle:  yt.Album,
		DurationSec: yt.DurationSec,
		Format:      yt.Format,
		Bitrate:     160,
		SourceType:  "ytmusic",
		ExternalID:  yt.VideoID,
		CoverURL:    yt.CoverURL,
	})
}

func (a *App) saveYTMusicTrackFast(yt sources.YTMusicTrack) (db.Track, error) {
	album := strings.TrimSpace(yt.Album)
	if album == "" || strings.EqualFold(album, "YouTube Music") {
		album = "Single"
	}
	return a.store.UpsertTrack(db.Track{
		Title:       yt.Title,
		ArtistName:  yt.Artist,
		AlbumTitle:  album,
		DurationSec: yt.DurationSec,
		Format:      yt.Format,
		Bitrate:     160,
		SourceType:  "ytmusic",
		ExternalID:  yt.VideoID,
		CoverURL:    yt.CoverURL,
	})
}

func (a *App) PlayYTMusicTrack(yt sources.YTMusicTrack) (db.Track, error) {
	tr, err := a.SaveYTMusicTrack(yt)
	if err != nil {
		return db.Track{}, err
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if resolved, rErr := a.ensureTrackPlayableExternalID(ctx, tr); rErr == nil {
		tr = resolved
	} else if isHybridTrackID(tr.ExternalID) {
		return db.Track{}, rErr
	}
	a.player.SetCurrentTrack(tr)
	return tr, nil
}

func (a *App) GetStreamURL(trackID int64) (string, error) {
	tr, err := a.store.GetTrackByID(trackID)
	if err != nil {
		return "", err
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if resolved, rErr := a.ensureTrackPlayableExternalID(ctx, tr); rErr == nil {
		tr = resolved
	}
	a.player.SetCurrentTrack(tr)
	return a.player.GetStreamURL(tr), nil
}

type TrackLoudnessResult struct {
	LoudnessDb  float64 `json:"loudnessDb"`
	HasLoudness bool    `json:"hasLoudness"`
}

func (a *App) GetTrackLoudness(trackID int64, externalID string) TrackLoudnessResult {
	if a.store != nil {
		if ldb, ok := a.store.GetTrackLoudness(trackID, externalID); ok {
			return TrackLoudnessResult{LoudnessDb: ldb, HasLoudness: true}
		}
		if externalID == "" && trackID > 0 {
			if tr, err := a.store.GetTrackByID(trackID); err == nil {
				externalID = tr.ExternalID
			}
		}
	}
	ext := strings.TrimSpace(externalID)
	if ext == "" || isHybridTrackID(ext) || a.ytClient == nil {
		return TrackLoudnessResult{}
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	quality := "high"
	if a.store != nil {
		if prefs, err := a.store.GetPreferences(); err == nil && prefs["stream_quality"] != "" {
			quality = prefs["stream_quality"]
		}
	}
	if ldb, ok := a.ytClient.ResolveTrackLoudness(reqCtx, ext, quality); ok {
		if a.store != nil && trackID > 0 {
			a.store.SaveTrackLoudness(trackID, ext, ldb)
		}
		return TrackLoudnessResult{LoudnessDb: ldb, HasLoudness: true}
	}
	return TrackLoudnessResult{}
}

func (a *App) PlayerControl(action string, value float64, mode string) audio.PlaybackState {
	return a.player.Control(action, value, mode)
}

func (a *App) RecordListenProgress(trackID int64, seconds int, completed bool) (stats.FireStreakStatus, error) {
	if err := a.statsSvc.RecordListen(trackID, seconds, completed, time.Now()); err != nil {
		return stats.FireStreakStatus{}, err
	}
	return a.statsSvc.GetFireStreak(time.Now())
}

func (a *App) GetPlayHistory() ([]db.PlayHistoryItem, error) {
	return a.store.GetPlayHistory(100)
}

func (a *App) ClearPlayHistory() error {
	return a.store.ClearPlayHistory()
}

func (a *App) GetListeningStats() (stats.ListeningStats, error) {
	return a.statsSvc.GetListeningStats()
}

func (a *App) GetListeningStatsByDate(dateStr string) (stats.ListeningStats, error) {
	return a.statsSvc.GetListeningStatsByDate(dateStr)
}

func (a *App) GetFireStreak() (stats.FireStreakStatus, error) {
	return a.statsSvc.GetFireStreak(time.Now())
}

func (a *App) GetMonthlyStreak(year int, month int) ([]db.DailyStreakRow, error) {
	return a.statsSvc.GetMonthlyStreak(year, month, time.Now())
}

func (a *App) SavePreferences(prefs map[string]string) error {
	if err := a.store.SavePreferences(prefs); err != nil {
		return err
	}
	if lang, ok := prefs["content.language"]; ok && lang != "" {
		a.SetDynamicIslandLanguage(lang)
	}
	if v, ok := prefs["dynamicIsland.hideApp"]; ok {
		a.SetDynamicIslandHideWhenAppActive(v != "false")
	}
	if v, ok := prefs["dynamicIsland.hideFull"]; ok {
		a.SetDynamicIslandHideInFullscreen(v != "false")
	}
	if v, ok := prefs["dynamicIsland.idleHideSec"]; ok {
		sec, _ := strconv.Atoi(strings.TrimSpace(v))
		a.SetDynamicIslandIdleHideSeconds(sec)
	}
	if v, ok := prefs["dynamicIsland.screen"]; ok {
		a.SetDynamicIslandScreen(v)
	}
	if v, ok := prefs["dynamicIsland.animSpeed"]; ok {
		a.SetDynamicIslandAnimSpeed(v)
	}
	if v, ok := prefs["dynamicIsland.openTrigger"]; ok {
		a.SetDynamicIslandOpenTrigger(v)
	}
	if v, ok := prefs["dynamicIsland.onlyPlaying"]; ok {
		a.SetDynamicIslandOnlyPlaying(strings.EqualFold(strings.TrimSpace(v), "true"))
	}
	if v, ok := prefs["dynamicIsland.pauseHide"]; ok {
		sec, _ := strconv.Atoi(strings.TrimSpace(v))
		a.SetDynamicIslandPauseHideSeconds(sec)
	}
	if v, ok := prefs["system.closeToTray"]; ok {
		a.closeToTray.Store(strings.EqualFold(strings.TrimSpace(v), "true"))
	}
	if v, ok := prefs["integrations.discord.enabled"]; ok && a.discordRPC != nil {
		a.discordRPC.SetEnabled(!strings.EqualFold(strings.TrimSpace(v), "false"))
	}
	if id, ok := prefs["integrations.discord.clientId"]; ok && a.discordRPC != nil {
		a.discordRPC.SetClientID(id)
	}
	if daysStr, ok := prefs["privacy.historyDays"]; ok {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 {
			_ = a.store.PrunePlayHistory(d)
		}
	}
	_, hasCountry := prefs["content.country"]
	_, hasLang := prefs["content.language"]
	if hasCountry || hasLang {
		if all, err := a.store.GetPreferences(); err == nil {
			lang := all["content.language"]
			if strings.TrimSpace(lang) == "" {
				lang = detectSystemLanguage()
			}
			a.applyContentLocale(all["content.country"], lang)
		} else {
			a.applyContentLocale(prefs["content.country"], prefs["content.language"])
		}
		a.invalidateHomeCache()
	}
	_, hasID := prefs["integrations.spotify.clientId"]
	_, hasSecret := prefs["integrations.spotify.clientSecret"]
	_, hasEnabled := prefs["integrations.spotify.enabled"]
	_, hasDeezer := prefs["integrations.deezer.enabled"]
	if hasID || hasSecret || hasEnabled || hasDeezer {
		if all, err := a.store.GetPreferences(); err == nil && a.spotifyClient != nil {
			a.spotifyClient.SetCredentials(all["integrations.spotify.clientId"], all["integrations.spotify.clientSecret"])
		}
		a.invalidateHomeCache()
	}
	return nil
}

func parseContentRegions(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, p := range parts {
		code := strings.ToUpper(strings.TrimSpace(p))
		if len(code) != 2 {
			continue
		}
		if _, dup := seen[code]; dup {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	if len(out) == 0 {
		out = []string{"ID"}
	}
	return out
}

func (a *App) applyContentRegions(raw string) {
	a.applyContentLocale(raw, "")
}

func (a *App) applyContentLocale(rawCountry, rawLang string) {
	regions := parseContentRegions(rawCountry)
	if a.ytClient != nil {
		if strings.TrimSpace(rawLang) != "" {
			a.ytClient.SetLocale(regions, rawLang)
		} else {
			a.ytClient.SetRegions(regions)
		}
	}
	if a.spotifyClient != nil {
		a.spotifyClient.SetRegion(regions[0])
	}
	if a.deezerClient != nil {
		a.deezerClient.SetRegions(regions)
	}
}

func (a *App) GetDataDir() string {
	return a.dataDir
}

func (a *App) OpenExternalURL(rawURL string) error {
	trimmed := strings.TrimSpace(rawURL)
	if !strings.HasPrefix(trimmed, "https://") && !strings.HasPrefix(trimmed, "http://") && !strings.HasPrefix(trimmed, "ms-windows-store://") {
		return fmt.Errorf("invalid url")
	}
	if a.ctx != nil {
		runtime.BrowserOpenURL(a.ctx, trimmed)
		return nil
	}
	return nil
}

func (a *App) GenerateQRDataURL(content string) string {
	return jam.BuildQRDataURL(strings.TrimSpace(content))
}

type UpdateCheckResult struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	HasUpdate      bool   `json:"hasUpdate"`
	ReleaseURL     string `json:"releaseUrl"`
	DownloadURL    string `json:"downloadUrl"`
	StoreURL       string `json:"storeUrl"`
	IsStoreApp     bool   `json:"isStoreApp"`
	ReleaseNotes   string `json:"releaseNotes"`
	PublishedAt    string `json:"publishedAt"`
}

const (
	veloraAppVersion         = "1.2.0"
	veloraFeedbackWebhookURL = "https://script.google.com/macros/s/AKfycbwhnfjZ_CFzPRc-3Yw6rtz39IKqbjRlc7CIz2OOUhe8tk64sDR1y5oMWa4zxgx66j9Cxw/exec"
)

func (a *App) SubmitFeedback(title string, detail string, screenshotName string, screenshotDataURL string) error {
	cleanTitle := strings.TrimSpace(title)
	cleanDetail := strings.TrimSpace(detail)
	if cleanTitle == "" || cleanDetail == "" {
		return fmt.Errorf("judul dan detail masukan wajib diisi")
	}
	if veloraFeedbackWebhookURL == "" || strings.Contains(veloraFeedbackWebhookURL, "YOUR_SCRIPT_ID") {
		return fmt.Errorf("endpoint masukan belum dikonfigurasi")
	}

	reqCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if a.ctx != nil {
		reqCtx, cancel = context.WithTimeout(a.ctx, 20*time.Second)
		defer cancel()
	}

	payload := map[string]string{
		"title":             cleanTitle,
		"message":           cleanDetail,
		"appVersion":        veloraAppVersion,
		"os":                goruntime.GOOS + "/" + goruntime.GOARCH,
		"screenshotName":    strings.TrimSpace(screenshotName),
		"screenshotDataURL": strings.TrimSpace(screenshotDataURL),
	}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, veloraFeedbackWebhookURL, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("gagal terhubung ke server")
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gagal mengirim masukan, silakan coba lagi nanti")
	}
	var parsed struct {
		OK    *bool  `json:"ok"`
		Error string `json:"error"`
	}
	if json.Unmarshal(respBytes, &parsed) == nil && parsed.OK != nil && !*parsed.OK {
		return fmt.Errorf("gagal mengirim masukan, silakan coba lagi nanti")
	}
	return nil
}

func parseSemverParts(v string) [3]int {
	cleaned := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V"))
	if idx := strings.IndexAny(cleaned, "-+"); idx >= 0 {
		cleaned = cleaned[:idx]
	}
	parts := strings.Split(cleaned, ".")
	var out [3]int
	for i := 0; i < 3 && i < len(parts); i++ {
		if n, err := strconv.Atoi(strings.TrimSpace(parts[i])); err == nil {
			out[i] = n
		}
	}
	return out
}

func isNewerVersion(latest, current string) bool {
	a := parseSemverParts(latest)
	b := parseSemverParts(current)
	for i := 0; i < 3; i++ {
		if a[i] > b[i] {
			return true
		}
		if a[i] < b[i] {
			return false
		}
	}
	return false
}

const veloraLatestReleaseAPI = "https://api.github.com/repos/Rafifdev/Velora/releases/latest"

type githubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type githubReleasePayload struct {
	TagName     string               `json:"tag_name"`
	HTMLURL     string               `json:"html_url"`
	Body        string               `json:"body"`
	PublishedAt string               `json:"published_at"`
	Assets      []githubReleaseAsset `json:"assets"`
}

func selectWindowsUpdateAsset(assets []githubReleaseAsset) string {
	var fallbackSetup string
	for _, asset := range assets {
		lower := strings.ToLower(strings.TrimSpace(asset.Name))
		if !strings.HasSuffix(lower, ".exe") || strings.TrimSpace(asset.BrowserDownloadURL) == "" {
			continue
		}
		if lower == "velora.exe" || (!strings.Contains(lower, "setup") && !strings.Contains(lower, "installer")) {
			return strings.TrimSpace(asset.BrowserDownloadURL)
		}
		if fallbackSetup == "" {
			fallbackSetup = strings.TrimSpace(asset.BrowserDownloadURL)
		}
	}
	return fallbackSetup
}

func checkForUpdatesFromURL(endpoint string) (UpdateCheckResult, error) {
	storeURL := getMicrosoftStoreURL()
	storeApp := isStorePackage()
	res := UpdateCheckResult{
		CurrentVersion: veloraAppVersion,
		LatestVersion:  veloraAppVersion,
		ReleaseURL:     "https://github.com/Rafifdev/Velora/releases/latest",
		StoreURL:       storeURL,
		IsStoreApp:     storeApp,
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return res, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Velora/"+veloraAppVersion)

	resp, err := client.Do(req)
	if err != nil {
		return res, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return res, fmt.Errorf("release check failed with HTTP %d", resp.StatusCode)
	}

	var payload githubReleasePayload
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return res, err
	}

	latest := strings.TrimPrefix(strings.TrimSpace(payload.TagName), "v")
	latest = strings.TrimPrefix(latest, "V")
	if latest != "" {
		res.LatestVersion = latest
	}
	if strings.TrimSpace(payload.HTMLURL) != "" {
		res.ReleaseURL = strings.TrimSpace(payload.HTMLURL)
	}
	res.ReleaseNotes = strings.TrimSpace(payload.Body)
	res.PublishedAt = strings.TrimSpace(payload.PublishedAt)
	res.DownloadURL = selectWindowsUpdateAsset(payload.Assets)
	res.HasUpdate = isNewerVersion(res.LatestVersion, veloraAppVersion)
	return res, nil
}

func (a *App) CheckForUpdates() (UpdateCheckResult, error) {
	return checkForUpdatesFromURL(veloraLatestReleaseAPI)
}

func (a *App) InstallUpdate(downloadURL string) error {
	return installAndRestartWindows(a, downloadURL)
}

func (a *App) OpenDataFolder() error {
	if a.dataDir == "" {
		return fmt.Errorf("data folder not initialized")
	}
	_ = os.MkdirAll(a.dataDir, 0o755)
	switch goruntime.GOOS {
	case "windows":
		return exec.Command("explorer.exe", a.dataDir).Start()
	case "darwin":
		return exec.Command("open", a.dataDir).Start()
	default:
		return exec.Command("xdg-open", a.dataDir).Start()
	}
}

func (a *App) ClearCache() (int64, error) {
	var freedBytes int64
	if entries, err := os.ReadDir(a.artworkDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				p := filepath.Join(a.artworkDir, e.Name())
				var sz int64
				if info, iErr := e.Info(); iErr == nil {
					sz = info.Size()
				}
				if os.Remove(p) == nil {
					freedBytes += sz
				}
			}
		}
	}
	if tmpEntries, err := os.ReadDir(os.TempDir()); err == nil {
		for _, e := range tmpEntries {
			if !e.IsDir() && strings.HasPrefix(e.Name(), "velora-transcode-") {
				p := filepath.Join(os.TempDir(), e.Name())
				var sz int64
				if info, iErr := e.Info(); iErr == nil {
					sz = info.Size()
				}
				if os.Remove(p) == nil {
					freedBytes += sz
				}
			}
		}
	}
	if a.ytClient != nil {
		a.ytClient.ClearAllStreamCache()
	}
	a.invalidateHomeCache()
	return freedBytes, nil
}

func (a *App) WindowMinimize() {
	runtime.WindowMinimise(a.ctx)
}

func (a *App) SetWindowTitle(title string) {
	if a.ctx == nil {
		return
	}
	t := strings.TrimSpace(title)
	if t == "" {
		t = "Velora"
	}
	runtime.WindowSetTitle(a.ctx, t)
}

func (a *App) SetKeepAwake(enabled bool) {
	setSystemKeepAwake(enabled)
}

func (a *App) WindowMaximize() {
	runtime.WindowToggleMaximise(a.ctx)
}

func (a *App) WindowIsMaximized() bool {
	if a.ctx == nil {
		return false
	}
	return runtime.WindowIsMaximised(a.ctx)
}

func (a *App) WindowClose() {
	if a.closeToTray.Load() && !a.quitting.Load() {
		go func() {
			if a.ctx != nil {
				a.beforeClose(a.ctx)
			}
		}()
		return
	}
	go a.forceQuitApp()
}

type DownloadCollectionItem struct {
	TrackID     int64  `json:"trackId"`
	ExternalID  string `json:"externalId"`
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Album       string `json:"album"`
	DurationSec int    `json:"durationSec"`
	CoverURL    string `json:"coverUrl"`
	Format      string `json:"format"`
}

func (a *App) DownloadTrack(trackID int64) (db.Track, error) {
	tr, err := a.store.GetTrackByID(trackID)
	if err != nil {
		return db.Track{}, err
	}
	a.store.ClearTrackDownloadGroup(tr.ID)
	return a.downloader.Enqueue(tr)
}

func (a *App) RetryDownloadTrack(trackID int64, group sources.DownloadGroup) (db.Track, error) {
	tr, err := a.store.GetTrackByID(trackID)
	if err != nil {
		return db.Track{}, err
	}
	if strings.TrimSpace(group.Key) != "" {
		a.store.SetTracksDownloadGroup([]int64{tr.ID}, group.Key, group.Title, group.Cover, group.Kind)
	}
	return a.downloader.Enqueue(tr)
}

func (a *App) DownloadYTMusicTrack(yt sources.YTMusicTrack) (db.Track, error) {
	tr, err := a.SaveYTMusicTrack(yt)
	if err != nil {
		return db.Track{}, err
	}
	a.store.ClearTrackDownloadGroup(tr.ID)
	return a.downloader.Enqueue(tr)
}

func (a *App) DownloadCollection(group sources.DownloadGroup, items []DownloadCollectionItem) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}
	batch := make([]db.Track, 0, len(items))
	for _, it := range items {
		if it.TrackID > 0 {
			if tr, err := a.store.GetTrackByID(it.TrackID); err == nil {
				batch = append(batch, tr)
			}
			continue
		}
		if strings.TrimSpace(it.ExternalID) != "" {
			tr, err := a.saveYTMusicTrackFast(sources.YTMusicTrack{
				VideoID:     it.ExternalID,
				Title:       it.Title,
				Artist:      it.Artist,
				Album:       it.Album,
				DurationSec: it.DurationSec,
				CoverURL:    it.CoverURL,
				Format:      it.Format,
			})
			if err == nil {
				batch = append(batch, tr)
			}
		}
	}
	if len(batch) == 0 {
		return 0, nil
	}
	a.downloader.EnqueueBatch(batch, group)
	return len(batch), nil
}

func (a *App) CancelDownloadGroup(groupKey string) {
	a.downloader.CancelGroup(groupKey)
}

func (a *App) PauseDownload(trackID int64) {
	a.downloader.Pause(trackID)
}

func (a *App) ResumeDownload(trackID int64) {
	a.downloader.Resume(trackID)
}

func (a *App) PauseDownloadGroup(groupKey string) {
	a.downloader.PauseGroup(groupKey)
}

func (a *App) ResumeDownloadGroup(groupKey string) {
	a.downloader.ResumeGroup(groupKey)
}

func (a *App) RemoveDownloadedTrack(trackID int64) (db.Track, error) {
	return a.downloader.Remove(trackID)
}

func (a *App) ClearAllDownloads() (int64, error) {
	if a.downloader == nil {
		return 0, nil
	}
	return a.downloader.RemoveAll()
}

func (a *App) RemoveDownloadedTracks(trackIDs []int64) error {
	for _, id := range trackIDs {
		if id > 0 {
			_, _ = a.downloader.Remove(id)
		}
	}
	return nil
}

func (a *App) CancelDownload(trackID int64) {
	a.downloader.Cancel(trackID)
}

func (a *App) GetDownloadedTracks() ([]db.Track, error) {
	return a.store.GetDownloadedTracks()
}

func (a *App) GetDownloadStates() []sources.DownloadProgress {
	return a.downloader.GetStates()
}

func (a *App) StartJam() (jam.Status, error) {
	if a.jamSrv == nil {
		return jam.Status{}, fmt.Errorf("jam server not initialized")
	}
	st, err := a.jamSrv.Start()
	if err == nil && a.store != nil {
		_ = a.store.SetPreference("integrations.jam.remoteUrl", st.RemoteURL)
		_ = a.store.SetPreference("integrations.jam.apiUrl", st.APIURL)
	}
	if err == nil && a.trayCtrl != nil {
		a.trayCtrl.EnsureStarted(st.Running, st.RemoteURL)
	}
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "jam:status", st)
	}
	return st, err
}

func (a *App) StopJam() jam.Status {
	if a.jamSrv == nil {
		return jam.Status{}
	}
	st := a.jamSrv.Stop()
	if a.store != nil {
		_ = a.store.SetPreference("integrations.jam.remoteUrl", "")
		_ = a.store.SetPreference("integrations.jam.apiUrl", "")
	}
	if a.trayCtrl != nil {
		a.trayCtrl.UpdateJamState(false, st.RemoteURL)
	}
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "jam:status", st)
	}
	return st
}

func (a *App) GetJamStatus() jam.Status {
	if a.jamSrv == nil {
		return jam.Status{}
	}
	return a.jamSrv.Status()
}

func (a *App) UpdateJamState(snap jam.PlaybackSnapshot) {
	if a.jamSrv != nil {
		a.jamSrv.UpdateState(snap)
	}
}

func (a *App) syncDiscordPresence(p DynamicIslandSyncPayload) {
	if a.discordRPC == nil {
		return
	}
	extID := ""
	if p.QueueIndex >= 0 && p.QueueIndex < len(p.Queue) {
		extID = p.Queue[p.QueueIndex].ExternalID
	}
	a.discordRPC.UpdatePresence(discord.TrackPresence{
		Title:       p.Title,
		Artist:      p.Artist,
		Album:       p.Album,
		CoverURL:    p.CoverURL,
		ExternalID:  extID,
		IsPlaying:   p.IsPlaying,
		PositionSec: p.PositionSec,
		DurationSec: p.DurationSec,
	})
}

func (a *App) updateDiscordProgress(positionSec, durationSec float64) {
	if a.discordRPC != nil {
		a.discordRPC.UpdateProgress(positionSec, durationSec)
	}
}

func (a *App) handleJamRPC(method string, rawArgs []json.RawMessage) (any, error) {
	switch method {
	case "WindowMinimize", "WindowMaximize", "WindowIsMaximized", "WindowClose", "SetFrontendFS", "StartJam", "StopJam", "UpdateJamState":
		return nil, fmt.Errorf("method %q is not permitted over remote RPC", method)
	}

	m := reflect.ValueOf(a).MethodByName(method)
	if !m.IsValid() {
		return nil, fmt.Errorf("unknown method %q", method)
	}

	mt := m.Type()
	numIn := mt.NumIn()
	if len(rawArgs) != numIn {
		return nil, fmt.Errorf("method %q expects %d args, got %d", method, numIn, len(rawArgs))
	}

	in := make([]reflect.Value, numIn)
	for i := 0; i < numIn; i++ {
		argVal := reflect.New(mt.In(i))
		if len(rawArgs[i]) > 0 {
			if err := json.Unmarshal(rawArgs[i], argVal.Interface()); err != nil {
				return nil, fmt.Errorf("invalid arg %d for %q: %w", i, method, err)
			}
		}
		in[i] = argVal.Elem()
	}

	out := m.Call(in)
	if len(out) == 0 {
		return nil, nil
	}
	errType := reflect.TypeOf((*error)(nil)).Elem()
	last := out[len(out)-1]
	if last.Type().Implements(errType) {
		var retErr error
		if !last.IsNil() {
			retErr = last.Interface().(error)
		}
		if len(out) == 1 {
			return nil, retErr
		}
		val := out[0].Interface()
		if method == "GetBootstrapData" && retErr == nil {
			if boot, ok := val.(BootstrapData); ok {
				boot.MediaBaseURL = ""
				return boot, nil
			}
		}
		return val, retErr
	}
	return out[0].Interface(), nil
}
