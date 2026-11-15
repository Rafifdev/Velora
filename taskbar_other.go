//go:build !windows

package main

import "context"

func (a *App) initTaskbar(ctx context.Context) {}

func (a *App) shutdownTaskbar() {}

func (a *App) UpdateTaskbarState(title string, artist string, coverURL string, isPlaying bool) {}
