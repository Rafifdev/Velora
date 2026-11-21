import { normalizeLang, tr, type LangCode, SUPPORTED_LANGUAGES } from './i18n';

export type WidgetType = 'toggle' | 'select' | 'multi-select' | 'combobox' | 'segmented' | 'stepper' | 'slider' | 'number-input' | 'text' | 'info' | 'action';

export interface PreferenceOption {
  value: string;
  label: string;
}

export type PreferenceCategoryId =
  | 'appearance'
  | 'player'
  | 'lyrics'
  | 'content'
  | 'storage'
  | 'privacy'
  | 'backup'
  | 'system'
  | 'dynamic-island'
  | 'about';

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
  snapPoints?: number[];
  actionLabel?: string;
  visibleWhen?: { id: string; equals: string };
  disabledWhen?: { id: string; equals: string };
}

export interface LocalizedPair {
  en: [string, string];
  id: [string, string];
}

const ITEM_TEXT: Record<string, LocalizedPair> = {
  // 1. Appearance
  'general.zoomLevel': {
    en: ['Display scale', 'Adjust interface zoom level between 70% and 130%.'],
    id: ['Skala tampilan', 'Atur tingkat pembesaran antarmuka dari 70% hingga 130%.']
  },
  'appearance.performance': {
    en: ['Animation effects', 'Control interface transitions and motion intensity.'],
    id: ['Efek animasi', 'Atur transisi dan intensitas animasi antarmuka.']
  },
  'appearance.shade': {
    en: ['Shadow effects', 'Adjust shadow depth on cards and floating panels.'],
    id: ['Efek bayangan', 'Atur intensitas bayangan pada kartu dan panel.']
  },
  'appearance.showStreak': {
    en: ['Streak on Home', 'Show listening streak badge on the Home screen.'],
    id: ['Streak di Beranda', 'Tampilkan lencana streak mendengarkan di Beranda.']
  },
  'layout.hideLeft': {
    en: ['Hide left sidebar', 'Keep the left navigation bar hidden until re-enabled here.'],
    id: ['Sembunyikan bilah kiri', 'Sembunyikan bilah navigasi kiri secara permanen hingga diaktifkan kembali.']
  },
  'layout.hideRight': {
    en: ['Hide right panel', 'Keep the right studio panel hidden until re-enabled here.'],
    id: ['Sembunyikan panel kanan', 'Sembunyikan panel kanan secara permanen hingga diaktifkan kembali.']
  },
  'appearance.meltCards': {
    en: ['Adaptive card tint', 'Blend card surfaces with colors sampled from album artwork.'],
    id: ['Warna kartu adaptif', 'Padukan latar kartu dengan warna dominan dari sampul album.']
  },
  'appearance.listDensity': {
    en: ['List density', 'Choose comfortable or compact row spacing in track lists.'],
    id: ['Kepadatan daftar', 'Pilih jarak baris nyaman atau rapat pada daftar lagu.']
  },
  'appearance.cornerStyle': {
    en: ['Card and cover corners', 'Adjust corner roundness for cards and album covers.'],
    id: ['Sudut kartu dan sampul', 'Atur kelengkungan sudut pada kartu dan sampul album.']
  },
  'appearance.startPage': {
    en: ['Start page', 'Applied when Velora opens.'],
    id: ['Mulai di halaman', 'Berlaku saat Velora dibuka.']
  },
  'appearance.showGreeting': {
    en: ['Show greeting', 'Display a time-based greeting at the top of Home.'],
    id: ['Tampilkan sapaan', 'Tampilkan sapaan sesuai waktu di bagian atas Beranda.']
  },
  'appearance.showRecentHome': {
    en: ['Show Recently Played', 'Display recently played tracks on the Home screen.'],
    id: ['Tampilkan Terakhir diputar', 'Tampilkan lagu yang baru diputar di halaman Beranda.']
  },

  // 2. Player
  'playback.autoplay': {
    en: ['Autoplay', 'Automatically continue with similar tracks when the queue ends.'],
    id: ['Putar otomatis', 'Lanjutkan pemutaran dengan lagu serupa saat antrean berakhir.']
  },
  'playback.audioNormalize': {
    en: ['Normalize volume', 'Maintain a consistent loudness level across different tracks.'],
    id: ['Normalisasi volume', 'Jaga tingkat kenyaringan suara tetap seimbang antar lagu.']
  },
  'playback.waveBar': {
    en: ['Wavy progress bar', 'Display an animated wave along the playback progress bar.'],
    id: ['Bilah progres bergelombang', 'Tampilkan garis bergelombang bergerak pada bilah progres pemutaran.']
  },
  'stream_quality': {
    en: ['Audio streaming quality', 'Select the preferred audio bitrate for online playback.'],
    id: ['Kualitas streaming audio', 'Pilih tingkat kualitas bitrate audio saat memutar musik daring.']
  },
  'playback.eqPreset': {
    en: ['Equalizer preset', 'Select a real-time 5-band equalizer sound profile.'],
    id: ['Preset Equalizer', 'Pilih karakter suara equalizer lima pita secara langsung.']
  },
  'playback.crossfadeSec': {
    en: ['Crossfade', 'Smoothly blend the end of one track into the start of the next.'],
    id: ['Crossfade antar lagu', 'Padukan akhir lagu secara halus ke awal lagu berikutnya.']
  },
  'playback.skipSilence': {
    en: ['Skip silence', 'Automatically bypass silent sections at track boundaries.'],
    id: ['Lewati bagian hening', 'Lewati bagian senyap di penghujung lagu secara otomatis.']
  },
  'playback.radioOnPick': {
    en: ['Queue similar tracks', 'Automatically add similar songs to the queue when playing a single track.'],
    id: ['Tambahkan lagu serupa', 'Buat antrean lagu serupa secara otomatis saat memutar satu lagu.']
  },
  'playback.radioSize': {
    en: ['Similar tracks count', 'Set how many similar songs are added automatically to the queue.'],
    id: ['Jumlah lagu serupa', 'Tentukan jumlah lagu serupa yang ditambahkan otomatis ke antrean.']
  },
  'playback.resumeQueue': {
    en: ['Restore queue on launch', 'Restore the previous queue and playback position in a paused state.'],
    id: ['Pulihkan antrean terakhir', 'Kembalikan antrean dan posisi lagu terakhir dalam kondisi jeda saat dibuka.']
  },
  'playback.shuffleOnStart': {
    en: ['Shuffle on launch', 'Enable shuffle mode automatically when Velora starts.'],
    id: ['Acak saat dibuka', 'Aktifkan mode acak secara otomatis setiap kali Velora dibuka.']
  },
  'playback.startVolume': {
    en: ['Startup volume', 'Set the initial playback volume when Velora launches.'],
    id: ['Volume awal', 'Atur tingkat volume awal saat Velora dibuka.']
  },
  'playback.prevRestartSec': {
    en: ['Previous track behavior', 'Restart the current song if playback has passed the selected duration.'],
    id: ['Perilaku tombol Sebelumnya', 'Ulangi lagu dari awal jika pemutaran telah melewati batas detik tertentu.']
  },

  // 3. Lyrics
  'lyrics.textSize': {
    en: ['Font size', 'Adjust text size in the lyrics panel.'],
    id: ['Ukuran huruf', 'Mengatur besar kecilnya huruf pada panel lirik.']
  },
  'lyrics.autoScroll': {
    en: ['Auto-scroll lyrics', 'Automatically follow the active synced lyric line.'],
    id: ['Gulir otomatis lirik', 'Menggulir lirik tersinkronisasi secara otomatis mengikuti posisi lagu.']
  },
  'lyrics.sweep': {
    en: ['Word-by-word sweep', 'Highlight lyrics smoothly word by word when timed data is available.'],
    id: ['Sapuan per kata', 'Sorot lirik per kata secara halus jika tersedia data waktu kata.']
  },
  'lyrics.blurOthers': {
    en: ['Blur inactive lines', 'Applies in Full performance mode only.'],
    id: ['Kaburkan baris lain', 'Hanya berlaku di Kinerja Penuh.']
  },
  'lyrics.miniLyrics': {
    en: ['Small lyrics in mini mode', 'Show compact lyrics in the mini player view.'],
    id: ['Lirik kecil di mode mini', 'Tampilkan lirik ringkas pada tampilan pemutar mini.']
  },
  'lyrics.nudgeMs': {
    en: ['Lyric timing nudge step', 'Step size for faster/slower lyric sync buttons.'],
    id: ['Langkah geser waktu lirik', 'Untuk tombol Lirik lebih cepat dan lebih lambat.']
  },
  'lyrics.plainFallback': {
    en: ['Show plain lyrics', 'Show unsynced lyrics when synced lyrics are unavailable.'],
    id: ['Tampilkan lirik biasa', 'Lirik tanpa waktu, jika lirik berjalan tidak ada.']
  },

  // 4. Content
  'content.country': {
    en: ['Content region', 'Catalog regions for Home and Search recommendations.'],
    id: ['Negara konten', 'Wilayah katalog untuk rekomendasi Beranda dan Search.']
  },
  'content.language': {
    en: ['App & Dynamic Island language', 'Applies across the entire Velora interface and Windows Dynamic Island. Defaults to your Windows locale (or English).'],
    id: ['Bahasa aplikasi & Dynamic Island', 'Berlaku di seluruh antarmuka Velora dan Dynamic Island Windows. Default mengikuti lokasi sistem Windows (atau English).']
  },
  'content.searchStart': {
    en: ['Default search filter', 'Initial tab selected when opening search results.'],
    id: ['Filter pencarian awal', 'Tab awal yang dipilih saat menampilkan hasil pencarian.']
  },
  'content.hideExplicit': {
    en: ['Hide explicit songs', 'Filter explicit tracks from search results and auto-queued recommendations.'],
    id: ['Sembunyikan lagu eksplisit', 'Sembunyikan lagu eksplisit dari hasil pencarian dan antrean otomatis.']
  },

  // 5. Storage
  'storage.clearCache': {
    en: ['Clear audio and artwork cache', 'Delete temporary streamed files and cover art to free disk space.'],
    id: ['Bersihkan cache lagu dan sampul', 'Hapus berkas sementara hasil streaming dan gambar sampul untuk menghemat ruang penyimpanan.']
  },
  'storage.cacheLimit': {
    en: ['Cache limit', 'Maximum number of recent songs kept in cache (up to ~1 GB).'],
    id: ['Batas cache', 'Jumlah lagu terakhir yang disimpan (paling banyak sekitar 1 GB).']
  },
  'storage.autoDownloadOnLike': {
    en: ['Download liked songs', 'Automatically save liked songs to your downloads folder for offline playback.'],
    id: ['Unduh lagu yang disukai', 'Lagu yang kamu sukai ikut disimpan di folder unduhan agar bisa diputar tanpa internet.']
  },
  'storage.clearCacheOnExit': {
    en: ['Clear song cache on exit', 'Downloaded songs are kept.'],
    id: ['Bersihkan cache lagu saat keluar', 'Unduhan tidak ikut dihapus.']
  },

  // 6. Privacy
  'history.enabled': {
    en: ['Save listening history', 'Used for Recently Played and Stats. Stored locally on this PC only.'],
    id: ['Simpan riwayat dengar', 'Dipakai untuk Terakhir diputar dan Statistik. Hanya tersimpan di komputer ini.']
  },
  'privacy.saveSearches': {
    en: ['Save search history', 'Record recent search queries locally for quick access.'],
    id: ['Simpan riwayat pencarian', 'Simpan kata kunci pencarian terbaru secara lokal agar mudah diakses lagi.']
  },
  'privacy.historyDays': {
    en: ['Auto-delete listening history', 'Older records are removed automatically. Streak days remain intact.'],
    id: ['Hapus riwayat dengar otomatis', 'Catatan yang lebih lama dihapus. Hari-hari streak tetap ada.']
  },

  // 7. Velora Jam
  'integrations.jam.enabled': {
    en: ['Wi-Fi remote control', 'Allow controlling playback from a phone or another device on the same Wi-Fi network.'],
    id: ['Aktifkan kontrol Wi-Fi', 'Izinkan kendali pemutaran musik dari ponsel atau perangkat lain di jaringan Wi-Fi yang sama.']
  },
  'integrations.jam.remoteUrl': {
    en: ['Remote address', 'Open this address in another device browser to control playback.'],
    id: ['Alamat remote', 'Buka alamat ini di browser perangkat lain untuk mengendalikan pemutaran.']
  },

  // 8. System
  'system.mediaKeys': {
    en: ['Media keys', 'Play, pause, next, and previous keys on keyboard or headset work from any window.'],
    id: ['Tombol media', 'Putar, jeda, berikutnya, dan sebelumnya di keyboard atau headset bekerja dari jendela mana pun.']
  },
  'system.closeToTray': {
    en: ['Hide to System Tray on close', 'Keep music playing in the background when the window is closed.'],
    id: ['Sembunyikan ke System Tray saat ditutup', 'Tetap putar musik di latar belakang saat jendela ditutup.']
  },
  'system.keepAwake': {
    en: ['Keep PC awake while playing', 'Display may still turn off.'],
    id: ['Jangan biarkan PC tidur saat musik diputar', 'Layar tetap boleh mati.']
  },
  'system.hideTitle': {
    en: ['Hide song title in window bar', 'Window title stays Velora while playing.'],
    id: ['Sembunyikan judul lagu di jendela', 'Judul jendela tetap Velora saat lagu diputar.']
  },

  // 9. Dynamic Island
  'dynamicIsland.enabled': {
    en: ['Dynamic Island', 'Compact player at the top edge of your screen to view songs and lyrics without opening Velora.'],
    id: ['Dynamic Island', 'Pemutar kecil di tepi atas layar, untuk melihat lagu dan lirik tanpa membuka Velora.']
  },
  'dynamicIsland.style': {
    en: ['Position', 'Choose docked at screen edge/notch or floating pill.'],
    id: ['Posisi', 'Pilih tampilan menempel di tepi atas layar atau melayang berbentuk pil.']
  },
  'dynamicIsland.compactStyle': {
    en: ['Dynamic Island style', 'Closed Dynamic Island appearance.'],
    id: ['Gaya dynamic island', 'Tampilan Dynamic Island saat tertutup.']
  },
  'dynamicIsland.size': {
    en: ['Size', 'Overall Dynamic Island scale preset.'],
    id: ['Ukuran', 'Preset ukuran Dynamic Island.']
  },
  'dynamicIsland.opacity': {
    en: ['Opacity', 'Below 100%, the screen behind becomes slightly visible.'],
    id: ['Kepekatan', 'Di bawah 100% layar di belakang terlihat sedikit.']
  },
  'dynamicIsland.showStatus': {
    en: ['Clock & system status', 'Display clock and Wi-Fi, GPS, Bluetooth icons at the top.'],
    id: ['Jam & status sistem', 'Tampilkan jam serta ikon Wi-Fi, GPS, dan Bluetooth di baris atas.']
  },
  'dynamicIsland.showCover': {
    en: ['Show cover art', 'Display album artwork inside Dynamic Island.'],
    id: ['Sampul album', 'Tampilkan sampul album di Dynamic Island.']
  },
  'dynamicIsland.showClosedCover': {
    en: ['Show cover when closed', 'Display small album art on the left side when Dynamic Island is closed.'],
    id: ['Sampul saat tertutup', 'Tampilkan gambar album kecil di sisi kiri saat Dynamic Island tertutup.']
  },
  'dynamicIsland.showArtist': {
    en: ['Show artist name', 'Display artist subtitle below track title.'],
    id: ['Nama artis', 'Tampilkan nama artis di bawah judul lagu.']
  },
  'dynamicIsland.showLyric': {
    en: ['Lyric', 'Lyric button and active lyric line in one-line pill.'],
    id: ['Lirik', 'Tombol lirik dan baris lirik aktif pada pil satu baris.']
  },
  'dynamicIsland.showHeart': {
    en: ['Show heart button', 'Quick favorite toggle inside Dynamic Island.'],
    id: ['Tombol hati', 'Tombol suka cepat di dalam Dynamic Island.']
  },
  'dynamicIsland.showDots': {
    en: ['Show page dots', 'Navigation dots for Player, Queue, and Devices.'],
    id: ['Titik halaman', 'Titik navigasi halaman Player, Antrean, dan Perangkat.']
  },
  'dynamicIsland.showProgress': {
    en: ['Show progress bar', 'Track playback progress indicator.'],
    id: ['Bilah progres', 'Indikator progres pemutaran lagu.']
  },
  'dynamicIsland.progressStyle': {
    en: ['Progress bar style', 'Choose thin, thick, dotted, or wave progress bar.'],
    id: ['Gaya progres', 'Pilih gaya bilah progres tipis, tebal, titik, atau gelombang.']
  },
  'dynamicIsland.animSpeed': {
    en: ['Animation speed', 'Adjust transition smoothness when Dynamic Island resizes.'],
    id: ['Kecepatan animasi', 'Atur kehalusan animasi saat ukuran Dynamic Island berubah.']
  },
  'dynamicIsland.openTrigger': {
    en: ['Open trigger', 'How Dynamic Island expands from compact mode.'],
    id: ['Membuka', 'Cara memperluas Dynamic Island dari mode ringkas.']
  },
  'dynamicIsland.pauseHide': {
    en: ['Hide when paused', 'Automatically collapse/hide after pausing playback.'],
    id: ['Sembunyikan saat dijeda', 'Sembunyikan otomatis setelah musik dijeda.']
  },
  'dynamicIsland.onlyPlaying': {
    en: ['Show only while playing', 'Dynamic Island hides when music is paused until playback resumes.'],
    id: ['Tampil hanya saat memutar', 'Saat musik dijeda, Dynamic Island hilang sampai lagu diputar lagi.']
  },
  'about.shortcuts': {
    en: ['Keyboard shortcuts', 'View all available keyboard shortcuts in Velora.'],
    id: ['Pintasan keyboard', 'Lihat semua pintasan keyboard yang tersedia di Velora.']
  },
  'about.dataFolder': {
    en: ['Data folder', 'Open the local folder where Velora stores database, cache, and downloads.'],
    id: ['Folder data', 'Buka folder lokal tempat Velora menyimpan database, cache, dan unduhan.']
  },
  'dynamicIsland.screen': {
    en: ['Display', 'Choose which monitor hosts Dynamic Island.'],
    id: ['Layar aktif', 'Pilih monitor tempat Dynamic Island tampil.']
  },
  'dynamicIsland.hideApp': {
    en: ['Hide when Velora window is active', 'Hide Dynamic Island while the main Velora window is focused.'],
    id: ['Sembunyikan saat jendela Velora aktif', 'Sembunyikan Dynamic Island saat jendela utama Velora sedang aktif.']
  },
  'dynamicIsland.hideFull': {
    en: ['Hide in fullscreen apps', 'Automatically hide Dynamic Island during fullscreen games or videos.'],
    id: ['Sembunyikan saat aplikasi layar penuh', 'Otomatis sembunyikan saat ada game atau video layar penuh.']
  },
  'dynamicIsland.idleHideSec': {
    en: ['Auto-hide when idle', 'Shrink into a top edge line after being idle for a chosen duration.'],
    id: ['Sembunyikan otomatis saat diam', 'Mengecil menjadi garis di tepi atas setelah diam selama waktu tertentu.']
  },
  'dynamicIsland.reset': {
    en: ['Reset Dynamic Island', 'Restore all Dynamic Island options to defaults. Enabled state stays unchanged.'],
    id: ['Atur ulang Dynamic Island', 'Semua pilihan di halaman ini kembali ke awal. Nyala atau matinya tidak berubah.']
  }
};

