//go:build !windows

package main

type AudioOutputDevice struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ConnType string `json:"connType"`
	IsActive bool   `json:"isActive"`
	BatPct   int    `json:"batPct"`
	VolPct   int    `json:"volPct"`
}

func listWindowsAudioOutputs(_ int, _ string) []AudioOutputDevice {
	return nil
}

func setDefaultWindowsAudioOutput(_ string) bool {
	return false
}
