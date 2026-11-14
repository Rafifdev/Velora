export type WidgetType = 'toggle' | 'select' | 'slider' | 'number-input' | 'text' | 'info';

export interface SettingOption {
  value: string;
  label: string;
}

export interface SettingItemDef {
  id: string;
  category: 'appearance' | 'general' | 'history' | 'integrations' | 'playback' | 'plugins' | 'updates';
  title: string;
  description: string;
  widget: WidgetType;
  defaultValue: string;
  options?: SettingOption[];
  min?: number;
  max?: number;
  step?: number;
  unit?: string;
  startLabel?: string;
  endLabel?: string;
}

export const LANGUAGE_OPTIONS: SettingOption[] = [
  { value: 'en_US', label: 'English' },
  { value: 'id_ID', label: 'Bahasa Indonesia' },
  { value: 'de_DE', label: 'Deutsch' },
  { value: 'es_ES', label: 'Español' },
  { value: 'fr_FR', label: 'Français' },
  { value: 'it_IT', label: 'Italiano' },
  { value: 'ja_JP', label: '日本語' },
  { value: 'pl_PL', label: 'Polski' },
  { value: 'pt_BR', label: 'Português (Brasil)' },
  { value: 'ru_RU', label: 'Русский' },
  { value: 'zh_CN', label: '简体中文' }
];

export const NUCLEAR_CATEGORIES = [
  { id: 'appearance', title: 'Appearance', icon: 'ph-paint-brush' },
  { id: 'general', title: 'General', icon: 'ph-sliders-horizontal' },
  { id: 'history', title: 'History', icon: 'ph-clock-counter-clockwise' },
  { id: 'integrations', title: 'Integrations', icon: 'ph-plugs-connected' },
  { id: 'playback', title: 'Playback', icon: 'ph-play-circle' },
  { id: 'plugins', title: 'Plugins', icon: 'ph-puzzle-piece' },
  { id: 'updates', title: 'Updates', icon: 'ph-arrows-clockwise' }
] as const;

export const NUCLEAR_APP_TABS = [
  { id: 'shortcuts', title: 'Key Shortcuts', icon: 'ph-keyboard' },
  { id: 'app-plugins', title: 'Plugins', icon: 'ph-puzzle-piece' },
  { id: 'themes', title: 'Themes', icon: 'ph-palette' },
  { id: 'whats-new', title: "What's New", icon: 'ph-sparkle' }
] as const;

