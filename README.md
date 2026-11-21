<div align="center">
  <img src="frontend/public/velora-logo-light.svg" alt="Velora Logo" width="96" height="96" />

  <h1>Velora</h1>
  <p><strong>Modern Desktop Music Player for Windows with Native Dynamic Island, Synchronized Lyrics, and Hybrid Streaming</strong></p>

  <p>
    <a href="https://github.com/Rafifdev/Velora/releases"><img src="https://img.shields.io/github/v/release/Rafifdev/Velora?include_prereleases&label=Release&color=FF9E64&style=flat-square" alt="Latest Release" /></a>
    <img src="https://img.shields.io/badge/Platform-Windows%2010%20%2F%2011%20(x64)-0078D6?style=flat-square&logo=windows" alt="Platform Windows" />
    <img src="https://img.shields.io/badge/Go-1.24-00ADD8?style=flat-square&logo=go" alt="Go 1.24" />
    <img src="https://img.shields.io/badge/Svelte-5-FF3E00?style=flat-square&logo=svelte" alt="Svelte 5" />
    <img src="https://img.shields.io/badge/Wails-v2-DF0000?style=flat-square" alt="Wails v2" />
  </p>
</div>

---

## Overview

**Velora** is a native desktop music player for Windows built with **Go (Wails v2)** and **Svelte 5**. It combines online catalog streaming, local library playback, multi-source time-synced lyrics, and a native Win32/Direct2D **Dynamic Island** overlay at the top edge of your screen - designed around an adaptive **Material Design 3** interface.

---

## Key Features

### 1. Hybrid Catalog & Local Library Playback
- **Online Streaming & Discovery**: Browse and stream tracks, albums, artists, top charts, and curated playlists powered by YouTube Music (InnerTube) and Deezer metadata enrichment.
- **Content Region & Filtering**: Configure content region (up to 3 countries), default search filter (`Song`, `Album`, `Artist`, `Playlist`, `Video`), and explicit content filtering.
- **Local Audio Library**: Play and index local audio files (`MP3`, `FLAC`, `WAV`, `M4A`, `AAC`, `OGG`, `OPUS`) with metadata tag extraction and embedded artwork support.
- **Offline Downloads**: Download tracks for offline listening with real-time progress tracking.
- **Playlist Management**: Create, reorder, and customize playlists, import/export `.M3U` files, or import public **Spotify playlists** directly via URL.

### 2. Native Windows Dynamic Island
- **Zero-Overhead Win32 Layered Window**: Rendered directly via Win32 GDI+ and **Direct2D / DirectWrite ClearType** for crisp typography and smooth 60+ FPS liquid transitions.
- **Interactive Multi-Page Overlay**:
  - **Now Playing**: Artwork, scrolling track title/artist, quick favorite toggle, seekbar, and transport controls.
  - **Live Lyrics**: Time-synced scrolling lyrics right inside the top-edge overlay.
  - **Queue Manager**: Inspect and jump between upcoming tracks in your queue.
  - **Audio Output Switcher**: View active audio hardware, Bluetooth battery status, and switch Windows audio output endpoints on the fly.
- **Full Customization**: Choose between *Dynamic Island*, *Single-Line*, or *Hidden* display modes, adjust width/height/opacity, toggle individual content elements, and pick progress bar styles (*Thin*, *Thick*, *Dots*, or *Animated Wave*).

### 3. Audio Engine & Mastering DSP
- **5-Band Mastering Equalizer**: Built-in presets (*Flat*, *Studio*, *Bass Boost*, *Vocal*, *Treble*, *Warm*, *Rock*, *Pop*, *Electronic*) plus custom dB adjustments.
- **Loudness Normalization & Headroom Protection**: Transparent `-14 LUFS` headroom leveling, sub-sonic DC filter, and true-peak safety limiting to prevent digital clipping.
- **Seamless Transitions**: Configurable crossfade (`0–12s`), playback speed (`0.5x–2.0x`), pitch preservation, and silence trimming.

