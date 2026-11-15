package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"velora/internal/audio"
	"velora/internal/db"
	"velora/internal/sources"
	"velora/internal/stats"
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
	ctx        context.Context
	dataDir    string
	uploadsDir string
	artworkDir string
	store      *db.Store
	statsSvc   *stats.Service
	localSrc   *sources.LocalScanner
	ytClient     *sources.YTMusicClient
	player       *audio.PlayerController
	ytHomeMu     sync.Mutex
	ytHomeDone   chan struct{}
	ytHomeFeed   sources.YTMusicHomeFeed
	ytHomeErr    error
	ytHomeCached bool
}

func NewApp() *App {
	return &App{}
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
	_ = os.MkdirAll(a.dataDir, 0o755)
	_ = os.MkdirAll(a.uploadsDir, 0o755)
	_ = os.MkdirAll(a.artworkDir, 0o755)

	store, err := db.Open(filepath.Join(a.dataDir, "velora.db"))
	if err != nil {
		panic(fmt.Errorf("failed to open velora.db: %w", err))
	}
	_ = store.SeedDefaults()

	a.store = store
	a.statsSvc = stats.NewService(store)
	a.localSrc = sources.NewLocalScanner(store, a.uploadsDir, a.artworkDir)
	a.ytClient = sources.NewYTMusicClient()
	a.startYTMusicHomePrefetch()
	a.player = audio.NewPlayerController(store, a.ytClient, a.artworkDir)
	_, _ = a.player.StartMediaServer()
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
		reqCtx, cancel := context.WithTimeout(context.Background(), 14*time.Second)
		defer cancel()
		feed, err := a.ytClient.GetHomeFeed(reqCtx)
		a.ytHomeMu.Lock()
		a.ytHomeFeed = feed
		a.ytHomeErr = err
		a.ytHomeCached = err == nil && (len(feed.Shelves) > 0 || len(feed.Tracks) > 0)
		a.ytHomeMu.Unlock()
	}()
}

func (a *App) domReady(ctx context.Context) {
	a.initTaskbar(ctx)
}

func (a *App) shutdown(ctx context.Context) {
	a.shutdownTaskbar()
	if a.player != nil {
		a.player.Shutdown()
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

func (a *App) ToggleFavoriteArtist(id int64) (bool, error) {
	return a.store.ToggleFavoriteArtist(id)
}

func (a *App) GetPlaylists(localOnly bool) ([]db.Playlist, error) {
	return a.store.GetPlaylists(localOnly)
}

func (a *App) GetPlaylistDetail(id int64) (db.PlaylistDetail, error) {
	return a.store.GetPlaylistDetail(id)
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

func (a *App) UploadLocalFiles() ([]db.Track, error) {
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Upload Audio Files ke Velora",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Audio Files (MP3, FLAC, M4A, AAC, OPUS, OGG, WAV, WebM)",
				Pattern:     "*.mp3;*.flac;*.m4a;*.aac;*.ogg;*.opus;*.wav;*.wma;*.webm",
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
	return a.ytClient.Search(reqCtx, query)
}

func (a *App) SearchYouTubeMusicPlaylists(query string) ([]sources.YTMusicCard, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	return a.ytClient.SearchPlaylists(reqCtx, query)
}

func (a *App) GetYTMusicHome() (sources.YTMusicHomeFeed, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	a.ytHomeMu.Lock()
	if a.ytHomeCached {
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
			cached := a.ytHomeCached
			feed := a.ytHomeFeed
			a.ytHomeMu.Unlock()
			if cached {
				return feed, nil
			}
		case <-ctx.Done():
			return sources.YTMusicHomeFeed{}, ctx.Err()
		}
	}

	reqCtx, cancel := context.WithTimeout(ctx, 14*time.Second)
	defer cancel()
	feed, err := a.ytClient.GetHomeFeed(reqCtx)
	if err == nil && (len(feed.Shelves) > 0 || len(feed.Tracks) > 0) {
		a.ytHomeMu.Lock()
		a.ytHomeFeed = feed
		a.ytHomeErr = nil
		a.ytHomeCached = true
		a.ytHomeMu.Unlock()
	}
	return feed, err
}

func (a *App) PlayYTMusicCollection(browseID, playlistID, title, subtitle, coverURL string) ([]db.Track, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 14*time.Second)
	defer cancel()
	ytTracks, err := a.ytClient.GetCollectionTracks(reqCtx, browseID, playlistID, subtitle, title, coverURL)
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
		a.player.SetCurrentTrack(saved[0])
	}
	return saved, nil
}