export const NUCLEAR_SETTINGS: SettingItemDef[] = [
  // Appearance
  {
    id: 'appearance.framelessWindow',
    category: 'appearance',
    title: 'Frameless window',
    description: 'Hide the native title bar. Drag the top bar to move the window.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'appearance.customTitleBar',
    category: 'appearance',
    title: 'Custom title bar',
    description: 'Replace the native title bar with a styled one that matches the app.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'appearance.titleBarStyle',
    category: 'appearance',
    title: 'Title bar style',
    description: 'Choose the visual style for the custom title bar controls.',
    widget: 'select',
    defaultValue: 'auto',
    options: [
      { value: 'auto', label: 'Auto' },
      { value: 'macos', label: 'macOS' },
      { value: 'windows', label: 'Windows / Linux' }
    ]
  },
  {
    id: 'theme.dark',
    category: 'appearance',
    title: 'Dark Mode',
    description: 'Switch between Material You Tonal Dark and Material You Tonal Light theme.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'lyrics.textSize',
    category: 'appearance',
    title: 'Lyrics text size',
    description: 'The size of the text in the lyrics view.',
    widget: 'slider',
    defaultValue: '1',
    min: 0,
    max: 2,
    step: 1,
    startLabel: 'Small',
    endLabel: 'Large'
  },
  {
    id: 'lyrics.autoScroll',
    category: 'appearance',
    title: 'Lyrics auto-scroll',
    description: 'Scroll synced lyrics along with the song.',
    widget: 'toggle',
    defaultValue: 'true'
  },

  // General
  {
    id: 'general.language',
    category: 'general',
    title: 'Language',
    description: 'Select your preferred language',
    widget: 'select',
    defaultValue: 'en_US',
    options: LANGUAGE_OPTIONS
  },

  // History
  {
    id: 'history.enabled',
    category: 'history',
    title: 'Record listening history',
    description: 'Keep a local record of the music you play.',
    widget: 'toggle',
    defaultValue: 'true'
  },

  // Integrations
  {
    id: 'integrations.jam.enabled',
    category: 'integrations',
    title: 'Nuclear Jam',
    description:
      "Control player from your phone or other devices on the same Wi-Fi. Anyone on that Wi-Fi can control it too, so don't use it on public Wi-Fi.",
    widget: 'toggle',
    defaultValue: 'false'
  },
  {
    id: 'integrations.jam.remoteUrl',
    category: 'integrations',
    title: 'Remote URL',
    description: 'Open this URL on another device to control playback remotely.',
    widget: 'info',
    defaultValue: 'http://127.0.0.1:8899/jam'
  },
  {
    id: 'integrations.jam.apiUrl',
    category: 'integrations',
    title: 'API URL',
    description: 'HTTP API base URL for programmatic access.',
    widget: 'info',
    defaultValue: 'http://127.0.0.1:8899/api'
  },
  {
    id: 'integrations.mcp.enabled',
    category: 'integrations',
    title: 'Enable MCP Server',
    description: 'Start a local MCP server that allows AI tools to control the player.',
    widget: 'toggle',
    defaultValue: 'false'
  },
  {
    id: 'integrations.mcp.serverUrl',
    category: 'integrations',
    title: 'MCP Server URL',
    description: 'Point your AI tool to this URL to connect.',
    widget: 'info',
    defaultValue: 'http://127.0.0.1:8800/mcp'
  },
  {
    id: 'integrations.mpd.enabled',
    category: 'integrations',
    title: 'Enable MPD Server',
    description: 'Start a local MPD-compatible server so MPD clients can control playback.',
    widget: 'toggle',
    defaultValue: 'false'
  },
  {
    id: 'integrations.mpd.serverUrl',
    category: 'integrations',
    title: 'MPD Server URL',
    description: 'Point your MPD client to this address to connect.',
    widget: 'info',
    defaultValue: 'mpd://127.0.0.1:6600'
  },
  {
    id: 'integrations.discord.enabled',
    category: 'integrations',
    title: 'Discord Rich Presence',
    description: "Show what you're listening to on your Discord profile.",
    widget: 'toggle',
    defaultValue: 'false'
  },

  // Playback
  {
    id: 'stream_quality',
    category: 'playback',
    title: 'Streaming audio quality',
    description: 'Select streaming codec & bitrate tier. High prioritizes Opus 160+ kbps for Spotify-like clarity.',
    widget: 'select',
    defaultValue: 'high',
    options: [
      { value: 'high', label: 'Very High (Opus / Best)' },
      { value: 'normal', label: 'Normal (128 kbps)' },
      { value: 'low', label: 'Data Saver (Low)' }
    ]
  },
  {
    id: 'playback.audioNormalize',
    category: 'playback',
    title: 'Loudness normalization (-14 LUFS)',
    description: 'Balance volume levels across tracks and prevent peak distortion with a mastering limiter.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'playback.eqPreset',
    category: 'playback',
    title: 'Sound profile (Equalizer)',
    description: 'Real-time 5-band mastering EQ applied cleanly through the Web Audio engine.',
    widget: 'select',
    defaultValue: 'spotify',
    options: [
      { value: 'spotify', label: 'Spotify Studio (Default)' },
      { value: 'flat', label: 'Flat (Reference)' },
      { value: 'bass', label: 'Deep Bass' },
      { value: 'vocal', label: 'Vocal Clarity' },
      { value: 'electronic', label: 'Electronic / EDM' }
    ]
  },
  {
    id: 'playback.discoveryVariety',
    category: 'playback',
    title: 'Discovery variety',
    description: 'How far recommendations deviate from your current listening. Lower is closer.',
    widget: 'slider',
    defaultValue: '0.5',
    min: 0,
    max: 1,
    step: 0.05,
    startLabel: 'Less crazy',
    endLabel: 'More crazy'
  },
  {
    id: 'playback.skipSeconds',
    category: 'playback',
    title: 'Skip duration',
    description: 'Number of seconds to skip forward/backward',
    widget: 'number-input',
    defaultValue: '5',
    min: 1,
    max: 60,
    step: 1,
    unit: 's'
  },
  {
    id: 'playback.crossfadeMs',
    category: 'playback',
    title: 'Crossfade',
    description: 'Crossfade duration between tracks in milliseconds',
    widget: 'number-input',
    defaultValue: '0',
    min: 0,
    max: 5000,
    step: 50,
    unit: 'ms'
  },
  {
    id: 'playback.streamExpiryMs',
    category: 'playback',
    title: 'Stream expiry time',
    description: 'How long before a resolved stream URL is considered expired and needs re-resolution',
    widget: 'number-input',
    defaultValue: '3600000',
    min: 300000,
    max: 86400000,
    step: 300000,
    unit: 'ms'
  },
  {
    id: 'playback.streamResolutionRetries',
    category: 'playback',
    title: 'Stream resolution retries',
    description: 'Number of times to retry resolving a stream URL before moving to the next candidate',
    widget: 'number-input',
    defaultValue: '3',
    min: 1,
    max: 10,
    step: 1
  },
  {
    id: 'playback.streamVerification',
    category: 'playback',
    title: 'Stream verification',
    description:
      'Plays the streams that you verified and shows the verification bar. When this setting is off, all verifications are ignored.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'playback.streamVerificationService',
    category: 'playback',
    title: 'Online verification',
    description:
      'Sends your verifications to the verification service and gets the streams that other users verified. This setting has no effect when stream verification is off.',
    widget: 'toggle',
    defaultValue: 'true'
  },

  // Plugins
  {
    id: 'plugins.autoUpdate',
    category: 'plugins',
    title: 'Auto-update plugins',
    description: 'Automatically download and install new versions of store plugins on startup',
    widget: 'toggle',
    defaultValue: 'true'
  },

  // Updates
  {
    id: 'updates.checkForUpdates',
    category: 'updates',
    title: 'Check for updates',
    description: 'Automatically check for new versions on startup',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'updates.autoInstall',
    category: 'updates',
    title: 'Auto-install updates',
    description: 'Download and install updates automatically when available',
    widget: 'toggle',
    defaultValue: 'false'
  }
];

