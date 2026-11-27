//go:build !windows

package discord

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"
)

func openDiscordIPC() (io.ReadWriteCloser, error) {
	dirs := []string{
		os.Getenv("XDG_RUNTIME_DIR"),
		os.Getenv("TMPDIR"),
		os.Getenv("TMP"),
		os.Getenv("TEMP"),
		"/tmp",
	}
	subDirs := []string{
		"",
		"app/com.discordapp.Discord",
		"snap.discord",
	}

	seen := make(map[string]struct{})
	for _, base := range dirs {
		if base == "" {
			continue
		}
		for _, sub := range subDirs {
			dir := base
			if sub != "" {
				dir = filepath.Join(base, sub)
			}
			for i := 0; i < 10; i++ {
				sockPath := filepath.Join(dir, fmt.Sprintf("discord-ipc-%d", i))
				if _, ok := seen[sockPath]; ok {
					continue
				}
				seen[sockPath] = struct{}{}
				conn, err := net.DialTimeout("unix", sockPath, 1500*time.Millisecond)
				if err == nil {
					return conn, nil
				}
			}
		}
	}
	return nil, errDiscordUnavailable
}