export const PREFERENCE_CATEGORIES = [
  { id: 'appearance', titleKey: 'cat.appearance', title: 'Appearance', icon: 'ph-palette' },
  { id: 'player', titleKey: 'cat.player', title: 'Player', icon: 'ph-play-circle' },
  { id: 'lyrics', titleKey: 'cat.lyrics', title: 'Lyrics', icon: 'ph-microphone-stage' },
  { id: 'content', titleKey: 'cat.content', title: 'Content', icon: 'ph-globe' },
  { id: 'storage', titleKey: 'cat.storage', title: 'Storage', icon: 'ph-hard-drive' },
  { id: 'privacy', titleKey: 'cat.privacy', title: 'Privacy', icon: 'ph-shield-check' },
  { id: 'backup', titleKey: 'cat.backup', title: 'Velora Jam', icon: 'ph-broadcast' },
  { id: 'system', titleKey: 'cat.system', title: 'System', icon: 'ph-desktop' },
  { id: 'dynamic-island', titleKey: 'cat.dynamic-island', title: 'Dynamic Island', icon: 'ph-sparkle' }
] as const;

export const PREFERENCE_APP_TABS = [
  { id: 'about', titleKey: 'cat.about', title: 'About', icon: 'ph-info' }
] as const;

export const PREFERENCE_SETTINGS: PreferenceItemDef[] = [
  // 1. Appearance (Single Dark Theme only — no light mode)
  {
    id: 'general.zoomLevel',
    category: 'appearance',
    group: 'grp.displayEffects',
    title: 'Display scale',
    description: 'Adjust interface zoom from 70% to 130%.',
    widget: 'stepper',
    defaultValue: '1',
    min: 0.7,
    max: 1.3,
    step: 0.1
  },
  {
    id: 'appearance.performance',
    category: 'appearance',
    group: 'grp.displayEffects',
    title: 'Animation effects',
    description: 'Control interface transitions and motion intensity.',
    widget: 'segmented',
    defaultValue: 'full',
    options: [
      { value: 'full', label: 'opt.full' },
      { value: 'lite', label: 'opt.lite' }
    ]
  },
  {
    id: 'appearance.shade',
    category: 'appearance',
    group: 'grp.displayEffects',
    title: 'Shadow effects',
    description: 'Adjust card and surface shadow intensity.',
    widget: 'segmented',
    defaultValue: 'soft',
    options: [
      { value: 'off', label: 'opt.off' },
      { value: 'soft', label: 'opt.soft' },
      { value: 'strong', label: 'opt.strong' }
    ]
  },
  {
    id: 'layout.hideLeft',
    category: 'appearance',
    group: 'grp.layout',
    title: 'Hide left sidebar',
    description: 'Permanently collapse the left navigation bar.',
    widget: 'toggle',
    defaultValue: 'false'
  },
  {
    id: 'layout.hideRight',
    category: 'appearance',
    group: 'grp.layout',
    title: 'Hide right panel',
    description: 'Permanently hide the right studio panel.',
    widget: 'toggle',
    defaultValue: 'false'
  },

  // 2. Player
  {
    id: 'playback.autoplay',
    category: 'player',
    group: 'grp.playback',
    title: 'Autoplay',
    description: 'Continue with similar songs when the queue ends.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'playback.audioNormalize',
    category: 'player',
    group: 'grp.playback',
    title: 'Normalize loudness',
    description: 'Keep consistent volume across tracks.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'playback.waveBar',
    category: 'player',
    group: 'grp.playback',
    title: 'Wavy progress bar',
    description: 'Display an animated wave along the playback progress bar.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'stream_quality',
    category: 'player',
    group: 'grp.sound',
    title: 'Audio streaming quality',
    description: 'Select streaming audio bitrate.',
    widget: 'select',
    defaultValue: 'high',
    options: [
      { value: 'high', label: 'opt.high' },
      { value: 'normal', label: 'opt.normal' },
      { value: 'low', label: 'opt.low' }
    ]
  },
  {
    id: 'playback.eqPreset',
    category: 'player',
    group: 'grp.sound',
    title: 'Equalizer sound profile',
    description: 'Live 5-band equalizer sound character.',
    widget: 'select',
    defaultValue: 'spotify',
    options: [
      { value: 'spotify', label: 'opt.eqStudio' },
      { value: 'flat', label: 'opt.eqFlat' },
      { value: 'bass', label: 'opt.eqBass' },
      { value: 'vocal', label: 'opt.eqVocal' },
      { value: 'electronic', label: 'opt.eqElectronic' }
    ]
  },
  {
    id: 'playback.crossfadeSec',
    category: 'player',
    group: 'grp.volumeTransitions',
    title: 'Song transition',
    description: 'Crossfade between songs.',
    widget: 'select',
    defaultValue: '0',
    options: [
      { value: '0', label: 'opt.gapless' },
      { value: '2', label: '2s' },
      { value: '4', label: '4s' },
      { value: '6', label: '6s' },
      { value: '8', label: '8s' },
      { value: '12', label: '12s' }
    ]
  },
  {
    id: 'playback.skipSilence',
    category: 'player',
    group: 'grp.volumeTransitions',
    title: 'Skip silence',
    description: 'Skip silent gaps at start and end of tracks.',
    widget: 'toggle',
    defaultValue: 'false'
  },
  {
    id: 'playback.radioOnPick',
    category: 'player',
    group: 'grp.queue',
    title: 'Queue similar tracks',
    description: 'Automatically add similar songs to the queue when playing a single track.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'playback.radioSize',
    category: 'player',
    group: 'grp.queue',
    title: 'Similar tracks count',
    description: 'Set how many similar songs are added automatically to the queue.',
    widget: 'segmented',
    defaultValue: '25',
    options: [
      { value: '10', label: '10' },
      { value: '25', label: '25' },
      { value: '0', label: 'opt.all' }
    ]
  },
  {
    id: 'playback.resumeQueue',
    category: 'player',
    group: 'grp.queue',
    title: 'Resume last queue',
    description: 'Restore previous queue paused on startup.',
    widget: 'toggle',
    defaultValue: 'false'
  },
  {
    id: 'playback.shuffleOnStart',
    category: 'player',
    group: 'grp.queue',
    title: 'Shuffle on startup',
    description: 'Enable shuffle when Velora opens.',
    widget: 'toggle',
    defaultValue: 'false'
  },
  {
    id: 'playback.startVolume',
    category: 'player',
    group: 'grp.controls',
    title: 'Volume on startup',
    description: 'Initial volume level.',
    widget: 'select',
    defaultValue: '-1',
    options: [
      { value: '-1', label: 'opt.last' },
      { value: '30', label: '30%' },
      { value: '50', label: '50%' },
      { value: '70', label: '70%' },
      { value: '100', label: '100%' }
    ]
  },
  {
    id: 'playback.prevRestartSec',
    category: 'player',
    group: 'grp.controls',
    title: 'Previous restarts song',
    description: 'Restart current track after N seconds.',
    widget: 'select',
    defaultValue: '3',
    options: [
      { value: '0', label: 'opt.never' },
      { value: '3', label: 'opt.3sec' },
      { value: '5', label: 'opt.5sec' },
      { value: '10', label: 'opt.10sec' }
    ]
  },

  // 3. Lyrics
  {
    id: 'lyrics.textSize',
    category: 'lyrics',
    group: 'grp.lyricsDisplay',
    title: 'Font size',
    description: 'Adjust lyrics text size.',
    widget: 'segmented',
    defaultValue: '1',
    options: [
      { value: '0', label: 'opt.small' },
      { value: '1', label: 'opt.medium' },
      { value: '2', label: 'opt.large' }
    ]
  },
  {
    id: 'lyrics.autoScroll',
    category: 'lyrics',
    group: 'grp.lyricsDisplay',
    title: 'Auto-scroll lyrics',
    description: 'Follow active synced lyric line.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'lyrics.sweep',
    category: 'lyrics',
    group: 'grp.lyricsDisplay',
    title: 'Word-by-word sweep',
    description: 'Smooth word-level highlighting.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'lyrics.blurOthers',
    category: 'lyrics',
    group: 'grp.lyricsDisplay',
    title: 'Blur inactive lines',
    description: 'Full performance mode only.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'lyrics.miniLyrics',
    category: 'lyrics',
    group: 'grp.lyricsDisplay',
    title: 'Small lyrics in mini mode',
    description: 'Compact lyrics in mini player.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'lyrics.nudgeMs',
    category: 'lyrics',
    group: 'grp.lyricsBehavior',
    title: 'Lyric timing nudge step',
    description: 'Step size for timing buttons.',
    widget: 'segmented',
    defaultValue: '500',
    options: [
      { value: '250', label: '0.25s' },
      { value: '500', label: '0.5s' },
      { value: '1000', label: '1s' }
    ]
  },
  {
    id: 'lyrics.plainFallback',
    category: 'lyrics',
    group: 'grp.lyricsBehavior',
    title: 'Show plain lyrics',
    description: 'Fallback when synced lyrics are unavailable.',
    widget: 'toggle',
    defaultValue: 'true'
  },

  // 4. Content
  {
    id: 'content.country',
    category: 'content',
    group: 'grp.regionLanguage',
    title: 'Content region',
    description: 'Catalog regions for Home and Search recommendations.',
    widget: 'multi-select',
    defaultValue: 'ID',
    options: [
      { value: 'ID', label: 'Indonesia' },
      { value: 'US', label: 'United States' },
      { value: 'GB', label: 'United Kingdom' },
      { value: 'JP', label: 'Japan' },
      { value: 'KR', label: 'South Korea' },
      { value: 'IN', label: 'India' },
      { value: 'BR', label: 'Brazil' },
      { value: 'DE', label: 'Germany' },
      { value: 'FR', label: 'France' },
      { value: 'MX', label: 'Mexico' },
      { value: 'AU', label: 'Australia' },
      { value: 'MY', label: 'Malaysia' }
    ]
  },
  {
    id: 'content.language',
    category: 'content',
    group: 'grp.regionLanguage',
    title: 'App & Dynamic Island language',
    description: 'Interface language for Velora & Dynamic Island.',
    widget: 'select',
    defaultValue: 'en',
    options: SUPPORTED_LANGUAGES.map((l) => ({ value: l.code, label: l.label }))
  },
  {
    id: 'content.searchStart',
    category: 'content',
    group: 'grp.search',
    title: 'Default search filter',
    description: 'Initial tab when searching.',
    widget: 'select',
    defaultValue: 'song',
    options: [
      { value: 'song', label: 'nav.songs' },
      { value: 'album', label: 'nav.albums' },
      { value: 'artist', label: 'nav.artists' },
      { value: 'playlist', label: 'nav.playlists' },
      { value: 'video', label: 'nav.videos' }
    ]
  },
  {
    id: 'content.hideExplicit',
    category: 'content',
    group: 'grp.search',
    title: 'Hide explicit songs',
    description: 'Filter explicit tracks from search results and auto-queued recommendations.',
    widget: 'toggle',
    defaultValue: 'false'
  },

  // 5. Storage
  {
    id: 'storage.clearCache',
    category: 'storage',
    group: 'grp.songCache',
    title: 'Clear audio and artwork cache',
    description: 'Delete temporary streamed files and cover art.',
    widget: 'action',
    defaultValue: '',
    actionLabel: 'nav.clear'
  },
  {
    id: 'storage.cacheLimit',
    category: 'storage',
    group: 'grp.songCache',
    title: 'Cache limit',
    description: 'Recent songs kept cached.',
    widget: 'segmented',
    defaultValue: '40',
    options: [
      { value: '20', label: '20' },
      { value: '40', label: '40' },
      { value: '100', label: '100' }
    ]
  },
  {
    id: 'storage.autoDownloadOnLike',
    category: 'storage',
    group: 'grp.downloads',
    title: 'Download liked songs',
    description: 'Save liked songs for offline playback.',
    widget: 'toggle',
    defaultValue: 'false'
  },
  {
    id: 'storage.clearCacheOnExit',
    category: 'storage',
    group: 'grp.cleanup',
    title: 'Clear song cache on exit',
    description: 'Downloads are preserved.',
    widget: 'toggle',
    defaultValue: 'false'
  },

  // 6. Privacy
  {
    id: 'history.enabled',
    category: 'privacy',
    group: 'grp.history',
    title: 'Save listening history',
    description: 'Stored locally on this PC only.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'privacy.saveSearches',
    category: 'privacy',
    group: 'grp.automatic',
    title: 'Save search history',
    description: 'Off: new searches are not recorded.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'privacy.historyDays',
    category: 'privacy',
    group: 'grp.automatic',
    title: 'Auto-delete listening history',
    description: 'Older records are removed automatically.',
    widget: 'select',
    defaultValue: '0',
    options: [
      { value: '0', label: 'opt.forever' },
      { value: '30', label: 'opt.30days' },
      { value: '90', label: 'opt.90days' },
      { value: '365', label: 'opt.1year' }
    ]
  },

  // 7. Velora Jam
  {
    id: 'integrations.jam.enabled',
    category: 'backup',
    group: 'grp.syncPhone',
    title: 'Wi-Fi remote control',
    description: 'Control playback from another device on Wi-Fi.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'integrations.jam.remoteUrl',
    category: 'backup',
    group: 'grp.syncPhone',
    title: 'Remote address',
    description: 'Open in browser on another device.',
    widget: 'info',
    defaultValue: ''
  },

  // 8. System
  {
    id: 'system.mediaKeys',
    category: 'system',
    group: 'grp.keysShortcuts',
    title: 'Media keys',
    description: 'Hardware play/pause/next/prev keys.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'system.closeToTray',
    category: 'system',
    group: 'grp.trayWindow',
    title: 'Hide to System Tray on close',
    description: 'Keep music playing in background.',
    widget: 'toggle',
    defaultValue: 'false'
  },
  {
    id: 'system.keepAwake',
    category: 'system',
    group: 'grp.power',
    title: 'Keep PC awake while playing',
    description: 'Display may still turn off.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'system.hideTitle',
    category: 'system',
    group: 'grp.trayWindow',
    title: 'Hide song title in window bar',
    description: 'Window title stays Velora.',
    widget: 'toggle',
    defaultValue: 'false'
  },

  // 9. Dynamic Island (No Light mode — pure black dark theme)
  {
    id: 'dynamicIsland.enabled',
    category: 'dynamic-island',
    group: 'grp.islandShapeSize',
    title: 'Dynamic Island',
    description: 'Compact top-edge player for playback & lyrics.',
    widget: 'toggle',
    defaultValue: 'true'
  },
  {
    id: 'dynamicIsland.style',
    category: 'dynamic-island',
    group: 'grp.islandShapeSize',
    title: 'Position',
    description: 'Docked at edge/notch or floating pill.',
    widget: 'segmented',
    defaultValue: 'notch',
    options: [
      { value: 'notch', label: 'opt.notch' },
      { value: 'pill', label: 'opt.pill' }
    ],
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.compactStyle',
    category: 'dynamic-island',
    group: 'grp.islandShapeSize',
    title: 'Compact style',
    description: 'Closed Dynamic Island appearance.',
    widget: 'segmented',
    defaultValue: 'RINGKAS',
    options: [
      { value: 'RINGKAS', label: 'opt.compact' },
      { value: 'SATU_BARIS', label: 'opt.oneLineLyric' },
      { value: 'TERSEMBUNYI', label: 'opt.hidden' }
    ],
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.size',
    category: 'dynamic-island',
    group: 'grp.islandShapeSize',
    title: 'Size',
    description: 'Scale preset.',
    widget: 'segmented',
    defaultValue: 'medium',
    options: [
      { value: 'small', label: 'opt.small' },
      { value: 'medium', label: 'opt.normal' },
      { value: 'large', label: 'opt.large' }
    ],
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.opacity',
    category: 'dynamic-island',
    group: 'grp.islandShapeSize',
    title: 'Opacity',
    description: 'Background opacity.',
    widget: 'slider',
    defaultValue: '100',
    min: 0,
    max: 100,
    step: 5,
    unit: '%',
    startLabel: '0%',
    endLabel: '100%',
    snapPoints: [0, 25, 50, 75, 100],
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.showStatus',
    category: 'dynamic-island',
    group: 'grp.islandContent',
    title: 'Clock & system status',
    description: 'Top-row clock and connectivity icons.',
    widget: 'toggle',
    defaultValue: 'true',
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.showCover',
    category: 'dynamic-island',
    group: 'grp.islandContent',
    title: 'Show cover art',
    description: 'Display album cover.',
    widget: 'toggle',
    defaultValue: 'true',
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.showClosedCover',
    category: 'dynamic-island',
    group: 'grp.islandContent',
    title: 'Show cover when closed',
    description: 'Album art on the left when closed.',
    widget: 'toggle',
    defaultValue: 'true',
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.showArtist',
    category: 'dynamic-island',
    group: 'grp.islandContent',
    title: 'Show artist name',
    description: 'Display artist subtitle.',
    widget: 'toggle',
    defaultValue: 'true',
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.showLyric',
    category: 'dynamic-island',
    group: 'grp.islandContent',
    title: 'Show lyric line',
    description: 'Active lyric line.',
    widget: 'toggle',
    defaultValue: 'true',
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.showHeart',
    category: 'dynamic-island',
    group: 'grp.islandContent',
    title: 'Show heart button',
    description: 'Quick favorite toggle.',
    widget: 'toggle',
    defaultValue: 'true',
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.showDots',
    category: 'dynamic-island',
    group: 'grp.islandContent',
    title: 'Show page dots',
    description: 'Page switcher dots.',
    widget: 'toggle',
    defaultValue: 'true',
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.showProgress',
    category: 'dynamic-island',
    group: 'grp.islandContent',
    title: 'Show progress bar',
    description: 'Track progress indicator.',
    widget: 'toggle',
    defaultValue: 'true',
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.progressStyle',
    category: 'dynamic-island',
    group: 'grp.islandContent',
    title: 'Progress bar style',
    description: 'Visual style.',
    widget: 'segmented',
    defaultValue: 'thin',
    options: [
      { value: 'thin', label: 'opt.thin' },
      { value: 'thick', label: 'opt.thick' },
      { value: 'dots', label: 'opt.dots' },
      { value: 'wave', label: 'opt.wave' }
    ],
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.animSpeed',
    category: 'dynamic-island',
    group: 'grp.islandMotion',
    title: 'Animation speed',
    description: 'Off: shapes snap instantly.',
    widget: 'segmented',
    defaultValue: 'smooth',
    options: [
      { value: 'off', label: 'opt.off' },
      { value: 'normal', label: 'opt.normal' },
      { value: 'smooth', label: 'opt.smooth' }
    ],
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.openTrigger',
    category: 'dynamic-island',
    group: 'grp.islandBehavior',
    title: 'Open trigger',
    description: 'How Dynamic Island expands.',
    widget: 'segmented',
    defaultValue: 'hover',
    options: [
      { value: 'hover', label: 'opt.onHover' },
      { value: 'click', label: 'opt.onClick' }
    ],
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.onlyPlaying',
    category: 'dynamic-island',
    group: 'grp.islandBehavior',
    title: 'Show only while playing',
    description: 'Hide whenever playback is paused.',
    widget: 'toggle',
    defaultValue: 'false',
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.pauseHide',
    category: 'dynamic-island',
    group: 'grp.islandBehavior',
    title: 'Hide when paused',
    description: 'Auto-hide delay after pausing.',
    widget: 'segmented',
    defaultValue: '0',
    options: [
      { value: '0', label: 'opt.no' },
      { value: '10', label: '10s' },
      { value: '30', label: '30s' },
      { value: '60', label: '60s' }
    ],
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' },
    disabledWhen: { id: 'dynamicIsland.onlyPlaying', equals: 'true' }
  },
  {
    id: 'dynamicIsland.screen',
    category: 'dynamic-island',
    group: 'grp.islandScreen',
    title: 'Display',
    description: 'Which monitor shows Dynamic Island.',
    widget: 'segmented',
    defaultValue: 'primary',
    options: [
      { value: 'primary', label: 'opt.primaryScreen' },
      { value: 'cursor', label: 'opt.cursorScreen' }
    ],
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.hideApp',
    category: 'dynamic-island',
    group: 'grp.islandScreen',
    title: 'Hide when Velora window is active',
    description: 'Hide while Velora is focused.',
    widget: 'toggle',
    defaultValue: 'true',
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.hideFull',
    category: 'dynamic-island',
    group: 'grp.islandScreen',
    title: 'Hide in fullscreen apps',
    description: 'Hide during fullscreen games/videos.',
    widget: 'toggle',
    defaultValue: 'true',
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.idleHideSec',
    category: 'dynamic-island',
    group: 'grp.islandScreen',
    title: 'Auto-hide when idle',
    description: 'Hide to top line after idle seconds.',
    widget: 'combobox',
    defaultValue: '0',
    min: 0,
    max: 3600,
    unit: 's',
    options: [
      { value: '0', label: 'Off' },
      { value: '5', label: '5s' },
      { value: '15', label: '15s' },
      { value: '30', label: '30s' }
    ],
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' },
    disabledWhen: { id: 'dynamicIsland.hideFull', equals: 'true' }
  },
  {
    id: 'dynamicIsland.reset',
    category: 'dynamic-island',
    group: 'grp.islandReset',
    title: 'Reset Dynamic Island',
    description: 'Restore Dynamic Island options to defaults.',
    widget: 'action',
    defaultValue: '',
    actionLabel: 'nav.reset',
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'about.dataFolder',
    category: 'about',
    group: 'nav.aboutTitle',
    title: 'Data folder',
    description: 'Open local data folder.',
    widget: 'action',
    defaultValue: '',
    actionLabel: 'nav.open'
  }
];

export function localizeItem(item: PreferenceItemDef, lang: LangCode | string): PreferenceItemDef {
  const code = normalizeLang(lang);
  const pair = ITEM_TEXT[item.id];
  const [title, description] = pair ? (code === 'id' ? pair.id : pair.en) : [item.title, item.description];
  return {
    ...item,
    group: item.group.startsWith('grp.') ? tr(code, item.group, item.group) : item.group,
    title,
    description,
    actionLabel: item.actionLabel?.startsWith('nav.') ? tr(code, item.actionLabel, item.actionLabel) : item.actionLabel,
    options: item.options?.map((o) => ({
      value: o.value,
      label: o.label.includes('.') ? tr(code, o.label, o.label) : o.label
    }))
  };
}

export function getPrefValue(prefs: Record<string, string>, id: string): string {
  if (prefs[id] !== undefined) return prefs[id];
  const def = PREFERENCE_SETTINGS.find((s) => s.id === id);
  return def ? def.defaultValue : '';
}

export function getPrefBool(prefs: Record<string, string>, id: string): boolean {
  return getPrefValue(prefs, id) === 'true';
}
