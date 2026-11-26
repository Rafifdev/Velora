//go:build !windows

package main

import (
	"context"
	"runtime/debug"
)

func trimAppMemoryWorkingSet() {
	debug.FreeOSMemory()
}

func startMemoryTrimMonitor(_ context.Context) {}