export const KEY_SHORTCUTS_LIST = [
  { id: 'playback.toggle', section: 'Playback', label: 'Play / Pause', defaultKey: 'Space' },
  { id: 'playback.next', section: 'Playback', label: 'Next track', defaultKey: 'Ctrl+Right' },
  { id: 'playback.previous', section: 'Playback', label: 'Previous track', defaultKey: 'Ctrl+Left' },
  { id: 'playback.seekForward', section: 'Playback', label: 'Seek forward', defaultKey: 'Right' },
  { id: 'playback.seekBackward', section: 'Playback', label: 'Seek backward', defaultKey: 'Left' },
  { id: 'playback.volumeUp', section: 'Playback', label: 'Volume up', defaultKey: 'Up' },
  { id: 'playback.volumeDown', section: 'Playback', label: 'Volume down', defaultKey: 'Down' },
  { id: 'playback.mute', section: 'Playback', label: 'Mute / Unmute', defaultKey: 'M' },
  { id: 'general.toggleSettings', section: 'General', label: 'Open settings', defaultKey: 'Ctrl+,' }
];

export const BUILTIN_PLUGINS = [
  {
    id: 'youtube-music',
    name: 'YouTube Music (InnerTube)',
    author: 'Velora Core',
    version: '2.1.0',
    description: 'Stream catalog & playlists directly from YouTube Music with live search.'
  },
  {
    id: 'local-audio',
    name: 'Local Library & Folder Scanner',
    author: 'Velora Core',
    version: '1.4.0',
    description: 'Zero-copy local MP3, FLAC, OPUS, M4A, OGG & WAV playback with artwork extraction.'
  },
  {
    id: 'lrclib-lyrics',
    name: 'LRCLIB Synced Lyrics',
    author: 'Community',
    version: '1.2.0',
    description: 'Real-time LRC & plain lyrics fetcher for the right-side Lyrics Card.'
  },
  {
    id: 'geiger-discovery',
    name: 'Geiger Discovery Engine',
    author: 'Nuclear Standard',
    version: '1.0.5',
    description: 'Powers Top Playlists, Top Artists, and Top Albums dashboard shelves.'
  }
];

