package sources

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"

	"velora/internal/db"
)

var SupportedAudioExtensions = map[string]string{
	".mp3":  "MP3",
	".mp2":  "MP2",
	".mp1":  "MP1",
	".mpa":  "MPA",
	".flac": "FLAC",
	".m4a":  "M4A",
	".m4b":  "M4B",
	".m4p":  "M4P",
	".mp4":  "MP4",
	".alac": "ALAC",
	".aac":  "AAC",
	".ogg":  "OGG",
	".oga":  "OGA",
	".opus": "OPUS",
	".spx":  "SPX",
	".wav":  "WAV",
	".wave": "WAV",
	".aiff": "AIFF",
	".aif":  "AIFF",
	".aifc": "AIFC",
	".caf":  "CAF",
	".au":   "AU",
	".snd":  "SND",
	".wma":  "WMA",
	".asf":  "ASF",
	".webm": "WEBM",
	".mka":  "MKA",
	".ape":  "APE",
	".wv":   "WV",
	".mpc":  "MPC",
	".tak":  "TAK",
	".tta":  "TTA",
	".dsf":  "DSF",
	".dff":  "DFF",
	".ac3":  "AC3",
	".dts":  "DTS",
	".amr":  "AMR",
	".awb":  "AWB",
	".3gp":  "3GP",
	".mid":  "MIDI",
	".midi": "MIDI",
}

type LocalScanner struct {
	store      *db.Store
	uploadsDir string
	artworkDir string
}

func NewLocalScanner(store *db.Store, uploadsDir, artworkDir string) *LocalScanner {
	_ = os.MkdirAll(uploadsDir, 0o755)
	_ = os.MkdirAll(artworkDir, 0o755)
	return &LocalScanner{
		store:      store,
		uploadsDir: uploadsDir,
		artworkDir: artworkDir,
	}
}

func IsSupportedAudioFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := SupportedAudioExtensions[ext]
	return ok
}

func (l *LocalScanner) ImportFiles(paths []string, copyToUploads bool) ([]db.Track, error) {
	imported := []db.Track{}
	for _, srcPath := range paths {
		if !IsSupportedAudioFile(srcPath) {
			continue
		}
		targetPath := srcPath
		if copyToUploads {
			base := filepath.Base(srcPath)
			sum := sha1.Sum([]byte(srcPath))
			dstName := fmt.Sprintf("%s_%s", hex.EncodeToString(sum[:4]), base)
			dstPath := filepath.Join(l.uploadsDir, dstName)
			if err := copyFileIfNeeded(srcPath, dstPath); err == nil {
				targetPath = dstPath
			}
		}
		tr, err := l.indexSingleFile(targetPath)
		if err == nil {
			imported = append(imported, tr)
		}
	}

	if copyToUploads && len(imported) > 0 {
		allLocal, err := l.store.GetTracks("local", "")
		if err == nil {
			_, _ = l.store.AddSource("Uploaded Local Files", "local_upload", l.uploadsDir, len(allLocal))
		}
	}
	return imported, nil
}

