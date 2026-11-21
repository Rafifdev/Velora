package db

import (
	"path/filepath"
	"testing"
)

func TestStoreCRUD(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	tr, err := store.UpsertTrack(Track{
		Title:       "Test Song",
		ArtistName:  "Test Artist",
		AlbumTitle:  "Test Album",
		DurationSec: 210,
		Format:      "FLAC",
		SourceType:  "local",
		FilePath:    "C:/music/test.flac",
	})
	if err != nil {
		t.Fatalf("upsert track: %v", err)
	}
	if tr.ID == 0 || tr.Format != "FLAC" {
		t.Fatalf("unexpected track: %+v", tr)
	}

	fav, err := store.ToggleFavoriteTrack(tr.ID)
	if err != nil || !fav {
		t.Fatalf("toggle favorite track failed: fav=%v err=%v", fav, err)
	}
	favTracks, err := store.GetTracks("favorites", "")
	if err != nil || len(favTracks) != 1 {
		t.Fatalf("expected 1 favorite track, got %d (err=%v)", len(favTracks), err)
	}

	albums, err := store.GetAlbums(false)
	if err != nil || len(albums) != 1 {
		t.Fatalf("expected 1 album, got %d", len(albums))
	}
	if _, err := store.ToggleFavoriteAlbum(albums[0].ID); err != nil {
		t.Fatalf("toggle album fav: %v", err)
	}
	favAlbums, _ := store.GetAlbums(true)
	if len(favAlbums) != 1 {
		t.Fatalf("expected 1 favorite album, got %d", len(favAlbums))
	}

	artists, err := store.GetArtists(false)
	if err != nil || len(artists) != 1 {
		t.Fatalf("expected 1 artist, got %d", len(artists))
	}
	if _, err := store.ToggleFavoriteArtist(artists[0].ID); err != nil {
		t.Fatalf("toggle artist fav: %v", err)
	}
	favArtists, _ := store.GetArtists(true)
	if len(favArtists) != 1 {
		t.Fatalf("expected 1 favorite artist, got %d", len(favArtists))
	}

	ytAlb, err := store.ToggleFavoriteYTMusicAlbum("MPREb_test123", "Six a Six", "Project Pop", "https://example.com/cover.jpg", 2007)
	if err != nil {
		t.Fatalf("toggle yt album fav failed: %v", err)
	}
	if !ytAlb.IsFavorite || ytAlb.SourceType != "ytmusic" || ytAlb.ExternalID != "MPREb_test123" {
		t.Fatalf("unexpected yt favorite album: %+v", ytAlb)
	}
	favAlbums, _ = store.GetAlbums(true)
	if len(favAlbums) != 2 {
		t.Fatalf("expected 2 favorite albums, got %d", len(favAlbums))
	}
	ytAlb2, err := store.ToggleFavoriteYTMusicAlbum("MPREb_test123", "Six a Six", "Project Pop", "https://example.com/cover.jpg", 2007)
	if err != nil || ytAlb2.IsFavorite {
		t.Fatalf("expected yt album unfavorited, got %+v (err=%v)", ytAlb2, err)
	}

	tr2, err := store.UpsertTrack(Track{
		Title:       "Second Song",
		ArtistName:  "Test Artist",
		AlbumTitle:  "Test Album",
		DurationSec: 190,
		Format:      "MP3",
		SourceType:  "local",
		FilePath:    "C:/music/test2.mp3",
	})
	if err != nil {
		t.Fatalf("upsert track 2: %v", err)
	}

	pl, err := store.CreatePlaylist("My Local Mix", "Roadtrip", true, "#1DB954")
	if err != nil || !pl.IsLocal {
		t.Fatalf("create local playlist failed: %+v err=%v", pl, err)
	}
	if err := store.AddTrackToPlaylist(pl.ID, tr.ID); err != nil {
		t.Fatalf("add track 1: %v", err)
	}
	if err := store.AddTrackToPlaylist(pl.ID, tr2.ID); err != nil {
		t.Fatalf("add track 2: %v", err)
	}
	if err := store.ReorderPlaylistTracks(pl.ID, []int64{tr2.ID, tr.ID}); err != nil {
		t.Fatalf("reorder playlist: %v", err)
	}
	detail, err := store.GetPlaylistDetail(pl.ID)
	if err != nil || len(detail.Tracks) != 2 || detail.Tracks[0].ID != tr2.ID {
		t.Fatalf("unexpected playlist order: %+v err=%v", detail.Tracks, err)
	}

	renamed, err := store.RenamePlaylist(pl.ID, "Renamed Local Mix")
	if err != nil || renamed.Name != "Renamed Local Mix" {
		t.Fatalf("rename playlist failed: %+v err=%v", renamed, err)
	}
}
