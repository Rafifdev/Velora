//go:build windows

package discord

import (
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

func openDiscordIPC() (io.ReadWriteCloser, error) {
	for i := 0; i < 10; i++ {
		pipePath := fmt.Sprintf(`\\.\pipe\discord-ipc-%d`, i)
		pathPtr, err := windows.UTF16PtrFromString(pipePath)
		if err != nil {
			continue
		}
		handle, err := windows.CreateFile(
			pathPtr,
			windows.GENERIC_READ|windows.GENERIC_WRITE,
			0,
			nil,
			windows.OPEN_EXISTING,
			windows.FILE_FLAG_OVERLAPPED,
			0,
		)
		if err == windows.ERROR_PIPE_BUSY {
			time.Sleep(50 * time.Millisecond)
			handle, err = windows.CreateFile(
				pathPtr,
				windows.GENERIC_READ|windows.GENERIC_WRITE,
				0,
				nil,
				windows.OPEN_EXISTING,
				windows.FILE_FLAG_OVERLAPPED,
				0,
			)
		}
		if err != nil || handle == windows.InvalidHandle {
			continue
		}
		f := os.NewFile(uintptr(handle), pipePath)
		if f == nil {
			_ = windows.CloseHandle(handle)
			continue
		}
		return f, nil
	}
	return nil, errDiscordUnavailable
}