type YTMusicAlbumView struct {
	BrowseID       string                `json:"browseId"`
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

	detail, err := a.ytClient.GetTrackAlbumDetail(reqCtx, videoID, browseID, title, artist, albumTitle, coverURL)
	if err != nil {
		return YTMusicAlbumView{}, err
	}

	limit := len(detail.Tracks)
	if limit > 35 {
		limit = 35
	}
	saved := make([]db.Track, 0, limit)
	for i := 0; i < limit; i++ {
		tr, err := a.SaveYTMusicTrack(detail.Tracks[i])
		if err == nil {
			saved = append(saved, tr)
		}
	}

	return YTMusicAlbumView{
		BrowseID:       detail.BrowseID,
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

func (a *App) GetTrackLyrics(trackID int64, title, artist, album string, durationSec int) sources.TrackLyrics {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}

	if trackID > 0 {
		if plain, syncedJSON, source, ok := a.store.GetCachedLyrics(trackID); ok {
			var lines []sources.LyricLine
			_ = json.Unmarshal([]byte(syncedJSON), &lines)
			if lines == nil {
				lines = []sources.LyricLine{}
			}
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

	res := sources.FetchLyrics(ctx, trackID, title, artist, album, durationSec)

	// Fallback to official YouTube Music lyrics tab (MPLYt_...) if not found in LRCLIB/NetEase/KuGou/OVH
	if res.Source == "fallback" && trackID > 0 {
		if tr, err := a.store.GetTrackByID(trackID); err == nil && strings.TrimSpace(tr.ExternalID) != "" {
			ytCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
			defer cancel()
			if ytLyrics, ok := a.ytClient.GetLyricsByVideoID(ytCtx, tr.ExternalID); ok {
				res.PlainLyrics = ytLyrics
				res.Source = "ytmusic"
			}
		}
	}

	if trackID > 0 && res.Source != "fallback" {
		b, _ := json.Marshal(res.SyncedLyrics)
		a.store.SaveCachedLyrics(trackID, res.PlainLyrics, string(b), res.Source)
	}
	return res
}

func (a *App) SaveYTMusicTrack(yt sources.YTMusicTrack) (db.Track, error) {
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

func (a *App) PlayYTMusicTrack(yt sources.YTMusicTrack) (db.Track, error) {
	tr, err := a.SaveYTMusicTrack(yt)
	if err != nil {
		return db.Track{}, err
	}
	a.player.SetCurrentTrack(tr)
	return tr, nil
}

func (a *App) GetStreamURL(trackID int64) (string, error) {
	tr, err := a.store.GetTrackByID(trackID)
	if err != nil {
		return "", err
	}
	a.player.SetCurrentTrack(tr)
	return a.player.GetStreamURL(tr), nil
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

func (a *App) GetFireStreak() (stats.FireStreakStatus, error) {
	return a.statsSvc.GetFireStreak(time.Now())
}

func (a *App) SavePreferences(prefs map[string]string) error {
	return a.store.SavePreferences(prefs)
}

func (a *App) ClearCache() (int, error) {
	entries, err := os.ReadDir(a.artworkDir)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, e := range entries {
		if !e.IsDir() {
			if os.Remove(filepath.Join(a.artworkDir, e.Name())) == nil {
				removed++
			}
		}
	}
	return removed, nil
}

func (a *App) WindowMinimize() {
	runtime.WindowMinimise(a.ctx)
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
	runtime.Quit(a.ctx)
}
