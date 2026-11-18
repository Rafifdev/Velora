//go:build !windows

package main

import (
	"context"
	"velora/internal/sources"
)

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
	StreakDays    int                      `json:"streakDays"`
	LyricsLoading bool                     `json:"lyricsLoading"`
	QueueTitle    string                   `json:"queueTitle"`
	QueueIndex    int                      `json:"queueIndex"`
	Queue         []DynamicIslandQueueItem `json:"queue"`
}

func (a *App) initDynamicIsland(_ context.Context)                       {}
func (a *App) shutdownDynamicIsland()                                    {}
func (a *App) updateDynamicIslandState(_, _, _ string, _ bool)           {}
func (a *App) UpdateDynamicIslandProgress(_ float64, _ float64)          {}
func (a *App) updateDynamicIslandLyrics(_ []sources.LyricLine, _ string) {}
func (a *App) SetDynamicIslandEnabled(_ bool)                            {}
func (a *App) SyncDynamicIsland(_ DynamicIslandSyncPayload)              {}
func (a *App) NotifyDynamicIslandAudioDevice(_ string)                   {}
