//go:build !windows

package main

import "fmt"

type UpdateProgress struct {
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	Error    string `json:"error,omitempty"`
}

func installAndRestartWindows(_ *App, _ string) error {
	return fmt.Errorf("in-app self-update is only supported on Windows")
}
