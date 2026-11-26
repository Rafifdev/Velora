//go:build windows

package main

import (
	"context"
	"os"
	"runtime/debug"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	memTh32csSnapProcess = 0x00000002
	memProcQueryLimited  = 0x1000
	memProcSetQuota      = 0x0100
)

var (
	memKernel32               = windows.NewLazySystemDLL("kernel32.dll")
	memUser32                 = windows.NewLazySystemDLL("user32.dll")
	memProcCreateToolhelp32   = memKernel32.NewProc("CreateToolhelp32Snapshot")
	memProcProcess32FirstW    = memKernel32.NewProc("Process32FirstW")
	memProcProcess32NextW     = memKernel32.NewProc("Process32NextW")
	memProcK32EmptyWorkingSet = memKernel32.NewProc("K32EmptyWorkingSet")
	memProcEnumWindows        = memUser32.NewProc("EnumWindows")
	memProcGetWinThreadPID    = memUser32.NewProc("GetWindowThreadProcessId")
	memProcIsWindowVisible    = memUser32.NewProc("IsWindowVisible")
	memProcIsIconic           = memUser32.NewProc("IsIconic")
	memProcGetWindowTextLenW  = memUser32.NewProc("GetWindowTextLengthW")
)

type memProcessEntry32W struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

func collectProcessTreePIDs(rootPID uint32) []uint32 {
	snap, _, _ := memProcCreateToolhelp32.Call(memTh32csSnapProcess, 0)
	if snap == 0 || snap == uintptr(windows.InvalidHandle) {
		return []uint32{rootPID}
	}
	defer func() {
		_ = windows.CloseHandle(windows.Handle(snap))
	}()

	children := make(map[uint32][]uint32)
	var pe memProcessEntry32W
	pe.Size = uint32(unsafe.Sizeof(pe))
	r1, _, _ := memProcProcess32FirstW.Call(snap, uintptr(unsafe.Pointer(&pe)))
	for r1 != 0 {
		children[pe.ParentProcessID] = append(children[pe.ParentProcessID], pe.ProcessID)
		r1, _, _ = memProcProcess32NextW.Call(snap, uintptr(unsafe.Pointer(&pe)))
	}

	seen := map[uint32]bool{rootPID: true}
	queue := []uint32{rootPID}
	out := []uint32{rootPID}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, ch := range children[cur] {
			if !seen[ch] {
				seen[ch] = true
				out = append(out, ch)
				queue = append(queue, ch)
			}
		}
	}
	return out
}

func trimAppMemoryWorkingSet() {
	debug.FreeOSMemory()
	rootPID := uint32(os.Getpid())
	for _, pid := range collectProcessTreePIDs(rootPID) {
		h, err := windows.OpenProcess(memProcQueryLimited|memProcSetQuota, false, pid)
		if err != nil || h == 0 {
			continue
		}
		_, _, _ = memProcK32EmptyWorkingSet.Call(uintptr(h))
		_ = windows.CloseHandle(h)
	}
}

func isMainWindowMinimizedOrHidden(pid uint32) (found bool, minimizedOrHidden bool) {
	type probeState struct {
		targetPID uint32
		found     bool
		visible   bool
		iconic    bool
	}
	st := &probeState{targetPID: pid}
	cb := syscall.NewCallback(func(hwnd uintptr, lParam uintptr) uintptr {
		s := (*probeState)(unsafe.Pointer(lParam))
		var wpid uint32
		_, _, _ = memProcGetWinThreadPID.Call(hwnd, uintptr(unsafe.Pointer(&wpid)))
		if wpid != s.targetPID {
			return 1
		}
		tlen, _, _ := memProcGetWindowTextLenW.Call(hwnd)
		if tlen == 0 {
			return 1
		}
		vis, _, _ := memProcIsWindowVisible.Call(hwnd)
		ico, _, _ := memProcIsIconic.Call(hwnd)
		if vis != 0 {
			s.found = true
			s.visible = true
			if ico != 0 {
				s.iconic = true
			}
			return 0
		}
		return 1
	})
	_, _, _ = memProcEnumWindows.Call(cb, uintptr(unsafe.Pointer(st)))
	if !st.found {
		return false, false
	}
	return true, !st.visible || st.iconic
}

func startMemoryTrimMonitor(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	pid := uint32(os.Getpid())
	go func() {
		ticker := time.NewTicker(2500 * time.Millisecond)
		defer ticker.Stop()

		var hiddenSince time.Time
		var lastTrimAt time.Time
		trimmedCycle := false

		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				found, minOrHid := isMainWindowMinimizedOrHidden(pid)
				if found && minOrHid {
					if hiddenSince.IsZero() {
						hiddenSince = now
					}
					if !trimmedCycle && now.Sub(hiddenSince) >= 5*time.Second && (lastTrimAt.IsZero() || now.Sub(lastTrimAt) >= 30*time.Second) {
						trimmedCycle = true
						lastTrimAt = now
						trimAppMemoryWorkingSet()
					}
				} else {
					hiddenSince = time.Time{}
					trimmedCycle = false
				}
			}
		}
	}()
}