### 4. Multi-Source Synchronized Lyrics
- **5-Tier Lyrics Resolver**: Automatically queries **LRCLIB** (exact & fuzzy duration match), **NetEase Cloud Music**, **KuGou Music**, **YouTube Music** official lyrics tab, and **Lyrics.ovh** fallback.
- **Immersive Presentation**: Dedicated full-screen and side-panel lyrics views with optional karaoke word sweep, inactive line blur, custom font sizing, and cover-matched ambient backdrops.

### 5. Velora Jam & Analytics
- **Velora Jam (Wi-Fi Remote)**: Optional local Wi-Fi remote server with instant QR code pairing to control playback and queue from any mobile browser on the same network.
- **Listening Analytics & Fire Streak**: Track daily listening goals, active listening streaks, top tracks/artists, and listening history with configurable auto-prune intervals.

---

## Tech Stack

| Layer | Technology |
| :--- | :--- |
| **Desktop Framework** | [Wails v2](https://wails.io/) (Go + WebView2) |
| **Backend** | Go 1.24, SQLite (`modernc.org/sqlite`, WAL mode), Native Win32 / Direct2D / DirectWrite / Windows Core Audio COM |
| **Frontend** | Svelte 5 (Runes), TypeScript, Tailwind CSS v4, Material Design 3 Tokens |
| **Audio Pipeline** | Web Audio API Mastering Graph + Local HTTP Range Stream Proxy & MPV IPC Bridge |
| **Installer** | Custom zero-dependency native Win32 Setup & Uninstall Wizard (`cmd/installer`) |

---

## Getting Started

### Prerequisites
- **Windows 10 / 11 (64-bit)** with Microsoft Edge WebView2 Runtime
- **Go 1.24+**
- **Node.js 20+** & **npm**
- **Wails CLI v2.11+** (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

### Development

```powershell
# 1. Install frontend dependencies
npm --prefix frontend install

# 2. Run type and Svelte diagnostics
npm --prefix frontend run check

# 3. Start Wails in live development mode
wails dev
```

### Building Production Binary & Installer

```powershell
# Build standalone application binary (build/bin/Velora.exe)
wails build -clean

# Build the native Win32 Setup Wizard (build/bin/Velora-Setup.exe)
powershell -ExecutionPolicy Bypass -File scripts/build_installer.ps1
```

---

## Project Structure

```text
velora/
├── app.go                              # Wails application lifecycle & frontend bindings
├── dynamic_island_windows.go           # Native Win32 GDI+ Dynamic Island overlay
├── dynamic_island_dwrite_windows.go    # Direct2D & DirectWrite ClearType text renderer
├── audio_devices_windows.go            # Windows Core Audio endpoint enumeration & switching
├── taskbar_windows.go                  # Windows taskbar thumbnail toolbar controls
├── cmd/installer/                      # Native Win32 Setup & Uninstall Wizard
├── internal/
│   ├── audio/                          # Audio stream proxy, transcode & MPV IPC controller
│   ├── db/                             # SQLite schema, migrations, and repository methods
│   ├── jam/                            # Velora Jam Wi-Fi remote HTTP/SSE server & mobile UI
│   ├── sources/                        # YouTube Music, Deezer, Spotify, LRCLIB & local scanner
│   ├── stats/                          # Listening stats & Fire Streak calculation service
│   └── tray/                           # Windows System Tray controller
└── frontend/
    └── src/
        ├── components/                 # Reusable M3 UI components (EmptyState, ModalDialog, EqIndicator, etc.)
        ├── views/                      # Main application views (Home, Library, Playlists, Settings)
        └── lib/                        # API bridge, Web Audio DSP engine, i18n, and preference definitions
```

---

## Keyboard Shortcuts

| Shortcut | Action |
| :--- | :--- |
| `Space` | Play / Pause current track |
| `←` / `→` | Seek backward / forward 5 seconds |
| `↑` / `↓` | Increase / Decrease volume by 5% |
| `Ctrl` + `+` / `-` | Zoom interface in / out (`70%` – `130%`) |
| `Ctrl` + `0` | Reset interface zoom to `100%` |
| `Tab` | Focus global search bar |

---

## Author

Developed by **Yoapipp** ([@Rafifdev](https://github.com/Rafifdev)) · [Instagram](https://www.instagram.com/yoapipp/)
