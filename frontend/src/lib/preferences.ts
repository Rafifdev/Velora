export type WidgetType = 'toggle' | 'select' | 'slider' | 'number-input' | 'text' | 'info';

export interface SettingOption {
  value: string;
  label: string;
}

export type PrefCategoryId = 'general' | 'playback' | 'audio' | 'devices';

export interface SettingItemDef {
  id: string;
  category: PrefCategoryId;
  group: string;
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

export const NUCLEAR_CATEGORIES = [
  { id: 'general', title: 'Umum', icon: 'ph-sliders-horizontal' },
  { id: 'playback', title: 'Pemutaran', icon: 'ph-play-circle' },
  { id: 'audio', title: 'Kualitas Audio', icon: 'ph-speaker-high' },
  { id: 'devices', title: 'Perangkat', icon: 'ph-devices' }
] as const;

export const NUCLEAR_APP_TABS = [
  { id: 'themes', title: 'Tema Warna', icon: 'ph-palette' },
  { id: 'whats-new', title: 'Tentang Velora', icon: 'ph-sparkle' }
] as const;

export const NUCLEAR_SETTINGS: SettingItemDef[] = [
  // Umum
  {
    id: 'theme.dark',
    category: 'general',
    group: 'Tampilan',
    title: 'Mode Gelap',
    description: 'Gunakan palet warna gelap yang nyaman di mata saat mendengarkan musik.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'general.zoomLevel',
    category: 'general',
    group: 'Tampilan',
    title: 'Skala Antarmuka',
    description: 'Sesuaikan ukuran tampilan aplikasi dari 70% hingga 130%.',
    widget: 'select',
    defaultValue: '1',
    options: [
      { value: '0.7', label: '70%' },
      { value: '0.8', label: '80%' },
      { value: '0.9', label: '90%' },
      { value: '1', label: '100%' },
      { value: '1.1', label: '110%' },
      { value: '1.2', label: '120%' },
      { value: '1.3', label: '130%' }
    ]
  },
  {
    id: 'layout.leftCollapsed',
    category: 'general',
    group: 'Tata Letak',
    title: 'Sembunyikan Menu Kiri',
    description: 'Perkecil bilah navigasi kiri agar area konten utama lebih luas.',
    widget: 'toggle',
    defaultValue: 'false'
  },
  {
    id: 'layout.rightCollapsed',
    category: 'general',
    group: 'Tata Letak',
    title: 'Sembunyikan Panel Kanan',
    description: 'Tutup panel studio kanan yang menampilkan sampul album, lirik, dan antrean.',
    widget: 'toggle',
    defaultValue: 'false'
  },
  {
    id: 'dynamicIsland.enabled',
    category: 'general',
    group: 'Desktop',
    title: 'Tampilkan Dynamic Island',
    description: 'Aktifkan bilah interaktif di bagian atas layar untuk kontrol cepat pemutaran musik.',
    widget: 'toggle',
    defaultValue: 'true'
  },

  // Pemutaran
  {
    id: 'lyrics.autoScroll',
    category: 'playback',
    group: 'Lirik Lagu',
    title: 'Gulir Otomatis Lirik',
    description: 'Menggulir lirik tersinkronisasi secara otomatis mengikuti posisi lagu yang diputar.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'lyrics.textSize',
    category: 'playback',
    group: 'Lirik Lagu',
    title: 'Ukuran Teks Lirik',
    description: 'Mengatur besar kecilnya huruf pada panel lirik.',
    widget: 'select',
    defaultValue: '1',
    options: [
      { value: '0', label: 'Kecil' },
      { value: '1', label: 'Sedang' },
      { value: '2', label: 'Besar' }
    ]
  },

  // Kualitas Audio
  {
    id: 'stream_quality',
    category: 'audio',
    group: 'Streaming',
    title: 'Kualitas Streaming Audio',
    description: 'Pilih tingkat bitrate streaming audio. Sangat Tinggi memprioritaskan Opus di atas 160 kbps.',
    widget: 'select',
    defaultValue: 'high',
    options: [
      { value: 'high', label: 'Sangat Tinggi' },
      { value: 'normal', label: 'Normal' },
      { value: 'low', label: 'Hemat Kuota' }
    ]
  },
  {
    id: 'playback.audioNormalize',
    category: 'audio',
    group: 'Volume',
    title: 'Normalisasi Kenyaringan',
    description: 'Setel tingkat volume yang konsisten antar lagu untuk mencegah lonjakan suara.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'playback.eqPreset',
    category: 'audio',
    group: 'Equalizer',
    title: 'Profil Suara Equalizer',
    description: 'Pilih karakter suara dari mesin audio lima band secara langsung.',
    widget: 'select',
    defaultValue: 'spotify',
    options: [
      { value: 'spotify', label: 'Spotify Studio' },
      { value: 'flat', label: 'Datar' },
      { value: 'bass', label: 'Bass Mendalam' },
      { value: 'vocal', label: 'Vokal Jernih' },
      { value: 'electronic', label: 'Elektronik' }
    ]
  },

  // Perangkat
  {
    id: 'integrations.jam.enabled',
    category: 'devices',
    group: 'Velora Jam',
    title: 'Kontrol Jarak Jauh Wi-Fi',
    description: 'Izinkan kendali pemutaran musik dari ponsel atau perangkat lain di jaringan Wi-Fi yang sama.',
    widget: 'toggle',
    defaultValue: 'false'
  },
  {
    id: 'integrations.jam.remoteUrl',
    category: 'devices',
    group: 'Velora Jam',
    title: 'Alamat Remote',
    description: 'Buka alamat ini di browser perangkat lain untuk mengendalikan pemutaran.',
    widget: 'info',
    defaultValue: 'http://127.0.0.1:8899/jam'
  },
  {
    id: 'integrations.jam.apiUrl',
    category: 'devices',
    group: 'Velora Jam',
    title: 'Alamat API Lokal',
    description: 'Alamat dasar HTTP API untuk akses perangkat di jaringan lokal.',
    widget: 'info',
    defaultValue: 'http://127.0.0.1:8899/api'
  }
];

export const NUCLEAR_THEMES = [
  {
    id: 'monolith-carbon-dark',
    seed: 'monolith',
    name: 'Monolith Carbon Gelap',
    description: 'Permukaan karbon gelap dengan aksen biru es titanium dan oranye Sunset Tangerine.',
    dark: true,
    accent: '#FF9E64',
    previewBg: '#101113',
    previewPanel: '#191B1E'
  },
  {
    id: 'monolith-carbon-light',
    seed: 'monolith',
    name: 'Monolith Carbon Terang',
    description: 'Permukaan terang sejuk dipadukan biru baja dan sorotan Sunset Tangerine.',
    dark: false,
    accent: '#EA6A1F',
    previewBg: '#F6F8FA',
    previewPanel: '#E8ECF2'
  },
  {
    id: 'material-violet-dark',
    seed: 'violet',
    name: 'Material You Violet Gelap',
    description: 'Permukaan gelap tonal Google Material Design 3 dengan aksen lavender lembut.',
    dark: true,
    accent: '#D0BCFF',
    previewBg: '#141218',
    previewPanel: '#211F26'
  },
  {
    id: 'material-violet-light',
    seed: 'violet',
    name: 'Material You Violet Terang',
    description: 'Permukaan terang tonal Material Design 3 dengan tombol ungu yang bersih.',
    dark: false,
    accent: '#6750A4',
    previewBg: '#FEF7FF',
    previewPanel: '#F3EDF7'
  },
  {
    id: 'material-blue-dark',
    seed: 'blue',
    name: 'Material Cobalt Gelap',
    description: 'Ruang kerja gelap batu tulis Material 3 dengan kontainer utama biru langit.',
    dark: true,
    accent: '#A8C7FA',
    previewBg: '#111318',
    previewPanel: '#1D2026'
  },
  {
    id: 'material-emerald-dark',
    seed: 'emerald',
    name: 'Material Emerald Gelap',
    description: 'Palet gelap studio Material 3 bernuansa hijau tonal dinamis.',
    dark: true,
    accent: '#8CD69D',
    previewBg: '#111412',
    previewPanel: '#1D221E'
  },
  {
    id: 'material-rose-dark',
    seed: 'rose',
    name: 'Material Rose Gelap',
    description: 'Tema gelap mawar hangat Material 3 dengan kontainer tonal yang halus.',
    dark: true,
    accent: '#FFB2BC',
    previewBg: '#181214',
    previewPanel: '#251D1F'
  },
  {
    id: 'material-rose-light',
    seed: 'rose',
    name: 'Material Rose Terang',
    description: 'Ruang kerja terang bernuansa merah muda lembut dengan elevasi yang rapi.',
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
