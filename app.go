package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"velora/internal/audio"
	"velora/internal/db"
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
	frontendFS   fs.FS
	trayIcon     []byte
	trayCtrl     *tray.Controller
	quitting     bool
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
	if prefs, pErr := a.store.GetPreferences(); pErr == nil {
		a.spotifyClient.SetCredentials(prefs["integrations.spotify.clientId"], prefs["integrations.spotify.clientSecret"])
	}
	a.downloader = sources.NewDownloadManager(store, a.ytClient, a.downloadsDir, func(eventName string, data any) {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, eventName, data)
		}
	})
	a.startYTMusicHomePrefetch()
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
			a.quitting = true
			if a.ctx != nil {
				runtime.Quit(a.ctx)
			}
		},
	)
	if prefs, pErr := a.store.GetPreferences(); pErr == nil && prefs["integrations.jam.enabled"] == "true" {
		_, _ = a.StartJam()
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
		reqCtx, cancel := context.WithTimeout(context.Background(), 14*time.Second)
		defer cancel()
		feed, err := a.buildHomeFeed(reqCtx)
		a.ytHomeMu.Lock()
		a.ytHomeFeed = feed
		a.ytHomeErr = err
		a.ytHomeCached = err == nil && (len(feed.Shelves) > 0 || len(feed.Tracks) > 0)
		a.ytHomeMu.Unlock()
	}()
}

