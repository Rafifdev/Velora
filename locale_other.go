//go:build !windows

package main

func detectSystemLanguage() string {
	return "en"
}
