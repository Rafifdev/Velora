//go:build windows

package main

import (
	"strings"
	"syscall"
	"unsafe"
)

var procGetUserDefaultLocaleName = syscall.NewLazyDLL("kernel32.dll").NewProc("GetUserDefaultLocaleName")

var supportedAppLanguages = map[string]bool{
	"id": true,
	"en": true,
	"ja": true,
	"ko": true,
	"hi": true,
	"pt": true,
	"de": true,
	"fr": true,
	"es": true,
	"ms": true,
}

func detectSystemLanguage() string {
	var buf [85]uint16 // LOCALE_NAME_MAX_LENGTH = 85
	r, _, _ := procGetUserDefaultLocaleName.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if r == 0 {
		return "en"
	}
	loc := syscall.UTF16ToString(buf[:])
	loc = strings.ToLower(strings.TrimSpace(loc))
	if idx := strings.IndexAny(loc, "-_"); idx > 0 {
		loc = loc[:idx]
	}
	if supportedAppLanguages[loc] {
		return loc
	}
	return "en"
}