func (a *App) buildHomeFeed(ctx context.Context) (sources.YTMusicHomeFeed, error) {
	var feed sources.YTMusicHomeFeed
	var err error
	if a.useSpotifyMetadata() {
		feed, err = a.spotifyClient.GetHomeFeed(ctx)
	}
	if (err != nil || (len(feed.Shelves) == 0 && len(feed.Tracks) == 0)) && a.useDeezerMetadata() {
		feed, err = a.deezerClient.GetHomeFeed(ctx)
		if err == nil && a.store != nil {
			var seeds []sources.HistorySeed
			if recent, sErr := a.store.GetRecentSeedTracks(12); sErr == nil {
				for _, tr := range recent {
					seeds = append(seeds, sources.HistorySeed{
						VideoID: strings.TrimSpace(tr.ExternalID),
						Title:   tr.Title,
						Artist:  tr.ArtistName,
					})
				}
			}
			if recTracks, rErr := a.deezerClient.GetRecommendedTracks(ctx, seeds, nil, 28); rErr == nil && len(recTracks) > 0 {
				items := make([]sources.YTMusicCard, 0, len(recTracks))
				for _, t := range recTracks {
					items = append(items, sources.YTMusicCard{
						ID:          "song:" + t.VideoID,
						Title:       t.Title,
						Subtitle:    t.Artist,
						CoverURL:    t.CoverURL,
						Kind:        "song",
						VideoID:     t.VideoID,
						Artist:      t.Artist,
						Album:       t.Album,
						DurationSec: t.DurationSec,
					})
				}
				feed.Shelves = append(feed.Shelves, sources.YTMusicShelf{
					Title: "Lagu Lainnya untuk Anda",
					Items: items,
				})
				feed.Tracks = recTracks
			}
		}
	}
	if err != nil || (len(feed.Shelves) == 0 && len(feed.Tracks) == 0) {
		feed, err = a.ytClient.GetHomeFeed(ctx)
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
	a.ytHomeFeed = sources.YTMusicHomeFeed{}
	a.ytHomeErr = nil
	a.ytHomeDone = nil
	a.ytHomeMu.Unlock()
}

func (a *App) ensureTrackPlayableExternalID(ctx context.Context, tr db.Track) (db.Track, error) {
	extID := strings.TrimSpace(tr.ExternalID)
	if !isHybridTrackID(extID) {
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
		vid, err = a.deezerClient.ResolveTrackToYTMusic(reqCtx, a.ytClient, extID, tr.Title, tr.ArtistName)
	} else if !strings.HasPrefix(extID, "sp:import:") && a.spotifyClient != nil {
		vid, err = a.spotifyClient.ResolveTrackToYTMusic(reqCtx, a.ytClient, extID, tr.Title, tr.ArtistName)
	} else {
		err = fmt.Errorf("fallback yt search")
	}
	if err != nil || strings.TrimSpace(vid) == "" {
		q := strings.TrimSpace(tr.Title + " " + tr.ArtistName)
		if hits, sErr := a.ytClient.Search(reqCtx, q); sErr == nil && len(hits) > 0 && strings.TrimSpace(hits[0].VideoID) != "" {
			vid = strings.TrimSpace(hits[0].VideoID)
			err = nil
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
}

func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	if !a.quitting && a.jamSrv != nil && a.jamSrv.Status().Running {
		runtime.WindowHide(ctx)
		return true
	}
	return false
}

func (a *App) shutdown(ctx context.Context) {
	a.shutdownTaskbar()
	a.shutdownDynamicIsland()
	if a.trayCtrl != nil {
		a.trayCtrl.Shutdown()
	}
	if a.jamSrv != nil {
		a.jamSrv.Stop()
	}
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
	return a.ytClient.Search(reqCtx, query)
}

func (a *App) SearchYouTubeMusicPlaylists(query string) ([]sources.YTMusicCard, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
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
	return a.ytClient.SearchPlaylists(reqCtx, query)
}

func (a *App) SearchYouTubeMusicAlbums(query string) ([]sources.YTMusicCard, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
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
	return a.ytClient.SearchAlbums(reqCtx, query)
}

func (a *App) SearchYouTubeMusicArtists(query string) ([]sources.YTMusicCard, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
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
	return a.ytClient.SearchArtists(reqCtx, query)
}

func (a *App) GetMoreArtists(offset int, excludeBrowseIDs []string) ([]sources.YTMusicCard, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if a.useDeezerMetadata() {
		if cards, err := a.deezerClient.GetMoreArtists(reqCtx, offset, excludeBrowseIDs); err == nil && len(cards) > 0 {
			return cards, nil
		}
	}
	if a.useSpotifyMetadata() {
		fallbackQ := []string{"pop", "rock", "rnb", "indie"}[(offset/28)%4]
		if cards, err := a.spotifyClient.SearchArtists(reqCtx, fallbackQ); err == nil && len(cards) > 0 {
			return cards, nil
		}
	}
	fallbackQ := []string{"Popular Indonesian Singers", "Top Global Pop Artists", "Best Indie Artists Indonesia"}[(offset/28)%3]
	return a.ytClient.SearchArtists(reqCtx, fallbackQ)
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
	if a.useDeezerMetadata() {
		if tracks, dErr := a.deezerClient.GetRecommendedTracks(reqCtx, seeds, excludeVideoIDs, 28); dErr == nil && len(tracks) > 0 {
			return tracks, nil
		}
	}
	return a.ytClient.GetHistoryBasedRecommended(reqCtx, seeds, excludeVideoIDs)
}

func (a *App) GetMoreTopSongs(offset int, excludeVideoIDs []string) ([]sources.YTMusicTrack, error) {
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	if a.useDeezerMetadata() {
		if tracks, err := a.deezerClient.GetMoreTopSongs(reqCtx, offset, excludeVideoIDs, 35); err == nil && len(tracks) > 0 {
			return tracks, nil
		}
	}
	return a.ytClient.GetHistoryBasedRecommended(reqCtx, nil, excludeVideoIDs)
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
	feed, err := a.buildHomeFeed(reqCtx)
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
	if limit > 35 {
		limit = 35
	}
	saved := make([]db.Track, 0, limit)
	for i := 0; i < limit; i++ {
		tr, err := a.SaveYTMusicTrack(detail.Tracks[i])
		if err == nil {
			tr.GlobalPlays = detail.Tracks[i].PlayCountText
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
			a.updateDynamicIslandLyrics(lines, plain)
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
	a.updateDynamicIslandLyrics(res.SyncedLyrics, res.PlainLyrics)
	return res
}

func (a *App) SaveYTMusicTrack(yt sources.YTMusicTrack) (db.Track, error) {
	isHybridTrack := isHybridTrackID(yt.VideoID)
	if !isHybridTrack && (strings.TrimSpace(yt.Album) == "" || strings.EqualFold(strings.TrimSpace(yt.Album), "YouTube Music")) {
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
					if h.Album != "" && !strings.EqualFold(h.Album, "YouTube Music") {
						yt.Album = h.Album
						if yt.CoverURL == "" && h.CoverURL != "" {
							yt.CoverURL = h.CoverURL
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
	if err := a.store.SavePreferences(prefs); err != nil {
		return err
	}
	if lang, ok := prefs["content.language"]; ok && lang != "" {
		a.SetDynamicIslandLanguage(lang)
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

func (a *App) DownloadTrack(trackID int64) (db.Track, error) {
	tr, err := a.store.GetTrackByID(trackID)
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
	return a.downloader.Enqueue(tr)
}

func (a *App) DownloadYTMusicTrack(yt sources.YTMusicTrack) (db.Track, error) {
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
	return a.downloader.Enqueue(tr)
}

func (a *App) RemoveDownloadedTrack(trackID int64) (db.Track, error) {
	return a.downloader.Remove(trackID)
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
		_ = a.store.SetPreference("integrations.jam.enabled", "true")
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
		_ = a.store.SetPreference("integrations.jam.enabled", "false")
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