func (l *LocalScanner) ScanFolder(folderPath string) ([]db.Track, error) {
	if strings.TrimSpace(folderPath) == "" {
		return nil, fmt.Errorf("folder path is empty")
	}
	var paths []string
	err := filepath.WalkDir(folderPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && IsSupportedAudioFile(path) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	tracks, err := l.ImportFiles(paths, false)
	if err != nil {
		return nil, err
	}
	_, _ = l.store.AddSource(filepath.Base(folderPath), "local_folder", folderPath, len(tracks))
	return tracks, nil
}

func (l *LocalScanner) indexSingleFile(filePath string) (db.Track, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	format := SupportedAudioExtensions[ext]
	if format == "" {
		format = strings.TrimPrefix(strings.ToUpper(ext), ".")
	}

	baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	title := baseName
	artist := "Unknown Artist"
	album := "Local Library"
	coverURL := ""

	f, err := os.Open(filePath)
	if err == nil {
		defer f.Close()
		if meta, err := tag.ReadFrom(f); err == nil {
			if strings.TrimSpace(meta.Title()) != "" {
				title = strings.TrimSpace(meta.Title())
			}
			if strings.TrimSpace(meta.Artist()) != "" {
				artist = strings.TrimSpace(meta.Artist())
			}
			if strings.TrimSpace(meta.Album()) != "" {
				album = strings.TrimSpace(meta.Album())
			}
			if pic := meta.Picture(); pic != nil && len(pic.Data) > 0 {
				coverExt := ".jpg"
				if strings.Contains(strings.ToLower(pic.MIMEType), "png") {
					coverExt = ".png"
				}
				h := sha1.Sum(pic.Data)
				artFile := hex.EncodeToString(h[:8]) + coverExt
				artPath := filepath.Join(l.artworkDir, artFile)
				if _, statErr := os.Stat(artPath); os.IsNotExist(statErr) {
					_ = os.WriteFile(artPath, pic.Data, 0o644)
				}
				coverURL = "/media/art/" + artFile
			}
		}
	}

	if artist == "Unknown Artist" && strings.Contains(baseName, " - ") {
		parts := strings.SplitN(baseName, " - ", 2)
		if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && strings.TrimSpace(parts[1]) != "" {
			artist = strings.TrimSpace(parts[0])
			title = strings.TrimSpace(parts[1])
		}
	}

	durationSec := estimateDurationSec(filePath, format)

	return l.store.UpsertTrack(db.Track{
		Title:       title,
		ArtistName:  artist,
		AlbumTitle:  album,
		DurationSec: durationSec,
		Format:      format,
		Bitrate:     320,
		SourceType:  "local",
		FilePath:    filePath,
		CoverURL:    coverURL,
	})
}

func estimateDurationSec(filePath, format string) int {
	info, err := os.Stat(filePath)
	if err != nil || info.Size() <= 0 {
		return 180
	}
	bytesPerSec := int64(24000) // ~192kbps average fallback
	switch format {
	case "WAV":
		bytesPerSec = 176400
	case "FLAC":
		bytesPerSec = 110000
	case "MP3", "M4A", "AAC":
		bytesPerSec = 32000
	case "OPUS", "OGG":
		bytesPerSec = 20000
	}
	est := int(info.Size() / bytesPerSec)
	if est < 15 {
		est = 15
	}
	if est > 1800 {
		est = 1800
	}
	return est
}

func copyFileIfNeeded(src, dst string) error {
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func ExportPlaylistM3U(detail db.PlaylistDetail, targetPath string) error {
	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	b.WriteString(fmt.Sprintf("#PLAYLIST:%s\n", detail.Playlist.Name))
	for _, t := range detail.Tracks {
		b.WriteString(fmt.Sprintf("#EXTINF:%d,%s - %s\n", t.DurationSec, t.ArtistName, t.Title))
		if t.FilePath != "" {
			b.WriteString(t.FilePath + "\n")
		} else if t.ExternalID != "" {
			b.WriteString("https://music.youtube.com/watch?v=" + t.ExternalID + "\n")
		}
	}
	return os.WriteFile(targetPath, []byte(b.String()), 0o644)
}

func (l *LocalScanner) ImportPlaylistM3U(m3uPath string) (db.Playlist, int, error) {
	raw, err := os.ReadFile(m3uPath)
	if err != nil {
		return db.Playlist{}, 0, err
	}
	name := strings.TrimSuffix(filepath.Base(m3uPath), filepath.Ext(m3uPath))
	var paths []string
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#PLAYLIST:") {
			if n := strings.TrimSpace(strings.TrimPrefix(line, "#PLAYLIST:")); n != "" {
				name = n
			}
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "http") {
			if !filepath.IsAbs(line) {
				line = filepath.Join(filepath.Dir(m3uPath), line)
			}
			if _, statErr := os.Stat(line); statErr == nil && IsSupportedAudioFile(line) {
				paths = append(paths, line)
			}
		}
	}

	pl, err := l.store.CreatePlaylist(name, "Diimpor dari "+filepath.Base(m3uPath), true, "#1DB954")
	if err != nil {
		return db.Playlist{}, 0, err
	}
	tracks, err := l.ImportFiles(paths, false)
	if err != nil {
		return pl, 0, err
	}
	for _, t := range tracks {
		_ = l.store.AddTrackToPlaylist(pl.ID, t.ID)
	}
	detail, err := l.store.GetPlaylistDetail(pl.ID)
	if err != nil {
		return pl, len(tracks), nil
	}
	return detail.Playlist, len(tracks), nil
}
