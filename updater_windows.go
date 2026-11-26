//go:build windows

package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type UpdateProgress struct {
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	Error    string `json:"error,omitempty"`
}

func (a *App) emitUpdateProgress(status string, progress int, errMsg string) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "update:progress", UpdateProgress{
		Status:   status,
		Progress: progress,
		Error:    errMsg,
	})
}

func validateUpdateDownloadURL(rawURL string) error {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return fmt.Errorf("download URL is empty")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("invalid download URL: %w", err)
	}
	if !strings.EqualFold(parsed.Scheme, "https") {
		return fmt.Errorf("download URL must use https")
	}
	host := strings.ToLower(parsed.Hostname())
	allowed := host == "github.com" ||
		strings.HasSuffix(host, ".github.com") ||
		host == "githubusercontent.com" ||
		strings.HasSuffix(host, ".githubusercontent.com")
	if !allowed {
		return fmt.Errorf("unsupported update host: %s", host)
	}
	return nil
}

func downloadUpdateToFile(downloadURL string, destPath string, onProgress func(int)) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Velora-Updater/"+veloraAppVersion)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("download failed with HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	total := resp.ContentLength
	var downloaded int64
	lastPct := -1
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)
			if total > 0 && onProgress != nil {
				pct := int(downloaded * 100 / total)
				if pct > 100 {
					pct = 100
				}
				if pct != lastPct {
					lastPct = pct
					onProgress(pct)
				}
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	if downloaded < 1024 {
		return fmt.Errorf("downloaded file is unexpectedly small (%d bytes)", downloaded)
	}
	if onProgress != nil && lastPct < 100 {
		onProgress(100)
	}
	return nil
}

func verifyPEExecutable(filePath string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var magic [2]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return fmt.Errorf("invalid executable header: %w", err)
	}
	if magic[0] != 'M' || magic[1] != 'Z' {
		return fmt.Errorf("downloaded file is not a valid Windows executable")
	}
	return nil
}

func isInstallerFilename(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "setup") || strings.Contains(lower, "installer")
}

func escapePSSingleQuoted(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func installAndRestartWindows(a *App, downloadURL string) error {
	if isStorePackage() {
		return fmt.Errorf("Microsoft Store installations must be updated through the Microsoft Store")
	}
	if err := validateUpdateDownloadURL(downloadURL); err != nil {
		return err
	}

	a.emitUpdateProgress("downloading", 0, "")

	parsed, _ := url.Parse(downloadURL)
	baseName := filepath.Base(parsed.Path)
	if baseName == "" || baseName == "." || !strings.HasSuffix(strings.ToLower(baseName), ".exe") {
		baseName = "Velora-update.exe"
	}

	tempDir, err := os.MkdirTemp("", "velora-update-*")
	if err != nil {
		a.emitUpdateProgress("error", 0, err.Error())
		return err
	}
	downloadedPath := filepath.Join(tempDir, baseName)

	if err := downloadUpdateToFile(downloadURL, downloadedPath, func(pct int) {
		a.emitUpdateProgress("downloading", pct, "")
	}); err != nil {
		a.emitUpdateProgress("error", 0, err.Error())
		return err
	}

	if err := verifyPEExecutable(downloadedPath); err != nil {
		a.emitUpdateProgress("error", 0, err.Error())
		return err
	}

	exePath, err := os.Executable()
	if err != nil {
		a.emitUpdateProgress("error", 0, err.Error())
		return err
	}
	exePath, _ = filepath.Abs(exePath)

	a.emitUpdateProgress("installing", 100, "")

	pid := os.Getpid()
	scriptPath := filepath.Join(tempDir, "apply-update.ps1")

	var script string
	if isInstallerFilename(baseName) {
		script = fmt.Sprintf(`$ErrorActionPreference = 'Stop'
$pidToWait = %d
try { Wait-Process -Id $pidToWait -Timeout 30 -ErrorAction SilentlyContinue } catch {}
Start-Sleep -Milliseconds 500
$installer = '%s'
$exe = '%s'
$proc = Start-Process -FilePath $installer -ArgumentList '/S' -Wait -PassThru
if ($proc.ExitCode -eq 0 -and (Test-Path $exe)) {
  Start-Process -FilePath $exe
}
Remove-Item -LiteralPath '%s' -Recurse -Force -ErrorAction SilentlyContinue
`, pid, escapePSSingleQuoted(downloadedPath), escapePSSingleQuoted(exePath), escapePSSingleQuoted(tempDir))
	} else {
		script = fmt.Sprintf(`$ErrorActionPreference = 'Stop'
$pidToWait = %d
try { Wait-Process -Id $pidToWait -Timeout 30 -ErrorAction SilentlyContinue } catch {}
Start-Sleep -Milliseconds 500
$src = '%s'
$dst = '%s'
$copied = $false
for ($i = 0; $i -lt 15; $i++) {
  try {
    Copy-Item -LiteralPath $src -Destination $dst -Force
    $copied = $true
    break
  } catch {
    Start-Sleep -Milliseconds 400
  }
}
if ($copied) {
  Start-Process -FilePath $dst
}
Remove-Item -LiteralPath '%s' -Recurse -Force -ErrorAction SilentlyContinue
`, pid, escapePSSingleQuoted(downloadedPath), escapePSSingleQuoted(exePath), escapePSSingleQuoted(tempDir))
	}

	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		a.emitUpdateProgress("error", 0, err.Error())
		return err
	}

	cmd := exec.Command(
		"powershell",
		"-NoProfile",
		"-NonInteractive",
		"-WindowStyle", "Hidden",
		"-ExecutionPolicy", "Bypass",
		"-File", scriptPath,
	)
	const detachedProcess = 0x00000008
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: detachedProcess,
	}
	if err := cmd.Start(); err != nil {
		a.emitUpdateProgress("error", 0, err.Error())
		return err
	}

	a.emitUpdateProgress("restarting", 100, "")
	go func() {
		time.Sleep(350 * time.Millisecond)
		a.forceQuitApp()
	}()
	_ = strconv.Itoa(pid)
	return nil
}
