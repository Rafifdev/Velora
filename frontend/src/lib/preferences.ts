export type WidgetType = 'toggle' | 'select' | 'segmented' | 'stepper' | 'slider' | 'number-input' | 'text' | 'info' | 'action';

export interface PreferenceOption {
  value: string;
  label: string;
}

export type PreferenceCategoryId = 'general' | 'playback' | 'audio' | 'devices' | 'storage';

export interface PreferenceItemDef {
  id: string;
  category: PreferenceCategoryId;
  group: string;
  title: string;
  description: string;
  widget: WidgetType;
  defaultValue: string;
  options?: PreferenceOption[];
  min?: number;
  max?: number;
  step?: number;
  unit?: string;
  startLabel?: string;
  endLabel?: string;
  actionLabel?: string;
  visibleWhen?: { id: string; equals: string };
}

export const PREFERENCE_CATEGORIES = [
  { id: 'general', title: 'Umum', icon: 'ph-sliders-horizontal' },
  { id: 'playback', title: 'Pemutaran', icon: 'ph-play-circle' },
  { id: 'audio', title: 'Kualitas Audio', icon: 'ph-speaker-high' },
  { id: 'devices', title: 'Perangkat', icon: 'ph-devices' },
  { id: 'storage', title: 'Penyimpanan', icon: 'ph-hard-drive' }
] as const;

export const PREFERENCE_APP_TABS = [
  { id: 'whats-new', title: 'Tentang Velora', icon: 'ph-sparkle' }
] as const;

export const PREFERENCE_SETTINGS: PreferenceItemDef[] = [
  // Umum
  {
    id: 'general.zoomLevel',
    category: 'general',
    group: 'Tampilan',
    title: 'Skala Antarmuka',
    description: 'Sesuaikan ukuran tampilan aplikasi dari 70% hingga 130%.',
    widget: 'stepper',
    defaultValue: '1',
    min: 0.7,
    max: 1.3,
    step: 0.1
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
  {
    id: 'dynamicIsland.style',
    category: 'general',
    group: 'Desktop',
    title: 'Gaya Dynamic Island',
    description: 'Pilih tampilan menempel di tepi atas layar atau melayang bebas berbentuk pil.',
    widget: 'segmented',
    defaultValue: 'notch',
    options: [
      { value: 'notch', label: 'Menempel' },
      { value: 'pill', label: 'Melayang' }
    ],
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
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
    widget: 'segmented',
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
    description: 'Pilih kualitas audio saat memutar musik secara daring. Sangat Tinggi memberikan detail suara paling jernih.',
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
      { value: 'spotify', label: 'Velora Studio' },
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
    defaultValue: 'http://127.0.0.1:8899/jam',
    visibleWhen: { id: 'integrations.jam.enabled', equals: 'true' }
  },
  {
    id: 'integrations.jam.apiUrl',
    category: 'devices',
    group: 'Velora Jam',
    title: 'Alamat API Lokal',
    description: 'Alamat dasar HTTP API untuk akses perangkat di jaringan lokal.',
    widget: 'info',
    defaultValue: 'http://127.0.0.1:8899/api',
    visibleWhen: { id: 'integrations.jam.enabled', equals: 'true' }
  },

  // Penyimpanan
  {
    id: 'storage.clearCache',
    category: 'storage',
    group: 'Cache Aplikasi',
    title: 'Bersihkan Cache Audio dan Sampul',
    description: 'Hapus berkas sementara hasil streaming dan gambar sampul untuk menghemat ruang penyimpanan.',
    widget: 'action',
    defaultValue: '',
    actionLabel: 'Bersihkan Cache'
  }
];

export function getPrefValue(prefs: Record<string, string>, id: string): string {
  if (prefs[id] !== undefined) return prefs[id];
  const def = PREFERENCE_SETTINGS.find((s) => s.id === id);
  return def ? def.defaultValue : '';
}

export function getPrefBool(prefs: Record<string, string>, id: string): boolean {
  return getPrefValue(prefs, id) === 'true';
}
