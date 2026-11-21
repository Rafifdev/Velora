# Privacy Policy — Velora

**Last Updated:** November 21, 2026  
**Application:** Velora Desktop for Windows  
**Developer:** Yoapipp ([GitHub: Rafifdev](https://github.com/Rafifdev))  
**Repository:** https://github.com/Rafifdev/Velora

---

## English

### 1. Overview
Velora is a desktop music player for Windows designed with a local-first architecture. Protecting user privacy is fundamental to Velora: the application does **not** require account registration, does **not** run third-party advertising or behavioral tracking SDKs, and does **not** collect or transmit personal data to developer-owned servers.

### 2. Personal Data Collection
Velora does **not** collect, store, sell, rent, or transmit personally identifiable information (such as your name, email address, phone number, payment details, precise location, or device identifiers) to the developer or any third-party analytics service.

### 3. Local On-Device Storage
All user preferences and listening data generated while using Velora are stored **100% locally** on your Windows device inside the application data directory (`%AppData%\Velora`), specifically:
- **SQLite Database (`velora.db`)**: Preferences, playlists, favorite tracks/albums/artists, listening history, listening statistics, daily fire streak progress, and local search history (which can be disabled in Settings).
- **Audio & Artwork Cache (`artwork_cache/`, `uploads/`, `downloads/`)**: Album artwork thumbnails, offline downloaded tracks, and temporary audio stream cache.

None of this locally stored data is uploaded to or synced with any cloud server controlled by the developer.

### 4. Internet Access & Third-Party Services
Velora uses internet access solely to provide core music playback, metadata lookup, lyrics, and update-checking features requested by the user:
- **Music Streaming & Catalog Metadata**: Queries public music catalog metadata, artwork, and audio streams via YouTube Music (InnerTube) and Deezer APIs, as well as optional public playlist imports from Spotify URLs.
- **Time-Synced Lyrics**: Queries public lyrics providers (**LRCLIB**, **NetEase Cloud Music**, **KuGou Music**, **YouTube Music**, and **Lyrics.ovh**) using track title, artist, album, and duration metadata only.
- **Application Update Checks**: Checks public release metadata via the GitHub Releases API (`https://api.github.com/repos/Rafifdev/Velora/releases/latest`) when triggered from Settings → About.

When Velora connects directly to these third-party endpoints from your device, standard network connection metadata (such as your IP address and HTTP request headers) may be processed by those external services according to their respective privacy policies.

### 5. Local Network Access (Velora Jam)
Velora includes an optional feature called **Velora Jam**, which starts a lightweight HTTP/WebSocket server strictly on your **local area network (LAN)** so you can control playback from another device on the same Wi-Fi network.
- Velora Jam requires explicit permission via **Settings → Velora Jam → Allow Velora Jam (Wi-Fi remote)** and must be manually turned ON/OFF by the user.
- No playback data or commands from Velora Jam are routed through external internet servers.

### 6. System & Hardware Access
Velora accesses local system capabilities only as needed for desktop music player functionality:
- **Local File System**: Reads audio files and folders explicitly added or selected by the user, and exports/imports `.M3U` playlists when requested.
- **Windows Audio Endpoints & Bluetooth Info**: Reads connected audio output devices (speakers, wired headphones, Bluetooth devices) and battery status metadata locally to display and switch audio outputs inside the Dynamic Island overlay.

### 7. User Control & Data Deletion
You retain complete control over your local data at all times:
- **Disable History or Search Logging**: Toggle **Save listening history** or **Save search history** off under **Settings → Privacy**, or set an automatic deletion schedule (`30 days`, `90 days`, or `1 year`).
- **Clear Cache & History**: Remove cached artwork/streams under **Settings → Storage → Clear all cache** or clear playback history from the **History** page.
- **Complete Removal**: Uninstalling Velora or deleting the `%AppData%\Velora` folder permanently deletes all locally stored application data.

### 8. Children's Privacy
Velora does not knowingly collect any personal information from children or any users of any age, and includes an optional **Hide explicit songs** filter under **Settings → Content**.

### 9. Changes to This Privacy Policy
We may update this Privacy Policy from time to time if new features are added. Any updates will be published on this page with an updated "Last Updated" date.

### 10. Contact Information
If you have questions or concerns regarding this Privacy Policy or Velora's data practices, please contact the developer through:
- **GitHub Issues**: https://github.com/Rafifdev/Velora/issues
- **GitHub Profile**: https://github.com/Rafifdev
- **Instagram**: [@yoapipp](https://www.instagram.com/yoapipp/)

---

## Bahasa Indonesia

### 1. Ringkasan
Velora adalah aplikasi pemutar musik desktop untuk Windows yang dirancang dengan arsitektur *local-first*. Velora **tidak** memerlukan akun atau login, **tidak** menyertakan pelacak iklan maupun analitik pihak ketiga, dan **tidak** mengirimkan data pribadi pengguna ke server pengembang.

### 2. Pengumpulan Data Pribadi
Velora **tidak** mengumpulkan, menyimpan, menjual, maupun mengirimkan informasi pribadi pengguna (seperti nama, email, nomor telepon, detail pembayaran, lokasi akurat, atau pengenal perangkat) ke server pengembang maupun pihak ketiga.

### 3. Penyimpanan Lokal di Perangkat
Seluruh preferensi dan data pemutaran musik disimpan **100% secara lokal** di perangkat Windows pengguna pada folder data aplikasi (`%AppData%\Velora`):
- **Basis Data SQLite (`velora.db`)**: Pengaturan aplikasi, playlist, lagu/album/artis favorit, riwayat putar, statistik mendengarkan, *fire streak* harian, dan riwayat pencarian lokal.
- **Cache Audio & Sampul (`artwork_cache/`, `uploads/`, `downloads/`)**: Cache sampul album, lagu unduhan offline, dan file sementara streaming audio.

### 4. Koneksi Internet & Layanan Pihak Ketiga
Velora mengakses internet hanya untuk fitur pemutaran dan pengambilan metadata lagu atas permintaan pengguna:
- **Streaming & Katalog Musik**: Mengambil metadata lagu, sampul album, dan stream audio dari YouTube Music (InnerTube) dan Deezer, serta impor playlist publik Spotify via tautan yang dimasukkan pengguna.
- **Lirik Tersinkronisasi**: Mengambil lirik dari penyedia publik (**LRCLIB**, **NetEase Cloud Music**, **KuGou Music**, **YouTube Music**, dan **Lyrics.ovh**) hanya dengan metadata judul, artis, album, dan durasi.
- **Cek Pembaruan**: Memeriksa rilis terbaru melalui API publik GitHub Releases saat pengguna menekan tombol cek pembaruan.

### 5. Akses Jaringan Lokal (Velora Jam)
Fitur opsional **Velora Jam** menjalankan server lokal khusus di **jaringan Wi-Fi lokal (LAN)** agar pengguna dapat mengontrol pemutaran dari browser HP di jaringan yang sama. Fitur ini memerlukan izin dan aktivasi manual dari pengguna serta tidak mengirimkan data keluar jaringan lokal.

### 6. Kontrol Pengguna & Penghapusan Data
Pengguna memegang kendali penuh untuk menonaktifkan penyimpanan riwayat/pencarian di **Pengaturan → Privasi**, membersihkan cache di **Pengaturan → Penyimpanan**, menghapus riwayat putar di halaman **Riwayat**, atau menghapus seluruh folder `%AppData%\Velora` saat melakukan uninstalasi.

### 7. Kontak Pengembang
Pertanyaan terkait kebijakan privasi ini dapat diajukan melalui:
- **GitHub Issues**: https://github.com/Rafifdev/Velora/issues
- **GitHub**: https://github.com/Rafifdev
- **Instagram**: [@yoapipp](https://www.instagram.com/yoapipp/)