export const NUCLEAR_THEMES = [
  {
    id: 'monolith-carbon-dark',
    seed: 'monolith',
    name: 'Monolith Carbon (Dark)',
    description: 'Obsidian carbon surfaces with titanium ice blue and Sunset Tangerine accents.',
    dark: true,
    accent: '#FF9E64',
    previewBg: '#101113',
    previewPanel: '#191B1E'
  },
  {
    id: 'monolith-carbon-light',
    seed: 'monolith',
    name: 'Monolith Carbon (Light)',
    description: 'Cool frost surfaces paired with deep steel cyan and Sunset Tangerine highlights.',
    dark: false,
    accent: '#EA6A1F',
    previewBg: '#F6F8FA',
    previewPanel: '#E8ECF2'
  },
  {
    id: 'material-violet-dark',
    seed: 'violet',
    name: 'Material You Violet (Dark)',
    description: 'Google Material Design 3 tonal dark surfaces with soft lavender accents.',
    dark: true,
    accent: '#D0BCFF',
    previewBg: '#141218',
    previewPanel: '#211F26'
  },
  {
    id: 'material-violet-light',
    seed: 'violet',
    name: 'Material You Violet (Light)',
    description: 'Google Material Design 3 tonal light surfaces with clean purple pills.',
    dark: false,
    accent: '#6750A4',
    previewBg: '#FEF7FF',
    previewPanel: '#F3EDF7'
  },
  {
    id: 'material-blue-dark',
    seed: 'blue',
    name: 'Material Cobalt (Dark)',
    description: 'Crisp Material 3 dark slate workspace with sky-blue primary containers.',
    dark: true,
    accent: '#A8C7FA',
    previewBg: '#111318',
    previewPanel: '#1D2026'
  },
  {
    id: 'material-emerald-dark',
    seed: 'emerald',
    name: 'Material Emerald (Dark)',
    description: 'Studio dark Material 3 palette inspired by dynamic green tonal surfaces.',
    dark: true,
    accent: '#8CD69D',
    previewBg: '#111412',
    previewPanel: '#1D221E'
  },
  {
    id: 'material-rose-dark',
    seed: 'rose',
    name: 'Material Rose (Dark)',
    description: 'Warm Material 3 rose quartz dark theme with smooth tonal containers.',
    dark: true,
    accent: '#FFB2BC',
    previewBg: '#181214',
    previewPanel: '#251D1F'
  },
  {
    id: 'material-rose-light',
    seed: 'rose',
    name: 'Material Rose (Light)',
    description: 'Soft Material 3 blush light workspace with refined elevation and chips.',
    dark: false,
    accent: '#9B4052',
    previewBg: '#FFF8F8',
    previewPanel: '#F9ECEE'
  }
];

export function getPrefValue(prefs: Record<string, string>, id: string): string {
  if (prefs[id] !== undefined) return prefs[id];
  const def = NUCLEAR_SETTINGS.find((s) => s.id === id);
  return def ? def.defaultValue : '';
}

export function getPrefBool(prefs: Record<string, string>, id: string): boolean {
  return getPrefValue(prefs, id) === 'true';
}
