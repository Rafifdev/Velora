//go:build windows

package main

import "syscall"

var (
	sysKernel32             = syscall.NewLazyDLL("kernel32.dll")
	procSetThreadExecState  = sysKernel32.NewProc("SetThreadExecutionState")
)

const (
	esContinuous     uintptr = 0x80000000
	esSystemRequired uintptr = 0x00000001
)

func setSystemKeepAwake(keepAwake bool) {
	flags := esContinuous
	if keepAwake {
		flags |= esSystemRequired
	}
	_, _, _ = procSetThreadExecState.Call(flags)
}
