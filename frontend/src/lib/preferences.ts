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

export type LocalizedPair = Partial<Record<LangCode, [string, string]>> & {
  en: [string, string];
  id: [string, string];
};

const ITEM_TEXT: Record<string, LocalizedPair> = {
  'general.zoomLevel': {
    en: ['Display scale', 'Adjust interface zoom level between 70% and 130%.'],
    id: ['Skala tampilan', 'Atur tingkat pembesaran antarmuka dari 70% hingga 130%.'],
    ms: ['Skala paparan', 'Laraskan tahap zum antara muka dari 70% hingga 130%.'],
    ja: ['表示スケール', 'インターフェースの拡大率を70%から130%の間で調整します。'],
    ko: ['화면 배율', '인터페이스 확대/축소 비율을 70%에서 130% 사이로 조절합니다.'],
    hi: ['डिस्प्ले स्केल', 'इंटरफ़ेस ज़ूम स्तर को 70% से 130% के बीच समायोजित करें।'],
    pt: ['Escala de exibição', 'Ajuste o nível de zoom da interface entre 70% e 130%.'],
    de: ['Anzeigeskalierung', 'Passe die Zoomstufe der Oberfläche zwischen 70 % und 130 % an.'],
    fr: ['Échelle d’affichage', 'Ajustez le niveau de zoom de l’interface entre 70 % et 130 %.'],
    es: ['Escala de pantalla', 'Ajusta el nivel de zoom de la interfaz entre 70% y 130%.']
  },
  'appearance.performance': {
    en: ['Animation effects', 'Control interface transitions and motion intensity.'],
    id: ['Efek animasi', 'Atur transisi dan intensitas animasi antarmuka.'],
    ms: ['Kesan animasi', 'Kawal peralihan antara muka dan keamatan gerakan.'],
    ja: ['アニメーション効果', '画面遷移とアニメーションの強度を制御します。'],
    ko: ['애니메이션 효과', '인터페이스 전환 및 모션 강도를 조절합니다.'],
    hi: ['एनिमेशन प्रभाव', 'इंटरफ़ेस ट्रांज़िशन और मोशन तीव्रता को नियंत्रित करें।'],
    pt: ['Efeitos de animação', 'Controle as transições da interface e intensidade de movimento.'],
    de: ['Animationseffekte', 'Steuere Übergänge und Bewegungsintensität der Oberfläche.'],
    fr: ['Effets d’animation', 'Contrôlez les transitions et l’intensité des animations.'],
    es: ['Efectos de animación', 'Controla las transiciones de la interfaz y la intensidad del movimiento.']
  },
  'appearance.shade': {
    en: ['Shadow effects', 'Adjust shadow depth on cards and floating panels.'],
    id: ['Efek bayangan', 'Atur intensitas bayangan pada kartu dan panel.'],
    ms: ['Kesan bayang', 'Laraskan kedalaman bayang pada kad dan panel terapung.'],
    ja: ['シャドウ効果', 'カードやパネルの影の深さを調整します。'],
    ko: ['그림자 효과', '카드 및 플로팅 패널의 그림자 깊이를 조절합니다.'],
    hi: ['छाया प्रभाव', 'कार्ड और पैनल पर छाया की गहराई समायोजित करें।'],
    pt: ['Efeitos de sombra', 'Ajuste a profundidade das sombras nos cartões e painéis.'],
    de: ['Schatteneffekte', 'Passe die Schattentiefe auf Karten und Panels an.'],
    fr: ['Effets d’ombre', 'Ajustez la profondeur des ombres sur les cartes et panneaux.'],
    es: ['Efectos de sombra', 'Ajusta la profundidad de sombra en tarjetas y paneles.']
  },
  'appearance.showStreak': {
    en: ['Streak on Home', 'Show listening streak badge on the Home screen.'],
    id: ['Streak di Beranda', 'Tampilkan lencana streak mendengarkan di Beranda.']
  },
  'layout.hideLeft': {
    en: ['Hide left sidebar', 'Keep the left navigation bar hidden until re-enabled here.'],
    id: ['Sembunyikan bilah kiri', 'Sembunyikan bilah navigasi kiri secara permanen hingga diaktifkan kembali.'],
    ms: ['Sembunyikan bar sisi kiri', 'Pastikan bar navigasi kiri tersembunyi sehingga diaktifkan semula di sini.'],
    ja: ['左サイドバーを隠す', 'ここで再度有効にするまで左ナビゲーションバーを非表示にします。'],
    ko: ['왼쪽 사이드바 숨기기', '여기서 다시 활성화할 때까지 왼쪽 탐색 바를 숨깁니다.'],
    hi: ['बायाँ साइडबार छिपाएं', 'यहाँ से दोबारा चालू करने तक बाएं नेविगेशन बार को छिपाए रखें।'],
    pt: ['Ocultar barra lateral esquerda', 'Mantém a barra de navegação esquerda oculta até ser reativada aqui.'],
    de: ['Linke Seitenleiste ausblenden', 'Blendet die linke Navigationsleiste dauerhaft aus, bis sie hier wieder aktiviert wird.'],
    fr: ['Masquer la barre latérale gauche', 'Garde la barre de navigation gauche masquée jusqu’à sa réactivation ici.'],
    es: ['Ocultar barra lateral izquierda', 'Mantiene oculta la barra de navegación izquierda hasta reactivarla aquí.']
  },
  'layout.hideRight': {
    en: ['Hide right panel', 'Keep the right studio panel hidden until re-enabled here.'],
    id: ['Sembunyikan panel kanan', 'Sembunyikan panel kanan secara permanen hingga diaktifkan kembali.'],
    ms: ['Sembunyikan panel kanan', 'Pastikan panel studio kanan tersembunyi sehingga diaktifkan semula di sini.'],
    ja: ['右パネルを隠す', 'ここで再度有効にするまで右パネルを非表示にします。'],
    ko: ['오른쪽 패널 숨기기', '여기서 다시 활성화할 때까지 오른쪽 패널을 숨깁니다.'],
    hi: ['दायाँ पैनल छिपाएं', 'यहाँ से दोबारा चालू करने तक दाएं पैनल को छिपाए रखें।'],
    pt: ['Ocultar painel direito', 'Mantém o painel direito oculto até ser reativado aqui.'],
    de: ['Rechtes Panel ausblenden', 'Blendet das rechte Panel dauerhaft aus, bis es hier wieder aktiviert wird.'],
    fr: ['Masquer le panneau droit', 'Garde le panneau droit masqué jusqu’à sa réactivation ici.'],
    es: ['Ocultar panel derecho', 'Mantiene oculto el panel derecho hasta reactivarlo aquí.']
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

  'playback.autoplay': {
    en: ['Autoplay', 'Automatically continue with similar tracks when the queue ends.'],
    id: ['Putar otomatis', 'Lanjutkan pemutaran dengan lagu serupa saat antrean berakhir.'],
    ms: ['Main automatik', 'Teruskan secara automatik dengan lagu serupa apabila baris gilir tamat.'],
    ja: ['自動再生', 'キューが終了したら似た曲を自動的に再生し続けます。'],
    ko: ['자동 재생', '대기열이 끝나면 비슷한 곡을 자동으로 이어서 재생합니다.'],
    hi: ['ऑटोप्ले', 'कतार समाप्त होने पर स्वचालित रूप से मिलते-जुलते गाने चलाएं।'],
    pt: ['Reprodução automática', 'Continua automaticamente com músicas semelhantes quando a fila termina.'],
    de: ['Autoplay', 'Spielt automatisch ähnliche Titel ab, wenn die Warteschlange endet.'],
    fr: ['Lecture automatique', 'Continue automatiquement avec des titres similaires à la fin de la file.'],
    es: ['Reproducción automática', 'Continúa automáticamente con canciones similares al terminar la cola.']
  },
  'playback.audioNormalize': {
    en: ['Normalize volume', 'Maintain a consistent loudness level across different tracks.'],
    id: ['Normalisasi volume', 'Jaga tingkat kenyaringan suara tetap seimbang antar lagu.'],
    ms: ['Normalkan kelantangan', 'Kekalkan tahap kelantangan yang konsisten merentas lagu berbeza.'],
    ja: ['音量の均一化', '異なる曲間で一定の音量レベルを維持します。'],
    ko: ['음량 평준화', '서로 다른 곡 간에 일정한 음량 수준을 유지합니다.'],
    hi: ['वॉल्यूम सामान्यीकरण', 'विभिन्न गानों के बीच एक समान आवाज़ स्तर बनाए रखें।'],
    pt: ['Normalizar volume', 'Mantém um nível de volume consistente entre diferentes músicas.'],
    de: ['Lautstärke normalisieren', 'Hält eine gleichmäßige Lautstärke über verschiedene Titel hinweg.'],
    fr: ['Normaliser le volume', 'Maintient un niveau sonore constant entre les différents titres.'],
    es: ['Normalizar volumen', 'Mantiene un nivel de volumen uniforme entre distintas canciones.']
  },
  'playback.waveBar': {
    en: ['Wavy progress bar', 'Display an animated wave along the playback progress bar.'],
    id: ['Bilah progres bergelombang', 'Tampilkan garis bergelombang bergerak pada bilah progres pemutaran.'],
    ms: ['Bar kemajuan beralun', 'Paparkan gelombang bergerak sepanjang bar kemajuan main balik.'],
    ja: ['波形プログレスバー', '再生プログレスバーに沿ってアニメーション波形を表示します。'],
    ko: ['물결 진행 바', '재생 진행 표시줄에 애니메이션 물결을 표시합니다.'],
    hi: ['लहरदार प्रोग्रेस बार', 'प्लेबैक प्रोग्रेस बार के साथ एनिमेटेड लहर दिखाएं।'],
    pt: ['Barra de progresso ondulada', 'Exibe uma onda animada ao longo da barra de progresso.'],
    de: ['Wellenförmiger Fortschrittsbalken', 'Zeigt eine animierte Welle entlang des Wiedergabefortschritts.'],
    fr: ['Barre de progression ondulée', 'Affiche une onde animée le long de la barre de progression.'],
    es: ['Barra de progreso ondulada', 'Muestra una onda animada a lo largo de la barra de progreso.']
  },
  'stream_quality': {
    en: ['Audio streaming quality', 'Select the preferred audio bitrate for online playback.'],
    id: ['Kualitas streaming audio', 'Pilih tingkat kualitas bitrate audio saat memutar musik daring.'],
    ms: ['Kualiti penstriman audio', 'Pilih kadar bit audio pilihan untuk main balik dalam talian.'],
    ja: ['オーディオストリーミング品質', 'オンライン再生時の優先ビットレートを選択します。'],
    ko: ['오디오 스트리밍 품질', '온라인 재생에 선호하는 오디오 비트레이트를 선택합니다.'],
    hi: ['ऑडियो स्ट्रीमिंग गुणवत्ता', 'ऑनलाइन प्लेबैक के लिए पसंदीदा ऑडियो बिटरेट चुनें।'],
    pt: ['Qualidade de streaming de áudio', 'Selecione a taxa de bits preferida para reprodução online.'],
    de: ['Audio-Streaming-Qualität', 'Wähle die bevorzugte Audio-Bitrate für die Online-Wiedergabe.'],
    fr: ['Qualité du streaming audio', 'Sélectionnez le débit audio préféré pour la lecture en ligne.'],
    es: ['Calidad de transmisión de audio', 'Selecciona la tasa de bits preferida para la reproducción en línea.']
  },
  'playback.eqPreset': {
    en: ['Equalizer preset', 'Select a real-time 5-band equalizer sound profile.'],
    id: ['Preset Equalizer', 'Pilih karakter suara equalizer lima pita secara langsung.'],
    ms: ['Pratetap Penyama (EQ)', 'Pilih profil bunyi penyama 5-jalur masa nyata.'],
    ja: ['イコライザープリセット', 'リアルタイム5バンドイコライザーのサウンドプロファイルを選択します。'],
    ko: ['이퀄라이저 프리셋', '실시간 5밴드 이퀄라이저 사운드 프로필을 선택합니다.'],
    hi: ['इक्वलाइज़र प्रीसेट', 'रीयल-टाइम 5-बैंड इक्वलाइज़र साउंड प्रोफ़ाइल चुनें।'],
    pt: ['Predefinição do Equalizador', 'Selecione um perfil de som do equalizador de 5 bandas em tempo real.'],
    de: ['Equalizer-Voreinstellung', 'Wähle ein Echtzeit-5-Band-Equalizer-Klangprofil.'],
    fr: ['Préréglage de l’égaliseur', 'Sélectionnez un profil sonore d’égaliseur 5 bandes en temps réel.'],
    es: ['Preajuste del ecualizador', 'Selecciona un perfil de sonido del ecualizador de 5 bandas en tiempo real.']
  },
  'playback.crossfadeSec': {
    en: ['Crossfade', 'Smoothly blend the end of one track into the start of the next.'],
    id: ['Crossfade antar lagu', 'Padukan akhir lagu secara halus ke awal lagu berikutnya.'],
    ms: ['Crossfade antara lagu', 'Adunkan penghujung lagu secara lancar ke permulaan lagu seterusnya.'],
    ja: ['クロスフェード', '曲の終わりを次の曲の始まりに滑らかに重ねます。'],
    ko: ['크로스페이드', '한 곡의 끝을 다음 곡의 시작 부분과 부드럽게 연결합니다.'],
    hi: ['क्रॉसफ़ेड', 'एक गाने के अंत को अगले गाने की शुरुआत में सहजता से मिलाएं।'],
    pt: ['Crossfade entre músicas', 'Misture suavemente o final de uma faixa com o início da próxima.'],
    de: ['Crossfade', 'Blende das Ende eines Titels sanft in den Beginn des nächsten über.'],
    fr: ['Fondu enchaîné (Crossfade)', 'Enchaîne progressivement la fin d’un titre avec le début du suivant.'],
    es: ['Crossfade entre canciones', 'Mezcla suavemente el final de una pista con el inicio de la siguiente.']
  },
  'playback.skipSilence': {
    en: ['Skip silence', 'Automatically bypass silent sections at track boundaries.'],
    id: ['Lewati bagian hening', 'Lewati bagian senyap di penghujung lagu secara otomatis.'],
    ms: ['Langkau bahagian senyap', 'Langkau bahagian senyap di sempadan lagu secara automatik.'],
    ja: ['無音部分をスキップ', '曲の前後にある無音区間を自動的にスキップします。'],
    ko: ['무음 구간 건너뛰기', '곡 시작과 끝의 무음 구간을 자동으로 건너뜁니다.'],
    hi: ['मौन भाग छोड़ें', 'गाने के शुरू और अंत में शांत हिस्सों को स्वचालित रूप से छोड़ें।'],
    pt: ['Pular silêncio', 'Ignora automaticamente trechos silenciosos no início e fim das faixas.'],
    de: ['Stille überspringen', 'Überspringt automatisch stille Abschnitte am Anfang und Ende von Titeln.'],
    fr: ['Ignorer les silences', 'Ignore automatiquement les passages silencieux en début et fin de piste.'],
    es: ['Saltar silencios', 'Omite automáticamente las secciones silenciosas al principio y final de las pistas.']
  },
  'playback.radioOnPick': {
    en: ['Queue similar tracks', 'Automatically add similar songs to the queue when playing a single track.'],
    id: ['Tambahkan lagu serupa', 'Buat antrean lagu serupa secara otomatis saat memutar satu lagu.'],
    ms: ['Tambah lagu serupa ke baris gilir', 'Tambah lagu serupa secara automatik apabila memainkan satu lagu.'],
    ja: ['似た曲をキューに追加', '1曲再生時に似た曲を自動的にキューへ追加します。'],
    ko: ['비슷한 곡 자동 대기열 추가', '단일 곡 재생 시 비슷한 노래를 대기열에 자동으로 추가합니다.'],
    hi: ['मिलते-जुलते गाने कतार में जोड़ें', 'एक गाना चलाने पर मिलते-जुलते गाने स्वचालित रूप से कतार में जोड़ें।'],
    pt: ['Enfileirar músicas semelhantes', 'Adiciona automaticamente músicas parecidas à fila ao tocar uma faixa.'],
    de: ['Ähnliche Titel einreihen', 'Fügt beim Abspielen eines einzelnen Titels automatisch ähnliche Songs zur Warteschlange hinzu.'],
    fr: ['Ajouter des titres similaires', 'Ajoute automatiquement des chansons similaires à la file lors de la lecture d’un titre.'],
    es: ['Encolar canciones similares', 'Añade automáticamente canciones similares a la cola al reproducir una sola pista.']
  },
  'playback.radioSize': {
    en: ['Similar tracks count', 'Set how many similar songs are added automatically to the queue.'],
    id: ['Jumlah lagu serupa', 'Tentukan jumlah lagu serupa yang ditambahkan otomatis ke antrean.'],
    ms: ['Bilangan lagu serupa', 'Tetapkan berapa banyak lagu serupa ditambah secara automatik.'],
    ja: ['似た曲の追加数', 'キューに自動追加する似た曲の数を設定します。'],
    ko: ['비슷한 곡 추가 개수', '대기열에 자동으로 추가할 비슷한 노래 수를 설정합니다.'],
    hi: ['मिलते-जुलते गानों की संख्या', 'कतार में स्वचालित रूप से जोड़े जाने वाले गानों की संख्या चुनें।'],
    pt: ['Quantidade de músicas semelhantes', 'Defina quantas músicas parecidas são adicionadas automaticamente à fila.'],
    de: ['Anzahl ähnlicher Titel', 'Lege fest, wie viele ähnliche Songs automatisch zur Warteschlange hinzugefügt werden.'],
    fr: ['Nombre de titres similaires', 'Définit combien de titres similaires sont ajoutés automatiquement à la file.'],
    es: ['Cantidad de canciones similares', 'Define cuántas canciones similares se añaden automáticamente a la cola.']
  },
  'playback.resumeQueue': {
    en: ['Restore queue on launch', 'Restore the previous queue and playback position in a paused state.'],
    id: ['Pulihkan antrean terakhir', 'Kembalikan antrean dan posisi lagu terakhir dalam kondisi jeda saat dibuka.'],
    ms: ['Pulihkan baris gilir semasa dibuka', 'Kembalikan baris gilir dan posisi main balik terakhir dalam keadaan jeda.'],
    ja: ['起動時に前回のキューを復元', '前回のキューと再生位置を一時停止状態で復元します。'],
    ko: ['시작 시 이전 대기열 복원', '앱 실행 시 이전 대기열과 재생 위치를 일시정지 상태로 복원합니다.'],
    hi: ['लॉन्च पर पिछली कतार बहाल करें', 'ऐप खुलने पर पिछली कतार और प्लेबैक स्थिति को पॉज़ अवस्था में बहाल करें।'],
    pt: ['Restaurar fila ao iniciar', 'Restaura a fila anterior e a posição de reprodução pausada ao abrir.'],
    de: ['Warteschlange beim Start wiederherstellen', 'Stellt die vorherige Warteschlange und Position beim Start pausiert wieder her.'],
    fr: ['Restaurer la file au lancement', 'Restaure la file précédente et la position de lecture en pause au démarrage.'],
    es: ['Restaurar cola al iniciar', 'Restaura la cola anterior y la posición de reproducción en pausa al abrir.']
  },
  'playback.shuffleOnStart': {
    en: ['Shuffle on launch', 'Enable shuffle mode automatically when Velora starts.'],
    id: ['Acak saat dibuka', 'Aktifkan mode acak secara otomatis setiap kali Velora dibuka.'],
    ms: ['Rawak semasa dibuka', 'Aktifkan mod rawak secara automatik apabila Velora bermula.'],
    ja: ['起動時にシャッフル', 'Veloraの起動時にシャッフルモードを自動的に有効にします。'],
    ko: ['시작 시 셔플 활성화', 'Velora가 시작될 때 셔플 모드를 자동으로 켭니다.'],
    hi: ['लॉन्च पर शफ़ल चालू करें', 'Velora शुरू होने पर स्वचालित रूप से शफ़ल मोड चालू करें।'],
    pt: ['Modo aleatório ao iniciar', 'Ativa o modo aleatório automaticamente quando o Velora inicia.'],
    de: ['Zufallswiedergabe beim Start', 'Aktiviert den Shuffle-Modus automatisch beim Start von Velora.'],
    fr: ['Lecture aléatoire au démarrage', 'Active automatiquement le mode aléatoire au lancement de Velora.'],
    es: ['Modo aleatorio al iniciar', 'Activa el modo aleatorio automáticamente cuando se inicia Velora.']
  },
  'playback.startVolume': {
    en: ['Startup volume', 'Set the initial playback volume when Velora launches.'],
    id: ['Volume awal', 'Atur tingkat volume awal saat Velora dibuka.'],
    ms: ['Kelantangan permulaan', 'Tetapkan kelantangan main balik awal apabila Velora dilancarkan.'],
    ja: ['起動時の音量', 'Velora起動時の初期再生音量を設定します。'],
    ko: ['시작 볼륨', 'Velora 실행 시 초기 재생 볼륨을 설정합니다.'],
    hi: ['शुरुआती वॉल्यूम', 'Velora लॉन्च होने पर प्रारंभिक प्लेबैक वॉल्यूम सेट करें।'],
    pt: ['Volume inicial', 'Defina o volume inicial de reprodução quando o Velora abrir.'],
    de: ['Startlautstärke', 'Lege die anfängliche Wiedergabelautstärke beim Start von Velora fest.'],
    fr: ['Volume au démarrage', 'Définit le volume de lecture initial au lancement de Velora.'],
    es: ['Volumen inicial', 'Establece el volumen de reproducción inicial al abrir Velora.']
  },
  'playback.prevRestartSec': {
    en: ['Previous track behavior', 'Restart the current song if playback has passed the selected duration.'],
    id: ['Perilaku tombol Sebelumnya', 'Ulangi lagu dari awal jika pemutaran telah melewati batas detik tertentu.'],
    ms: ['Tingkah laku lagu sebelumnya', 'Mulakan semula lagu semasa jika main balik melepasi tempoh dipilih.'],
    ja: ['「前へ」ボタンの動作', '選択した秒数を過ぎている場合は現在の曲を最初から再生します。'],
    ko: ['이전 곡 버튼 동작', '선택한 시간이 지난 경우 현재 곡을 처음부터 다시 시작합니다.'],
    hi: ['पिछला ट्रैक व्यवहार', 'चयनित समय बीत जाने पर वर्तमान गाने को शुरू से चलाएं।'],
    pt: ['Comportamento do botão Anterior', 'Reinicia a música atual se a reprodução tiver passado da duração selecionada.'],
    de: ['Verhalten der Zurück-Taste', 'Startet den aktuellen Titel neu, wenn die gewählte Dauer überschritten wurde.'],
    fr: ['Comportement du bouton Précédent', 'Redémarre le titre actuel si la lecture a dépassé la durée choisie.'],
    es: ['Comportamiento del botón Anterior', 'Reinicia la canción actual si la reproducción ha superado la duración elegida.']
  },

  'lyrics.textSize': {
    en: ['Font size', 'Adjust text size in the lyrics panel.'],
    id: ['Ukuran huruf', 'Mengatur besar kecilnya huruf pada panel lirik.'],
    ms: ['Saiz fon', 'Laraskan saiz teks dalam panel lirik.'],
    ja: ['フォントサイズ', '歌詞パネルの文字サイズを調整します。'],
    ko: ['글꼴 크기', '가사 패널의 텍스트 크기를 조절합니다.'],
    hi: ['फ़ॉन्ट आकार', 'लिरिक्स पैनल में टेक्स्ट का आकार समायोजित करें।'],
    pt: ['Tamanho da fonte', 'Ajuste o tamanho do texto no painel de letras.'],
    de: ['Schriftgröße', 'Passe die Textgröße im Songtext-Panel an.'],
    fr: ['Taille de police', 'Ajustez la taille du texte dans le panneau des paroles.'],
    es: ['Tamaño de fuente', 'Ajusta el tamaño del texto en el panel de letras.']
  },
  'lyrics.autoScroll': {
    en: ['Auto-scroll lyrics', 'Automatically follow the active synced lyric line.'],
    id: ['Gulir otomatis lirik', 'Menggulir lirik tersinkronisasi secara otomatis mengikuti posisi lagu.'],
    ms: ['Tatal lirik automatik', 'Ikuti baris lirik segerak aktif secara automatik.'],
    ja: ['歌詞の自動スクロール', '現在再生中の同期歌詞の行を自動的に追従します。'],
    ko: ['가사 자동 스크롤', '현재 재생 중인 동기화된 가사 줄을 자동으로 따라갑니다.'],
    hi: ['लिरिक्स ऑटो-स्क्रॉल', 'सक्रिय सिंक किए गए लिरिक्स लाइन का स्वचालित रूप से अनुसरण करें।'],
    pt: ['Rolagem automática da letra', 'Segue automaticamente a linha ativa da letra sincronizada.'],
    de: ['Songtext automatisch scrollen', 'Folgt automatisch der aktiven synchronisierten Textzeile.'],
    fr: ['Défilement automatique des paroles', 'Suit automatiquement la ligne de paroles synchronisée active.'],
    es: ['Desplazamiento automático de letras', 'Sigue automáticamente la línea activa de la letra sincronizada.']
  },
  'lyrics.sweep': {
    en: ['Word-by-word sweep', 'Highlight lyrics smoothly word by word when timed data is available.'],
    id: ['Sapuan per kata', 'Sorot lirik per kata secara halus jika tersedia data waktu kata.'],
    ms: ['Sapuan perkataan demi perkataan', 'Serlahkan lirik secara lancar perkataan demi perkataan.'],
    ja: ['単語ごとのハイライト', 'タイミングデータがある場合に単語ごとに滑らかに強調表示します。'],
    ko: ['단어별 하이라이트', '타이밍 데이터가 있을 때 단어별로 부드럽게 강조합니다.'],
    hi: ['शब्द-दर-शब्द हाइलाइट', 'समय डेटा उपलब्ध होने पर शब्दों को सहजता से हाइलाइट करें।'],
    pt: ['Destaque palavra por palavra', 'Destaca a letra suavemente palavra por palavra quando disponível.'],
    de: ['Wort-für-Wort-Hervorhebung', 'Hebt den Songtext sanft Wort für Wort hervor.'],
    fr: ['Balayage mot par mot', 'Surligne les paroles mot par mot de manière fluide.'],
    es: ['Barrido palabra por palabra', 'Resalta la letra suavemente palabra por palabra.']
  },
  'lyrics.blurOthers': {
    en: ['Blur inactive lines', 'Applies in Full performance mode only.'],
    id: ['Kaburkan baris lain', 'Hanya berlaku di Kinerja Penuh.'],
    ms: ['Kaburkan baris tidak aktif', 'Berlaku dalam mod prestasi Penuh sahaja.'],
    ja: ['非アクティブ行をぼかす', 'フルパフォーマンスモードでのみ適用されます。'],
    ko: ['비활성 가사 줄 흐리게', '전체 성능 모드에서만 적용됩니다.'],
    hi: ['निष्क्रिय पंक्तियों को धुंधला करें', 'केवल पूर्ण प्रदर्शन मोड में लागू होता है।'],
    pt: ['Desfocar linhas inativas', 'Aplicado apenas no modo de desempenho Completo.'],
    de: ['Inaktive Zeilen weichzeichnen', 'Gilt nur im vollen Leistungsmodus.'],
    fr: ['Flouter les lignes inactives', 'S’applique uniquement en mode de performance Complet.'],
    es: ['Desenfocar líneas inactivas', 'Se aplica solo en el modo de rendimiento Completo.']
  },
  'lyrics.miniLyrics': {
    en: ['Small lyrics in mini mode', 'Show compact lyrics in the mini player view.'],
    id: ['Lirik kecil di mode mini', 'Tampilkan lirik ringkas pada tampilan pemutar mini.'],
    ms: ['Lirik kecil dalam mod mini', 'Tunjukkan lirik padat dalam paparan pemain mini.'],
    ja: ['ミニモードで歌詞を表示', 'ミニプレーヤー表示でコンパクトな歌詞を表示します。'],
    ko: ['미니 모드에서 작은 가사 표시', '미니 플레이어 보기에서 간결한 가사를 표시합니다.'],
    hi: ['मिनी मोड में छोटे बोल', 'मिनी प्लेयर दृश्य में संक्षिप्त बोल दिखाएं।'],
    pt: ['Letras compactas no modo mini', 'Mostra letras compactas na visualização do mini player.'],
    de: ['Kleine Songtexte im Mini-Modus', 'Zeigt kompakte Songtexte in der Mini-Player-Ansicht.'],
    fr: ['Petites paroles en mode mini', 'Affiche des paroles compactes dans le mini-lecteur.'],
    es: ['Letras pequeñas en modo mini', 'Muestra letras compactas en la vista del minirreproductor.']
  },
  'lyrics.nudgeMs': {
    en: ['Lyric timing nudge step', 'Step size for faster/slower lyric sync buttons.'],
    id: ['Langkah geser waktu lirik', 'Untuk tombol Lirik lebih cepat dan lebih lambat.'],
    ms: ['Langkah anjakan masa lirik', 'Saiz langkah untuk butang pelarasan masa lirik.'],
    ja: ['歌詞タイミング微調整ステップ', '歌詞のタイミング調整ボタンのステップ幅を設定します。'],
    ko: ['가사 싱크 미세 조정 단위', '가사 싱크 빠름/느림 버튼의 조정 단위를 설정합니다.'],
    hi: ['लिरिक्स टाइमिंग स्टेप', 'लिरिक्स सिंक तेज़/धीमा करने वाले बटन का स्टेप आकार।'],
    pt: ['Passo de ajuste de tempo da letra', 'Tamanho do passo para os botões de sincronização da letra.'],
    de: ['Schrittweite für Songtext-Timing', 'Schrittgröße für die Tasten zur Timing-Anpassung.'],
    fr: ['Pas de décalage des paroles', 'Valeur du pas pour les boutons de synchronisation des paroles.'],
    es: ['Paso de ajuste de tiempo de letra', 'Tamaño del paso para los botones de sincronización de letra.']
  },
  'lyrics.plainFallback': {
    en: ['Show plain lyrics', 'Show unsynced lyrics when synced lyrics are unavailable.'],
    id: ['Tampilkan lirik biasa', 'Lirik tanpa waktu, jika lirik berjalan tidak ada.'],
    ms: ['Tunjukkan lirik biasa', 'Tunjukkan lirik tanpa masa apabila lirik segerak tiada.'],
    ja: ['通常歌詞を表示', '同期歌詞がない場合に通常の歌詞を表示します。'],
    ko: ['일반 가사 표시', '동기화된 가사가 없을 때 일반 가사를 표시합니다.'],
    hi: ['सामान्य बोल दिखाएं', 'सिंक किए गए बोल उपलब्ध न होने पर सामान्य बोल दिखाएं।'],
    pt: ['Mostrar letra simples', 'Mostra letras sem sincronização quando a letra sincronizada não estiver disponível.'],
    de: ['Einfachen Songtext anzeigen', 'Zeigt nicht synchronisierten Text an, wenn kein synchronisierter verfügbar ist.'],
    fr: ['Afficher les paroles simples', 'Affiche les paroles non synchronisées si les paroles synchronisées sont indisponibles.'],
    es: ['Mostrar letra normal', 'Muestra letras sin sincronizar cuando no haya letras sincronizadas disponibles.']
  },

  'content.country': {
    en: ['Content region', 'Catalog regions for Home and Search recommendations.'],
    id: ['Negara konten', 'Wilayah katalog untuk rekomendasi Beranda dan Search.'],
    ms: ['Rantau kandungan', 'Rantau katalog untuk cadangan Laman Utama dan Carian.'],
    ja: ['コンテンツ地域', 'ホームや検索のおすすめに使用するカタログ地域。'],
    ko: ['콘텐츠 지역', '홈 및 검색 추천에 사용할 카탈로그 지역입니다.'],
    hi: ['सामग्री क्षेत्र', 'होम और खोज सुझावों के लिए कैटलॉग क्षेत्र।'],
    pt: ['Região de conteúdo', 'Regiões do catálogo para recomendações de Início e Busca.'],
    de: ['Inhaltsregion', 'Katalogregionen für Startseiten- und Suchempfehlungen.'],
    fr: ['Région du contenu', 'Régions du catalogue pour les recommandations Accueil et Recherche.'],
    es: ['Región de contenido', 'Regiones del catálogo para recomendaciones de Inicio y Búsqueda.']
  },
  'content.language': {
    en: ['App & Dynamic Island language', 'Applies across the entire Velora interface and Windows Dynamic Island. Defaults to your Windows locale (or English).'],
    id: ['Bahasa aplikasi & Dynamic Island', 'Berlaku di seluruh antarmuka Velora dan Dynamic Island Windows. Default mengikuti lokasi sistem Windows (atau English).'],
    ms: ['Bahasa aplikasi & Dynamic Island', 'Berlaku di seluruh antara muka Velora dan Dynamic Island Windows.'],
    ja: ['アプリとDynamic Islandの言語', 'Velora全体とWindows Dynamic Islandの表示言語を設定します。'],
    ko: ['앱 및 Dynamic Island 언어', 'Velora 인터페이스 전체와 Windows Dynamic Island에 적용됩니다.'],
    hi: ['ऐप और Dynamic Island भाषा', 'संपूर्ण Velora इंटरफ़ेस और Windows Dynamic Island पर लागू होता है।'],
    pt: ['Idioma do app e Dynamic Island', 'Aplicado em toda a interface do Velora e no Dynamic Island do Windows.'],
    de: ['App- & Dynamic Island-Sprache', 'Gilt für die gesamte Velora-Oberfläche und das Windows Dynamic Island.'],
    fr: ['Langue de l’application et Dynamic Island', 'S’applique à toute l’interface Velora et au Dynamic Island Windows.'],
    es: ['Idioma de la app y Dynamic Island', 'Se aplica en toda la interfaz de Velora y en Dynamic Island de Windows.']
  },
  'content.searchStart': {
    en: ['Default search filter', 'Initial tab selected when opening search results.'],
    id: ['Filter pencarian awal', 'Tab awal yang dipilih saat menampilkan hasil pencarian.'],
    ms: ['Penapis carian lalai', 'Tab awal yang dipilih apabila membuka hasil carian.'],
    ja: ['デフォルトの検索フィルター', '検索結果を開くときに最初に選択されるタブ。'],
    ko: ['기본 검색 필터', '검색 결과를 열 때 처음 선택되는 탭입니다.'],
    hi: ['डिफ़ॉल्ट खोज फ़िल्टर', 'खोज परिणाम खोलते समय चुना गया प्रारंभिक टैब।'],
    pt: ['Filtro de busca padrão', 'Aba inicial selecionada ao abrir os resultados de busca.'],
    de: ['Standard-Suchfilter', 'Anfänglicher Tab beim Öffnen der Suchergebnisse.'],
    fr: ['Filtre de recherche par défaut', 'Onglet initial sélectionné à l’ouverture des résultats de recherche.'],
    es: ['Filtro de búsqueda predeterminado', 'Pestaña inicial seleccionada al abrir los resultados de búsqueda.']
  },
  'content.hideExplicit': {
    en: ['Hide explicit songs', 'Filter explicit tracks from search results and auto-queued recommendations.'],
    id: ['Sembunyikan lagu eksplisit', 'Sembunyikan lagu eksplisit dari hasil pencarian dan antrean otomatis.'],
    ms: ['Sembunyikan lagu eksplisit', 'Tapis lagu eksplisit daripada hasil carian dan cadangan automatik.'],
    ja: ['露骨な表現の曲を隠す', '検索結果や自動おすすめからExplicit（露骨な表現）の曲を除外します。'],
    ko: ['성인용(Explicit) 곡 숨기기', '검색 결과 및 자동 추천 대기열에서 Explicit 곡을 필터링합니다.'],
    hi: ['एक्सप्लिसिट गाने छिपाएं', 'खोज परिणामों और ऑटो-कतार सुझावों से एक्सप्लिसिट ट्रैक फ़िल्टर करें।'],
    pt: ['Ocultar músicas explícitas', 'Filtra faixas explícitas dos resultados de busca e recomendações automáticas.'],
    de: ['Explizite Songs ausblenden', 'Filtert explizite Titel aus Suchergebnissen und automatischen Empfehlungen.'],
    fr: ['Masquer les titres explicites', 'Filtre les titres explicites des résultats de recherche et des recommandations.'],
    es: ['Ocultar canciones explícitas', 'Filtra pistas explícitas de los resultados de búsqueda y recomendaciones automáticas.']
  },

  'storage.clearCache': {
    en: ['Clear audio and artwork cache', 'Delete temporary streamed files and cover art to free disk space.'],
    id: ['Bersihkan cache lagu dan sampul', 'Hapus berkas sementara hasil streaming dan gambar sampul untuk menghemat ruang penyimpanan.'],
    ms: ['Kosongkan cache audio dan sampul', 'Padamkan fail strim sementara dan gambar sampul untuk menjimatkan ruang.'],
    ja: ['音声とカバー画像のキャッシュを消去', '一時的なストリーミングファイルとカバー画像を削除して空き容量を増やします。'],
    ko: ['오디오 및 아트워크 캐시 비우기', '임시 스트리밍 파일과 커버 아트를 삭제하여 디스크 공간을 확보합니다.'],
    hi: ['ऑडियो और आर्टवर्क कैश साफ़ करें', 'डिस्क स्थान खाली करने के लिए अस्थायी स्ट्रीम फ़ाइलें और कवर आर्ट हटाएं।'],
    pt: ['Limpar cache de áudio e capas', 'Exclui arquivos temporários de streaming e capas para liberar espaço.'],
    de: ['Audio- und Cover-Cache leeren', 'Löscht temporäre Streaming-Dateien und Coverbilder, um Speicherplatz freizugeben.'],
    fr: ['Vider le cache audio et pochettes', 'Supprime les fichiers de flux temporaires et les pochettes pour libérer de l’espace.'],
    es: ['Borrar caché de audio y portadas', 'Elimina archivos temporales de transmisión y portadas para liberar espacio.']
  },
  'storage.cacheLimit': {
    en: ['Cache limit', 'Maximum number of recent songs kept in cache (up to ~1 GB).'],
    id: ['Batas cache', 'Jumlah lagu terakhir yang disimpan (paling banyak sekitar 1 GB).'],
    ms: ['Had cache', 'Bilangan maksimum lagu terkini yang disimpan dalam cache (sehingga ~1 GB).'],
    ja: ['キャッシュ上限', 'キャッシュに保持する最近の曲の最大数（最大約1GB）。'],
    ko: ['캐시 한도', '캐시에 보관할 최근 노래의 최대 개수(최대 약 1GB)입니다.'],
    hi: ['कैश सीमा', 'कैश में रखे जाने वाले हाल के गानों की अधिकतम संख्या (~1 GB तक)।'],
    pt: ['Limite de cache', 'Número máximo de músicas recentes mantidas em cache (até ~1 GB).'],
    de: ['Cache-Limit', 'Maximale Anzahl kürzlich gespielter Songs im Cache (bis ca. 1 GB).'],
    fr: ['Limite du cache', 'Nombre maximal de titres récents conservés en cache (jusqu’à ~1 Go).'],
    es: ['Límite de caché', 'Número máximo de canciones recientes guardadas en caché (hasta ~1 GB).']
  },
  'storage.autoDownloadOnLike': {
    en: ['Download liked songs', 'Automatically save liked songs to your downloads folder for offline playback.'],
    id: ['Unduh lagu yang disukai', 'Lagu yang kamu sukai ikut disimpan di folder unduhan agar bisa diputar tanpa internet.'],
    ms: ['Muat turun lagu yang disukai', 'Simpan lagu yang disukai secara automatik untuk main balik luar talian.'],
    ja: ['お気に入りの曲を自動ダウンロード', '「いいね」した曲をオフライン再生用に自動的にダウンロード保存します。'],
    ko: ['좋아요 표시한 곡 자동 다운로드', '좋아요를 누른 곡을 오프라인 재생을 위해 자동으로 다운로드합니다.'],
    hi: ['पसंद किए गए गाने डाउनलोड करें', 'ऑफ़लाइन प्लेबैक के लिए पसंद किए गए गानों को स्वचालित रूप से सहेजें।'],
    pt: ['Baixar músicas curtidas', 'Salva automaticamente as músicas curtidas para reprodução offline.'],
    de: ['Lieblingssongs herunterladen', 'Speichert gelikte Songs automatisch für die Offline-Wiedergabe.'],
    fr: ['Télécharger les titres aimés', 'Enregistre automatiquement les titres aimés pour une écoute hors ligne.'],
    es: ['Descargar canciones que te gustan', 'Guarda automáticamente las canciones que te gustan para escucharlas sin conexión.']
  },
  'storage.clearCacheOnExit': {
    en: ['Clear song cache on exit', 'Downloaded songs are kept.'],
    id: ['Bersihkan cache lagu saat keluar', 'Unduhan tidak ikut dihapus.'],
    ms: ['Kosongkan cache lagu semasa keluar', 'Lagu yang dimuat turun akan dikekalkan.'],
    ja: ['終了時に曲キャッシュを消去', 'ダウンロード済みの曲は保持されます。'],
    ko: ['종료 시 노래 캐시 비우기', '다운로드한 오프라인 곡은 유지됩니다.'],
    hi: ['बंद करते समय गाना कैश साफ़ करें', 'डाउनलोड किए गए गाने सुरक्षित रहते हैं।'],
    pt: ['Limpar cache de músicas ao sair', 'As músicas baixadas são mantidas.'],
    de: ['Song-Cache beim Beenden leeren', 'Heruntergeladene Songs bleiben erhalten.'],
    fr: ['Vider le cache des titres à la fermeture', 'Les titres téléchargés sont conservés.'],
    es: ['Borrar caché de canciones al salir', 'Las canciones descargadas se conservan.']
  },

  'history.enabled': {
    en: ['Save listening history', 'Used for Recently Played and Stats. Stored locally on this PC only.'],
    id: ['Simpan riwayat dengar', 'Dipakai untuk Terakhir diputar dan Statistik. Hanya tersimpan di komputer ini.'],
    ms: ['Simpan sejarah pendengaran', 'Digunakan untuk Baru Dimainkan dan Statistik. Disimpan secara tempatan sahaja.'],
    ja: ['再生履歴を保存', '最近再生した曲や統計に使用されます。このPC内にのみ保存されます。'],
    ko: ['청취 기록 저장', '최근 재생 및 통계에 사용됩니다. 이 PC에만 로컬로 저장됩니다.'],
    hi: ['सुनने का इतिहास सहेजें', 'हाल ही में चलाए गए और आँकड़ों के लिए उपयोग किया जाता है। केवल इस पीसी पर सहेजा जाता है।'],
    pt: ['Salvar histórico de reprodução', 'Usado para Tocadas Recentemente e Estatísticas. Armazenado apenas neste PC.'],
    de: ['Hörverlauf speichern', 'Wird für Kürzlich gespielt und Statistiken verwendet. Nur lokal auf diesem PC gespeichert.'],
    fr: ['Enregistrer l’historique d’écoute', 'Utilisé pour Écoutés récemment et les statistiques. Stocké localement sur ce PC.'],
    es: ['Guardar historial de escucha', 'Se usa para Escuchado recientemente y Estadísticas. Se guarda solo en esta PC.']
  },
  'privacy.saveSearches': {
    en: ['Save search history', 'Record recent search queries locally for quick access.'],
    id: ['Simpan riwayat pencarian', 'Simpan kata kunci pencarian terbaru secara lokal agar mudah diakses lagi.'],
    ms: ['Simpan sejarah carian', 'Rekod kata kunci carian terkini secara tempatan untuk akses pantas.'],
    ja: ['検索履歴を保存', '最近の検索キーワードをローカルに保存して素早くアクセスできるようにします。'],
    ko: ['검색 기록 저장', '빠른 접근을 위해 최근 검색어를 로컬에 기록합니다.'],
    hi: ['खोज इतिहास सहेजें', 'त्वरित पहुँच के लिए हाल की खोजों को स्थानीय रूप से सहेजें।'],
    pt: ['Salvar histórico de busca', 'Registra buscas recentes localmente para acesso rápido.'],
    de: ['Suchverlauf speichern', 'Speichert letzte Suchanfragen lokal für schnellen Zugriff.'],
    fr: ['Enregistrer l’historique de recherche', 'Enregistre localement les recherches récentes pour un accès rapide.'],
    es: ['Guardar historial de búsqueda', 'Guarda las búsquedas recientes localmente para un acceso rápido.']
  },
  'privacy.historyDays': {
    en: ['Auto-delete listening history', 'Older records are removed automatically. Streak days remain intact.'],
    id: ['Hapus riwayat dengar otomatis', 'Catatan yang lebih lama dihapus. Hari-hari streak tetap ada.'],
    ms: ['Padam sejarah pendengaran automatik', 'Rekod lama dipadamkan secara automatik. Hari streak kekal utuh.'],
    ja: ['再生履歴の自動削除', '古い記録は自動的に削除されます。ストリーク日수는維持されます。'],
    ko: ['청취 기록 자동 삭제', '오래된 기록은 자동으로 삭제됩니다. 스트릭 일수는 유지됩니다.'],
    hi: ['सुनने का इतिहास स्वतः हटाएं', 'पुराने रिकॉर्ड स्वचालित रूप से हट जाते हैं। स्ट्रीक दिन सुरक्षित रहते हैं।'],
    pt: ['Excluir histórico automaticamente', 'Registros antigos são removidos automaticamente. Os dias de sequência permanecem.'],
    de: ['Hörverlauf automatisch löschen', 'Ältere Einträge werden automatisch entfernt. Serien-Tage bleiben erhalten.'],
    fr: ['Suppression auto de l’historique', 'Les anciens enregistrements sont supprimés automatiquement. Les jours de série restent intacts.'],
    es: ['Eliminar historial automáticamente', 'Los registros antiguos se eliminan automáticamente. Los días de racha se mantienen.']
  },

  'integrations.jam.enabled': {
    en: ['Allow Velora Jam (Wi-Fi remote)', 'Grant permission to use the Velora Jam button in the top bar to turn Wi-Fi remote control on or off.'],
    id: ['Izinkan fitur Velora Jam', 'Izinkan penggunaan tombol Velora Jam di bilah atas untuk menyalakan atau mematikan kontrol Wi-Fi.'],
    ms: ['Benarkan fungsi Velora Jam', 'Berikan kebenaran untuk menghidupkan atau mematikan Velora Jam dari bar atas.'],
    ja: ['Velora Jam（Wi-Fiリモコン）を許可', '上部バーのVelora JamボタンからWi-Fiリモート制御をオン/オフすることを許可します。'],
    ko: ['Velora Jam(Wi-Fi 리모컨) 허용', '상단 바의 Velora Jam 버튼으로 Wi-Fi 원격 제어를 켜거나 끌 수 있도록 허용합니다.'],
    hi: ['Velora Jam (Wi-Fi रिमोट) की अनुमति दें', 'शीर्ष बार में Velora Jam बटन से Wi-Fi रिमोट चालू या बंद करने की अनुमति दें।'],
    pt: ['Permitir Velora Jam (controle Wi-Fi)', 'Permite ligar ou desligar o controle remoto Wi-Fi pelo botão Velora Jam na barra superior.'],
    de: ['Velora Jam (WLAN-Fernbedienung) erlauben', 'Erlaubt das Ein- und Ausschalten der WLAN-Fernsteuerung über die obere Leiste.'],
    fr: ['Autoriser Velora Jam (télécommande Wi-Fi)', 'Autorise l’activation ou la désactivation du contrôle Wi-Fi depuis la barre supérieure.'],
    es: ['Permitir Velora Jam (control Wi-Fi)', 'Permite activar o desactivar el control remoto Wi-Fi desde el botón Velora Jam en la barra superior.']
  },
  'integrations.jam.remoteUrl': {
    en: ['Remote address', 'Open this address in another device browser to control playback.'],
    id: ['Alamat remote', 'Buka alamat ini di browser perangkat lain untuk mengendalikan pemutaran.'],
    ms: ['Alamat remote', 'Buka alamat ini dalam pelayar peranti lain untuk mengawal main balik.'],
    ja: ['リモートアドレス', '同じWi-Fi内の別デバイスのブラウザでこのアドレスを開いて操作します。'],
    ko: ['원격 주소', '다른 기기의 브라우저에서 이 주소를 열어 재생을 제어하세요.'],
    hi: ['रिमोट पता', 'प्लेबैक नियंत्रित करने के लिए किसी अन्य डिवाइस के ब्राउज़र में यह पता खोलें।'],
    pt: ['Endereço remoto', 'Abra este endereço no navegador de outro dispositivo para controlar a reprodução.'],
    de: ['Remote-Adresse', 'Öffne diese Adresse im Browser eines anderen Geräts zur Steuerung.'],
    fr: ['Adresse distante', 'Ouvrez cette adresse dans le navigateur d’un autre appareil pour contrôler la lecture.'],
    es: ['Dirección remota', 'Abre esta dirección en el navegador de otro dispositivo para controlar la reproducción.']
  },

  'system.mediaKeys': {
    en: ['Media keys', 'Play, pause, next, and previous keys on keyboard or headset work from any window.'],
    id: ['Tombol media', 'Putar, jeda, berikutnya, dan sebelumnya di keyboard atau headset bekerja dari jendela mana pun.'],
    ms: ['Kekunci media', 'Kekunci main, jeda, seterusnya dan sebelumnya berfungsi dari mana-mana tetingkap.'],
    ja: ['メディアキー', 'キーボードやヘッドセットの再生・一時停止・次/前キーがどのウィンドウからでも機能します。'],
    ko: ['미디어 키', '키보드나 헤드셋의 재생/일시정지/다음/이전 키가 모든 창에서 작동합니다.'],
    hi: ['मीडिया कुंजियाँ', 'कीबोर्ड या हेडसेट की प्ले, पॉज़, नेक्स्ट और प्रीवियस कुंजियाँ किसी भी विंडो से काम करती हैं।'],
    pt: ['Teclas de mídia', 'As teclas de reproduzir, pausar, próxima e anterior funcionam de qualquer janela.'],
    de: ['Medientasten', 'Wiedergabe-, Pause-, Vor- und Zurück-Tasten funktionieren aus jedem Fenster.'],
    fr: ['Touches multimédias', 'Les touches lecture, pause, suivant et précédent fonctionnent depuis n’importe quelle fenêtre.'],
    es: ['Teclas multimedia', 'Las teclas de reproducir, pausar, siguiente y anterior funcionan desde cualquier ventana.']
  },
  'system.closeToTray': {
    en: ['Hide to System Tray on close', 'Keep music playing in the background when the window is closed.'],
    id: ['Sembunyikan ke System Tray saat ditutup', 'Tetap putar musik di latar belakang saat jendela ditutup.'],
    ms: ['Sembunyikan ke Dulang Sistem apabila ditutup', 'Pastikan muzik terus dimainkan di latar belakang apabila tetingkap ditutup.'],
    ja: ['閉じるときにシステムトレイに隠す', 'ウィンドウを閉じてもバックグラウンドで音楽の再生を続けます。'],
    ko: ['닫을 때 시스템 트레이로 숨기기', '창을 닫아도 백그라운드에서 음악 재생을 계속 유지합니다.'],
    hi: ['बंद करने पर सिस्टम ट्रे में छिपाएं', 'विंडो बंद होने पर भी बैकग्राउंड में संगीत चालू रखें।'],
    pt: ['Ocultar na Bandeja do Sistema ao fechar', 'Mantém a música tocando em segundo plano quando a janela é fechada.'],
    de: ['Beim Schließen in den System-Tray minimieren', 'Spielt Musik im Hintergrund weiter, wenn das Fenster geschlossen wird.'],
    fr: ['Masquer dans la barre système à la fermeture', 'Continue la lecture de la musique en arrière-plan lorsque la fenêtre est fermée.'],
    es: ['Ocultar en la bandeja del sistema al cerrar', 'Mantiene la música sonando en segundo plano al cerrar la ventana.']
  },
  'system.keepAwake': {
    en: ['Keep PC awake while playing', 'Display may still turn off.'],
    id: ['Jangan biarkan PC tidur saat musik diputar', 'Layar tetap boleh mati.'],
    ms: ['Jangan biarkan PC tidur semasa muzik dimainkan', 'Paparan skrin masih boleh dimatikan.'],
    ja: ['再生中はPCのスリープを防止', 'ディスプレイの電源は通常通りオフになる場合があります。'],
    ko: ['재생 중 PC 절전 모드 방지', '화면은 계속 꺼질 수 있습니다.'],
    hi: ['संगीत चलते समय पीसी को स्लीप मोड से रोकें', 'डिस्प्ले फिर भी बंद हो सकता है।'],
    pt: ['Manter PC acordado durante a reprodução', 'A tela ainda pode desligar normalmente.'],
    de: ['PC während der Wiedergabe wach halten', 'Der Bildschirm kann sich weiterhin ausschalten.'],
    fr: ['Empêcher la mise en veille pendant la lecture', 'L’écran peut toujours s’éteindre.'],
    es: ['Mantener la PC activa mientras se reproduce', 'La pantalla aún puede apagarse.']
  },
  'system.hideTitle': {
    en: ['Hide song title in window bar', 'Window title stays Velora while playing.'],
    id: ['Sembunyikan judul lagu di jendela', 'Judul jendela tetap Velora saat lagu diputar.'],
    ms: ['Sembunyikan tajuk lagu di bar tetingkap', 'Tajuk tetingkap kekal Velora semasa lagu dimainkan.'],
    ja: ['ウィンドウバーの曲名を隠す', '再生中もウィンドウタイトルを「Velora」のままにします。'],
    ko: ['창 제목 표시줄에서 곡 제목 숨기기', '재생 중에도 창 제목을 Velora로 유지합니다.'],
    hi: ['विंडो बार में गाने का शीर्षक छिपाएं', 'गाना बजते समय भी विंडो का शीर्षक Velora ही रहेगा।'],
    pt: ['Ocultar título da música na barra da janela', 'O título da janela permanece Velora durante a reprodução.'],
    de: ['Songtitel in der Fenstertitelleiste ausblenden', 'Der Fenstertitel bleibt während der Wiedergabe „Velora“.'],
    fr: ['Masquer le titre dans la barre de fenêtre', 'Le titre de la fenêtre reste Velora pendant la lecture.'],
    es: ['Ocultar título de canción en la barra de ventana', 'El título de la ventana permanece como Velora durante la reproducción.']
  },

  'dynamicIsland.enabled': {
    en: ['Dynamic Island', 'Compact player at the top edge of your screen to view songs and lyrics without opening Velora.'],
    id: ['Dynamic Island', 'Pemutar kecil di tepi atas layar, untuk melihat lagu dan lirik tanpa membuka Velora.'],
    ms: ['Dynamic Island', 'Pemain padat di tepi atas skrin untuk melihat lagu dan lirik tanpa membuka Velora.'],
    ja: ['Dynamic Island', 'Veloraを開かずに画面上端で曲や歌詞を確認できるコンパクトプレーヤー。'],
    ko: ['Dynamic Island', 'Velora를 열지 않고도 화면 상단에서 곡과 가사를 볼 수 있는 컴팩트 플레이어입니다.'],
    hi: ['Dynamic Island', 'Velora खोले बिना गाने और बोल देखने के लिए स्क्रीन के शीर्ष किनारे पर कॉम्पैक्ट प्लेयर।'],
    pt: ['Dynamic Island', 'Player compacto na borda superior da tela para ver músicas e letras sem abrir o Velora.'],
    de: ['Dynamic Island', 'Kompakter Player am oberen Bildschirmrand für Songs und Songtexte.'],
    fr: ['Dynamic Island', 'Lecteur compact en haut de l’écran pour voir les titres et paroles sans ouvrir Velora.'],
    es: ['Dynamic Island', 'Reproductor compacto en el borde superior de la pantalla para ver canciones y letras.']
  },
  'dynamicIsland.style': {
    en: ['Position', 'Choose docked at screen edge/notch or floating pill.'],
    id: ['Posisi', 'Pilih tampilan menempel di tepi atas layar atau melayang berbentuk pil.'],
    ms: ['Kedudukan', 'Pilih menempel di tepi atas skrin atau pil terapung.'],
    ja: ['位置スタイル', '画面上端にドックするか、フローティングピルにするかを選択します。'],
    ko: ['위치 스타일', '화면 상단 고정형 또는 플로팅 알약 형태를 선택합니다.'],
    hi: ['स्थिति', 'स्क्रीन के किनारे डॉक किया हुआ या फ़्लोटिंग पिल चुनें।'],
    pt: ['Posição', 'Escolha fixo na borda superior da tela ou pílula flutuante.'],
    de: ['Position', 'Wähle am oberen Bildschirmrand angedockt oder schwebende Pille.'],
    fr: ['Position', 'Choisissez ancré au bord supérieur de l’écran ou pilule flottante.'],
    es: ['Posición', 'Elige anclado al borde superior de la pantalla o píldora flotante.']
  },
  'dynamicIsland.compactStyle': {
    en: ['Closed style', 'Default shape when Dynamic Island is closed (Hidden shrinks into a top-edge line unless active lyrics playback is running).'],
    id: ['Gaya saat tertutup', 'Bentuk Dynamic Island saat tertutup (Tersembunyi mengecil ke garis tepi atas, kecuali saat mode playback lirik aktif).'],
    ms: ['Gaya semasa tertutup', 'Bentuk Dynamic Island semasa tertutup (Tersembunyi mengecil ke garis tepi atas melainkan mod main balik lirik aktif).'],
    ja: ['閉じたときのスタイル', '閉じているときのDynamic Islandの形状（「非表示」は上端の細いラインに縮小しますが、歌詞再生モード中は表示を維持します）。'],
    ko: ['닫혔을 때 스타일', '닫혀 있을 때 Dynamic Island 모양(숨김은 상단 얇은 선으로 축소되지만 가사 재생 모드에서는 유지됩니다).'],
    hi: ['बंद होने पर शैली', 'बंद होने पर Dynamic Island का आकार (सक्रिय लिरिक्स प्लेबैक के अलावा छिपा हुआ ऊपरी रेखा में सिमट जाता है)।'],
    pt: ['Estilo quando fechado', 'Formato quando fechado (Oculto encolhe para uma linha superior, exceto durante a reprodução ativa de letras).'],
    de: ['Stil im geschlossenen Zustand', 'Form im geschlossenen Zustand (Ausgeblendet schrumpft zur Linie oben, außer bei aktiver Liedtext-Wiedergabe).'],
    fr: ['Style une fois fermé', 'Forme une fois fermé (Masqué se réduit en ligne supérieure, sauf en mode lecture des paroles active).'],
    es: ['Estilo al cerrarse', 'Forma al cerrarse (Oculto se reduce a una línea superior salvo en reproducción activa de letras).']
  },
  'dynamicIsland.size': {
    en: ['Size', 'Overall Dynamic Island scale preset.'],
    id: ['Ukuran', 'Preset ukuran Dynamic Island.'],
    ms: ['Saiz', 'Pratetap skala keseluruhan Dynamic Island.'],
    ja: ['サイズ', 'Dynamic Island全体のサイズプリセット。'],
    ko: ['크기', 'Dynamic Island 전체 크기 프리셋입니다.'],
    hi: ['आकार', 'समग्र Dynamic Island आकार प्रीसेट।'],
    pt: ['Tamanho', 'Predefinição de escala geral do Dynamic Island.'],
    de: ['Größe', 'Skalierungsvoreinstellung für das Dynamic Island.'],
    fr: ['Taille', 'Préréglage de la taille globale du Dynamic Island.'],
    es: ['Tamaño', 'Preajuste de escala general de Dynamic Island.']
  },
  'dynamicIsland.opacity': {
    en: ['Opacity', 'Below 100%, the screen behind becomes slightly visible.'],
    id: ['Kepekatan', 'Di bawah 100% layar di belakang terlihat sedikit.'],
    ms: ['Kelegapan', 'Di bawah 100%, skrin di belakang kelihatan sedikit.'],
    ja: ['不透明度', '100%未満にすると背後の画面が少し透けて見えます。'],
    ko: ['불투명도', '100% 미만으로 설정하면 뒤쪽 화면이 살짝 비칩니다.'],
    hi: ['अपारदर्शिता (Opacity)', '100% से कम होने पर पीछे की स्क्रीन हल्की दिखाई देती है।'],
    pt: ['Opacidade', 'Abaixo de 100%, a tela atrás fica levemente visível.'],
    de: ['Deckkraft', 'Unter 100 % scheint der Hintergrund leicht durch.'],
    fr: ['Opacité', 'En dessous de 100 %, l’écran derrière devient légèrement visible.'],
    es: ['Opacidad', 'Por debajo del 100%, la pantalla detrás se vuelve ligeramente visible.']
  },
  'dynamicIsland.showStatus': {
    en: ['Clock & system status', 'Display clock and Wi-Fi, GPS, Bluetooth icons at the top.'],
    id: ['Jam & status sistem', 'Tampilkan jam serta ikon Wi-Fi, GPS, dan Bluetooth di baris atas.'],
    ms: ['Jam & status sistem', 'Paparkan jam serta ikon Wi-Fi, GPS, dan Bluetooth di bahagian atas.'],
    ja: ['時計とシステム状態', '上部に時計とWi-Fi、GPS、Bluetoothアイコンを表示します。'],
    ko: ['시계 및 시스템 상태', '상단에 시계와 Wi-Fi, GPS, Bluetooth 아이콘을 표시합니다.'],
    hi: ['घड़ी और सिस्टम स्थिति', 'शीर्ष पर घड़ी और Wi-Fi, GPS, Bluetooth आइकन दिखाएं।'],
    pt: ['Relógio e status do sistema', 'Exibe o relógio e ícones de Wi-Fi, GPS e Bluetooth no topo.'],
    de: ['Uhr & Systemstatus', 'Zeigt Uhrzeit sowie WLAN-, GPS- und Bluetooth-Symbole oben an.'],
    fr: ['Horloge et état du système', 'Affiche l’horloge et les icônes Wi-Fi, GPS et Bluetooth en haut.'],
    es: ['Reloj y estado del sistema', 'Muestra el reloj y los iconos de Wi-Fi, GPS y Bluetooth en la parte superior.']
  },
  'dynamicIsland.showCover': {
    en: ['Show cover art', 'Display album artwork inside Dynamic Island.'],
    id: ['Sampul album', 'Tampilkan sampul album di Dynamic Island.'],
    ms: ['Sampul album', 'Paparkan karya seni album di dalam Dynamic Island.'],
    ja: ['アルバムカバーを表示', 'Dynamic Island内にアルバムアートワークを表示します。'],
    ko: ['앨범 커버 표시', 'Dynamic Island 내부에 앨범 아트워크를 표시합니다.'],
    hi: ['कवर आर्ट दिखाएं', 'Dynamic Island के अंदर एल्बम आर्टवर्क दिखाएं।'],
    pt: ['Mostrar capa do álbum', 'Exibe a capa do álbum dentro do Dynamic Island.'],
    de: ['Albumcover anzeigen', 'Zeigt das Albumcover im Dynamic Island an.'],
    fr: ['Afficher la pochette', 'Affiche la pochette d’album dans le Dynamic Island.'],
    es: ['Mostrar portada del álbum', 'Muestra la portada del álbum dentro de Dynamic Island.']
  },
  'dynamicIsland.coverStyle': {
    en: ['Album art style', 'Choose square or circular cover art inside Dynamic Island.'],
    id: ['Gaya gambar album', 'Pilih bentuk gambar album kotak atau lingkaran di Dynamic Island.'],
    ms: ['Gaya gambar album', 'Pilih bentuk gambar album persegi atau bulatan di Dynamic Island.'],
    ja: ['アルバム画像のスタイル', 'Dynamic Island内のカバー形状を四角または円形から選択します。'],
    ko: ['앨범 이미지 스타일', 'Dynamic Island 내부 커버 모양을 사각형 또는 원형으로 선택합니다.'],
    hi: ['एल्बम आर्ट स्टाइल', 'Dynamic Island में चौकोर या गोलाकार कवर आर्ट चुनें।'],
    pt: ['Estilo da capa do álbum', 'Escolha capa quadrada ou circular dentro do Dynamic Island.'],
    de: ['Albumcover-Stil', 'Wähle quadratisches oder rundes Albumcover im Dynamic Island.'],
    fr: ['Style de la pochette', 'Choisissez une pochette carrée ou circulaire dans le Dynamic Island.'],
    es: ['Estilo de portada', 'Elige portada cuadrada o circular dentro de Dynamic Island.']
  },
  'dynamicIsland.coverSpin': {
    en: ['Rotate circular album art', 'Spin circular album artwork while music is playing.'],
    id: ['Putar gambar album lingkaran', 'Putar gambar album berbentuk lingkaran saat lagu diputar.'],
    ms: ['Putar gambar album bulatan', 'Putar gambar album berbentuk bulatan semasa muzik dimainkan.'],
    ja: ['円形アルバム画像を回転', '音楽の再生中に円形のアルバムアートを回転させます。'],
    ko: ['원형 앨범 이미지 회전', '음악이 재생되는 동안 원형 앨범 아트를 회전합니다.'],
    hi: ['गोलाकार एल्बम आर्ट घुमाएं', 'संगीत बजते समय गोलाकार एल्बम आर्टवर्क घुमाएं।'],
    pt: ['Girar capa circular', 'Gira a capa circular do álbum enquanto a música toca.'],
    de: ['Rundes Albumcover drehen', 'Dreht das runde Albumcover während der Wiedergabe.'],
    fr: ['Faire tourner la pochette circulaire', 'Fait tourner la pochette circulaire pendant la lecture.'],
    es: ['Girar portada circular', 'Gira la portada circular mientras se reproduce la música.']
  },
  'dynamicIsland.showClosedCover': {
    en: ['Show cover when closed', 'Display small album art on the left side when Dynamic Island is closed.'],
    id: ['Sampul saat tertutup', 'Tampilkan gambar album kecil di sisi kiri saat Dynamic Island tertutup.'],
    ms: ['Sampul semasa tertutup', 'Paparkan gambar album kecil di sebelah kiri apabila Dynamic Island tertutup.'],
    ja: ['閉じた状態のカバー表示', 'Dynamic Islandが閉じているときに左側に小さなアルバムアートを表示します。'],
    ko: ['닫혔을 때 커버 표시', 'Dynamic Island가 닫혀 있을 때 왼쪽에 작은 앨범 아트를 표시합니다.'],
    hi: ['बंद होने पर कवर दिखाएं', 'Dynamic Island बंद होने पर बाईं ओर छोटा एल्बम आर्ट दिखाएं।'],
    pt: ['Mostrar capa quando fechado', 'Exibe uma pequena capa à esquerda quando o Dynamic Island está fechado.'],
    de: ['Cover im geschlossenen Zustand', 'Zeigt links ein kleines Albumcover, wenn das Dynamic Island geschlossen ist.'],
    fr: ['Pochette lorsque fermé', 'Affiche une petite pochette à gauche lorsque le Dynamic Island est fermé.'],
    es: ['Mostrar portada cuando está cerrado', 'Muestra una pequeña portada a la izquierda cuando Dynamic Island está cerrado.']
  },
  'dynamicIsland.closedCoverSpin': {
    en: ['Rotate cover when closed', 'Spin small song artwork when Dynamic Island is closed and playing.'],
    id: ['Putar gambar saat tertutup', 'Putar gambar lagu kecil saat Dynamic Island tertutup dan memutar musik.'],
    ms: ['Putar gambar semasa tertutup', 'Putar gambar lagu kecil apabila Dynamic Island tertutup dan sedang bermain.'],
    ja: ['閉じた状態のカバーを回転', 'Dynamic Islandが閉じて再生中のときに小さなカバー画像を回転させます。'],
    ko: ['닫혔을 때 커버 회전', 'Dynamic Island가 닫힌 상태에서 재생 중일 때 작은 커버를 회전합니다.'],
    hi: ['बंद होने पर कवर घुमाएं', 'Dynamic Island बंद होने और संगीत बजने पर छोटा कवर आर्ट घुमाएं।'],
    pt: ['Girar capa quando fechado', 'Gira a pequena capa da música quando o Dynamic Island está fechado e tocando.'],
    de: ['Cover im geschlossenen Zustand drehen', 'Dreht das kleine Cover, wenn das Dynamic Island geschlossen ist und Musik läuft.'],
    fr: ['Faire tourner la pochette fermée', 'Fait tourner la petite pochette lorsque le Dynamic Island est fermé et en lecture.'],
    es: ['Girar portada cuando está cerrado', 'Gira la pequeña portada cuando Dynamic Island está cerrado y reproduciendo.']
  },
  'dynamicIsland.showArtist': {
    en: ['Show artist name', 'Display artist subtitle below track title.'],
    id: ['Nama artis', 'Tampilkan nama artis di bawah judul lagu.'],
    ms: ['Nama artis', 'Paparkan sari kata artis di bawah tajuk lagu.'],
    ja: ['アーティスト名を表示', '曲名の下にアーティスト名を表示します。'],
    ko: ['아티스트 이름 표시', '곡 제목 아래에 아티스트 이름을 표시합니다.'],
    hi: ['कलाकार का नाम दिखाएं', 'गाने के शीर्षक के नीचे कलाकार का नाम दिखाएं।'],
    pt: ['Mostrar nome do artista', 'Exibe o nome do artista abaixo do título da música.'],
    de: ['Künstlernamen anzeigen', 'Zeigt den Künstlernamen unter dem Songtitel an.'],
    fr: ['Afficher le nom de l’artiste', 'Affiche le nom de l’artiste sous le titre.'],
    es: ['Mostrar nombre del artista', 'Muestra el nombre del artista debajo del título de la canción.']
  },
  'dynamicIsland.showLyric': {
    en: ['Lyric', 'Lyric button and active lyric line in one-line pill.'],
    id: ['Lirik', 'Tombol lirik dan baris lirik aktif pada pil satu baris.'],
    ms: ['Lirik', 'Butang lirik dan baris lirik aktif pada pil satu baris.'],
    ja: ['歌詞', '歌詞ボタンおよび1行ピルでの現在の歌詞行を表示します。'],
    ko: ['가사', '가사 버튼과 한 줄 모드에서의 현재 가사를 표시합니다.'],
    hi: ['बोल (Lyric)', 'लिरिक बटन और एक-पंक्ति पिल में सक्रिय लिरिक लाइन।'],
    pt: ['Letra', 'Botão de letra e linha de letra ativa na pílula de uma linha.'],
    de: ['Songtext', 'Songtext-Taste und aktive Textzeile in der einzeiligen Ansicht.'],
    fr: ['Paroles', 'Bouton des paroles et ligne active dans la pilule sur une ligne.'],
    es: ['Letra', 'Botón de letra y línea de letra activa en la píldora de una línea.']
  },
  'dynamicIsland.showHeart': {
    en: ['Show heart button', 'Quick favorite toggle inside Dynamic Island.'],
    id: ['Tombol hati', 'Tombol suka cepat di dalam Dynamic Island.'],
    ms: ['Tombol hati', 'Togol kegemaran pantas di dalam Dynamic Island.'],
    ja: ['ハートボタンを表示', 'Dynamic Island内でお気に入りを素早く切り替えます。'],
    ko: ['하트 버튼 표시', 'Dynamic Island 내부에 빠른 좋아요 토글을 표시합니다.'],
    hi: ['हार्ट बटन दिखाएं', 'Dynamic Island के अंदर त्वरित पसंदीदा टॉगल।'],
    pt: ['Mostrar botão de coração', 'Alternância rápida de favoritos dentro do Dynamic Island.'],
    de: ['Herz-Taste anzeigen', 'Schnelles Favoriten-Umschalten im Dynamic Island.'],
    fr: ['Afficher le bouton cœur', 'Bascule rapide des favoris dans le Dynamic Island.'],
    es: ['Mostrar botón de corazón', 'Alternancia rápida de favoritos dentro de Dynamic Island.']
  },
  'dynamicIsland.showDots': {
    en: ['Show page dots', 'Navigation dots for Player, Queue, and Devices.'],
    id: ['Titik halaman', 'Titik navigasi halaman Player, Antrean, dan Perangkat.'],
    ms: ['Titik halaman', 'Titik navigasi untuk Pemain, Baris Gilir, dan Peranti.'],
    ja: ['ページドットを表示', 'プレーヤー、キュー、デバイス画面の切り替えドットを表示します。'],
    ko: ['페이지 점 표시', '플레이어, 대기열, 기기 화면 전환 점을 표시합니다.'],
    hi: ['पेज डॉट्स दिखाएं', 'प्लेयर, कतार और डिवाइस के लिए नेविगेशन डॉट्स।'],
    pt: ['Mostrar pontos de página', 'Pontos de navegação para Player, Fila e Dispositivos.'],
    de: ['Seitenpunkte anzeigen', 'Navigationspunkte für Player, Warteschlange und Geräte.'],
    fr: ['Afficher les points de page', 'Points de navigation pour Lecteur, File et Appareils.'],
    es: ['Mostrar puntos de página', 'Puntos de navegación para Reproductor, Cola y Dispositivos.']
  },
  'dynamicIsland.showProgress': {
    en: ['Show progress bar', 'Track playback progress indicator.'],
    id: ['Bilah progres', 'Indikator progres pemutaran lagu.'],
    ms: ['Bar kemajuan', 'Penunjuk kemajuan main balik lagu.'],
    ja: ['プログレスバーを表示', '曲の再生進行状況インジケーターを表示します。'],
    ko: ['진행 표시줄 표시', '곡 재생 진행률 표시줄을 표시합니다.'],
    hi: ['प्रोग्रेस बार दिखाएं', 'गाने की प्लेबैक प्रगति संकेतक।'],
    pt: ['Mostrar barra de progresso', 'Indicador de progresso da reprodução da faixa.'],
    de: ['Fortschrittsbalken anzeigen', 'Anzeige des Wiedergabefortschritts.'],
    fr: ['Afficher la barre de progression', 'Indicateur de progression de la lecture du titre.'],
    es: ['Mostrar barra de progreso', 'Indicador de progreso de reproducción de la pista.']
  },
  'dynamicIsland.showClosedProgress': {
    en: ['Show circle when closed', 'Progress ring on the right when closed.'],
    id: ['Lingkaran saat tertutup', 'Tampilkan lingkaran progres kecil di sisi kanan saat Dynamic Island tertutup.'],
    ms: ['Bulatan semasa tertutup', 'Paparkan bulatan kemajuan kecil di sebelah kanan apabila Dynamic Island tertutup.'],
    ja: ['閉じた状態の進捗リング', 'Dynamic Islandが閉じているときに右側に小さな円形プログレスを表示します。'],
    ko: ['닫혔을 때 원형 진행 표시', 'Dynamic Island가 닫혀 있을 때 오른쪽에 작은 원형 진행 표시를 나타냅니다.'],
    hi: ['बंद होने पर रिंग दिखाएं', 'Dynamic Island बंद होने पर दाईं ओर छोटा गोलाकार प्रगति रिंग दिखाएं।'],
    pt: ['Mostrar anel quando fechado', 'Exibe um pequeno anel de progresso à direita quando o Dynamic Island está fechado.'],
    de: ['Kreis im geschlossenen Zustand', 'Zeigt rechts einen kleinen Fortschrittsring, wenn das Dynamic Island geschlossen ist.'],
    fr: ['Anneau lorsque fermé', 'Affiche un petit anneau de progression à droite lorsque le Dynamic Island est fermé.'],
    es: ['Anillo cuando está cerrado', 'Muestra un pequeño anillo de progreso a la derecha cuando Dynamic Island está cerrado.']
  },
  'dynamicIsland.progressStyle': {
    en: ['Progress bar style', 'Choose thin, thick, dotted, or wave progress bar.'],
    id: ['Gaya progres', 'Pilih gaya bilah progres tipis, tebal, titik, atau gelombang.'],
    ms: ['Gaya bar kemajuan', 'Pilih gaya bar kemajuan tipis, tebal, titik, atau gelombang.'],
    ja: ['プログレスバーのスタイル', '細線、太線、ドット、またはウェーブから選択します。'],
    ko: ['진행 표시줄 스타일', '얇게, 굵게, 점선 또는 물결 스타일을 선택합니다.'],
    hi: ['प्रोग्रेस बार स्टाइल', 'पतला, मोटा, डॉट्स या लहरदार प्रोग्रेस बार चुनें।'],
    pt: ['Estilo da barra de progresso', 'Escolha barra de progresso fina, grossa, pontilhada ou ondulada.'],
    de: ['Fortschrittsbalken-Stil', 'Wähle dünnen, dicken, gepunkteten oder wellenförmigen Balken.'],
    fr: ['Style de la barre de progression', 'Choisissez une barre fine, épaisse, en pointillés ou ondulée.'],
    es: ['Estilo de barra de progreso', 'Elige barra de progreso fina, gruesa, punteada u ondulada.']
  },
  'dynamicIsland.animSpeed': {
    en: ['Animation speed', 'Adjust transition smoothness when Dynamic Island resizes.'],
    id: ['Kecepatan animasi', 'Atur kehalusan animasi saat ukuran Dynamic Island berubah.'],
    ms: ['Kelajuan animasi', 'Laraskan kelancaran peralihan apabila saiz Dynamic Island berubah.'],
    ja: ['アニメーション速度', 'Dynamic Islandのサイズ変更時の遷移の滑らかさを調整します。'],
    ko: ['애니메이션 속도', 'Dynamic Island 크기가 변경될 때 전환 부드러움을 조절합니다.'],
    hi: ['एनिमेशन गति', 'Dynamic Island का आकार बदलने पर ट्रांज़िशन की सहजता समायोजित करें।'],
    pt: ['Velocidade da animação', 'Ajuste a suavidade da transição quando o Dynamic Island muda de tamanho.'],
    de: ['Animationsgeschwindigkeit', 'Passe die Übergangsgeschwindigkeit bei Größenänderungen an.'],
    fr: ['Vitesse d’animation', 'Ajustez la fluidité des transitions lors du redimensionnement.'],
    es: ['Velocidad de animación', 'Ajusta la suavidad de la transición cuando cambia el tamaño de Dynamic Island.']
  },
  'dynamicIsland.openTrigger': {
    en: ['Open trigger', 'How Dynamic Island expands from compact mode.'],
    id: ['Membuka', 'Cara memperluas Dynamic Island dari mode ringkas.'],
    ms: ['Cara membuka', 'Cara Dynamic Island berkembang daripada mod padat.'],
    ja: ['展開トリガー', 'コンパクトモードからDynamic Islandを展開する方法。'],
    ko: ['열기 방식', '컴팩트 모드에서 Dynamic Island가 확장되는 방식입니다.'],
    hi: ['खोलने का तरीका', 'कॉम्पैक्ट मोड से Dynamic Island कैसे फैलता है।'],
    pt: ['Gatilho de abertura', 'Como o Dynamic Island se expande a partir do modo compacto.'],
    de: ['Öffnungsauslöser', 'Wie sich das Dynamic Island aus dem Kompaktmodus erweitert.'],
    fr: ['Déclencheur d’ouverture', 'Comment le Dynamic Island s’agrandit depuis le mode compact.'],
    es: ['Activador de apertura', 'Cómo se expande Dynamic Island desde el modo compacto.']
  },
  'dynamicIsland.pauseHide': {
    en: ['Hide when paused', 'Automatically collapse/hide after pausing playback.'],
    id: ['Sembunyikan saat dijeda', 'Sembunyikan otomatis setelah musik dijeda.'],
    ms: ['Sembunyikan apabila dijeda', 'Sembunyikan secara automatik selepas menjeda main balik.'],
    ja: ['一時停止時に隠す', '再生を一時停止した後に自動的に折りたたむまでの時間。'],
    ko: ['일시정지 시 숨기기', '재생 일시정지 후 자동으로 숨길 지연 시간입니다.'],
    hi: ['पॉज़ होने पर छिपाएं', 'प्लेबैक पॉज़ करने के बाद स्वचालित रूप से छिपाएं।'],
    pt: ['Ocultar ao pausar', 'Recolhe/oculta automaticamente após pausar a reprodução.'],
    de: ['Bei Pause ausblenden', 'Nach dem Pausieren automatisch ausblenden.'],
    fr: ['Masquer en pause', 'Masque automatiquement après la mise en pause.'],
    es: ['Ocultar al pausar', 'Se oculta automáticamente después de pausar la reproducción.']
  },
  'dynamicIsland.onlyPlaying': {
    en: ['Show only while playing', 'Dynamic Island hides when music is paused until playback resumes.'],
    id: ['Tampil hanya saat memutar', 'Saat musik dijeda, Dynamic Island hilang sampai lagu diputar lagi.'],
    ms: ['Tampil hanya semasa memainkan lagu', 'Dynamic Island tersembunyi apabila muzik dijeda sehingga dimainkan semula.'],
    ja: ['再生中のみ表示', '音楽が一時停止されると、再生が再開されるまでDynamic Islandを隠します。'],
    ko: ['재생 중에만 표시', '음악이 일시정지되면 다시 재생될 때까지 Dynamic Island를 숨깁니다.'],
    hi: ['केवल बजते समय दिखाएं', 'संगीत पॉज़ होने पर Dynamic Island छिप जाता है जब तक दोबारा प्ले न हो।'],
    pt: ['Mostrar apenas durante a reprodução', 'O Dynamic Island fica oculto quando a música é pausada até voltar a tocar.'],
    de: ['Nur während der Wiedergabe anzeigen', 'Blendet das Dynamic Island bei pausierter Musik aus.'],
    fr: ['Afficher uniquement pendant la lecture', 'Le Dynamic Island se masque lorsque la musique est en pause.'],
    es: ['Mostrar solo durante la reproducción', 'Dynamic Island se oculta cuando la música está en pausa.']
  },
  'about.shortcuts': {
    en: ['Keyboard shortcuts', 'View all available keyboard shortcuts in Velora.'],
    id: ['Pintasan keyboard', 'Lihat semua pintasan keyboard yang tersedia di Velora.'],
    ms: ['Pintasan papan kekunci', 'Lihat semua pintasan papan kekunci yang tersedia dalam Velora.'],
    ja: ['キーボードショートカット', 'Veloraで利用可能なすべてのショートカットを表示します。'],
    ko: ['키보드 단축키', 'Velora에서 사용 가능한 모든 키보드 단축키를 확인합니다.'],
    hi: ['कीबोर्ड शॉर्टकट', 'Velora में उपलब्ध सभी कीबोर्ड शॉर्टकट देखें।'],
    pt: ['Atalhos de teclado', 'Veja todos os atalhos de teclado disponíveis no Velora.'],
    de: ['Tastenkombinationen', 'Alle verfügbaren Tastenkombinationen in Velora anzeigen.'],
    fr: ['Raccourcis clavier', 'Afficher tous les raccourcis clavier disponibles dans Velora.'],
    es: ['Atajos de teclado', 'Ver todos los atajos de teclado disponibles en Velora.']
  },
  'about.dataFolder': {
    en: ['Data folder', 'Open the local folder where Velora stores database, cache, and downloads.'],
    id: ['Folder data', 'Buka folder lokal tempat Velora menyimpan database, cache, dan unduhan.'],
    ms: ['Folder data', 'Buka folder tempatan tempat Velora menyimpan pangkalan data, cache dan muat turun.'],
    ja: ['データフォルダー', 'データベース、キャッシュ、ダウンロードが保存されているローカルフォルダーを開きます。'],
    ko: ['데이터 폴더', 'Velora가 데이터베이스, 캐시 및 다운로드를 저장하는 로컬 폴더를 엽니다.'],
    hi: ['डेटा फ़ोल्डर', 'वह स्थानीय फ़ोल्डर खोलें जहाँ Velora डेटाबेस, कैश और डाउनलोड सहेजता है।'],
    pt: ['Pasta de dados', 'Abre a pasta local onde o Velora armazena banco de dados, cache e downloads.'],
    de: ['Datenordner', 'Öffnet den lokalen Ordner für Datenbank, Cache und Downloads.'],
    fr: ['Dossier de données', 'Ouvre le dossier local où Velora stocke la base de données, le cache et les téléchargements.'],
    es: ['Carpeta de datos', 'Abre la carpeta local donde Velora guarda la base de datos, caché y descargas.']
  },
  'dynamicIsland.screen': {
    en: ['Display', 'Choose which monitor hosts Dynamic Island.'],
    id: ['Layar aktif', 'Pilih monitor tempat Dynamic Island tampil.'],
    ms: ['Skrin aktif', 'Pilih monitor tempat Dynamic Island dipaparkan.'],
    ja: ['表示ディスプレイ', 'Dynamic Islandを表示するモニターを選択します。'],
    ko: ['표시 모니터', 'Dynamic Island를 표시할 모니터를 선택합니다.'],
    hi: ['डिस्प्ले', 'चुनें कि कौन सा मॉनिटर Dynamic Island दिखाएगा।'],
    pt: ['Tela ativa', 'Escolha qual monitor exibe o Dynamic Island.'],
    de: ['Bildschirm', 'Wähle den Monitor aus, auf dem das Dynamic Island angezeigt wird.'],
    fr: ['Écran', 'Choisissez le moniteur qui affiche le Dynamic Island.'],
    es: ['Pantalla', 'Elige qué monitor muestra Dynamic Island.']
  },
  'dynamicIsland.hideApp': {
    en: ['Hide when Velora window is active', 'Hide Dynamic Island while the main Velora window is focused.'],
    id: ['Sembunyikan saat jendela Velora aktif', 'Sembunyikan Dynamic Island saat jendela utama Velora sedang aktif.'],
    ms: ['Sembunyikan semasa tetingkap Velora aktif', 'Sembunyikan Dynamic Island semasa tetingkap utama Velora sedang difokuskan.'],
    ja: ['Veloraウィンドウがアクティブな時は隠す', 'メインのVeloraウィンドウがフォーカスされている間はDynamic Islandを非表示にします。'],
    ko: ['Velora 창이 활성화된 동안 숨기기', '메인 Velora 창이 포커스되어 있을 때 Dynamic Island를 숨깁니다.'],
    hi: ['Velora विंडो सक्रिय होने पर छिपाएं', 'मुख्य Velora विंडो फ़ोकस में होने पर Dynamic Island छिपाएं।'],
    pt: ['Ocultar quando a janela do Velora estiver ativa', 'Oculta o Dynamic Island enquanto a janela principal do Velora estiver em foco.'],
    de: ['Ausblenden, wenn Velora-Fenster aktiv ist', 'Blendet das Dynamic Island aus, solange das Velora-Hauptfenster fokussiert ist.'],
    fr: ['Masquer lorsque la fenêtre Velora est active', 'Masque le Dynamic Island lorsque la fenêtre principale de Velora est au premier plan.'],
    es: ['Ocultar cuando la ventana de Velora esté activa', 'Oculta Dynamic Island mientras la ventana principal de Velora tenga el foco.']
  },
  'dynamicIsland.hideFull': {
    en: ['Hide completely in fullscreen apps', 'Completely hide Dynamic Island during fullscreen games or videos.'],
    id: ['Sembunyikan penuh saat layar penuh', 'Sembunyikan Dynamic Island sepenuhnya saat bermain game atau menonton video layar penuh.'],
    ms: ['Sembunyikan sepenuhnya dalam skrin penuh', 'Sembunyikan Dynamic Island sepenuhnya semasa permainan atau video skrin penuh.'],
    ja: ['フルスクリーンアプリでは完全に隠す', 'フルスクリーンのゲームや動画の再生中はDynamic Islandを完全に非表示にします。'],
    ko: ['전체 화면 앱에서 완전히 숨기기', '전체 화면 게임이나 동영상을 볼 때 Dynamic Island를 완전히 숨깁니다.'],
    hi: ['फ़ुलस्क्रीन ऐप्स में पूरी तरह छिपाएं', 'फ़ुलस्क्रीन गेम या वीडियो के दौरान Dynamic Island को पूरी तरह छिपाएं।'],
    pt: ['Ocultar totalmente em tela cheia', 'Oculta totalmente o Dynamic Island durante jogos ou vídeos em tela cheia.'],
    de: ['Im Vollbildmodus komplett ausblenden', 'Blendet das Dynamic Island bei Vollbild-Spielen oder -Videos vollständig aus.'],
    fr: ['Masquer complètement en plein écran', 'Masque complètement le Dynamic Island pendant les jeux ou vidéos en plein écran.'],
    es: ['Ocultar por completo en pantalla completa', 'Oculta por completo Dynamic Island durante juegos o videos a pantalla completa.']
  },
  'dynamicIsland.idleHideSec': {
    en: ['Auto-hide when idle', 'Shrink into a thin top-edge line when untouched (paused during active lyrics playback).'],
    id: ['Sembunyikan otomatis saat diam', 'Mengecil menjadi garis tipis di tepi atas saat diam (jeda saat mode playback lirik aktif).'],
    ms: ['Sembunyikan automatik semasa melahu', 'Mengecil menjadi garis nipis di tepi atas semasa melahu (dijeda semasa main balik lirik aktif).'],
    ja: ['無操作時に自動で隠す', '無操作時に上端の細いラインに縮小します（歌詞再生モード中は一時停止）。'],
    ko: ['유휴 시 자동 숨기기', '조작이 없으면 상단 가장자리의 얇은 선으로 축소됩니다(가사 재생 모드 중에는 유지).'],
    hi: ['निष्क्रिय होने पर स्वतः छिपाएं', 'निष्क्रिय रहने पर ऊपरी किनारे की पतली रेखा में सिमट जाता है (सक्रिय लिरिक्स प्लेबैक के दौरान रुका रहता है)।'],
    pt: ['Ocultar ao ficar ocioso', 'Encolhe para uma linha fina na borda superior quando inativo (pausado durante a reprodução ativa de letras).'],
    de: ['Bei Inaktivität automatisch ausblenden', 'Schrumpft ohne Interaktion zu einer dünnen Linie am oberen Rand (pausiert bei aktiver Liedtext-Wiedergabe).'],
    fr: ['Masquage auto si inactif', 'Se réduit en une fine ligne en haut de l’écran en cas d’inactivité (suspendu pendant la lecture active des paroles).'],
    es: ['Ocultar al estar inactivo', 'Se reduce a una línea fina en el borde superior al estar inactivo (en pausa durante reproducción activa de letras).']
  },
  'dynamicIsland.reset': {
    en: ['Reset Dynamic Island', 'Restore all Dynamic Island options to defaults. Enabled state stays unchanged.'],
    id: ['Atur ulang Dynamic Island', 'Semua pilihan di halaman ini kembali ke awal. Nyala atau matinya tidak berubah.'],
    ms: ['Tetap semula Dynamic Island', 'Kembalikan semua pilihan Dynamic Island kepada asal.'],
    ja: ['Dynamic Islandをリセット', 'Dynamic Islandのすべてのオプションを初期設定に戻します。'],
    ko: ['Dynamic Island 초기화', '모든 Dynamic Island 옵션을 기본값으로 복원합니다.'],
    hi: ['Dynamic Island रीसेट करें', 'सभी Dynamic Island विकल्पों को डिफ़ॉल्ट पर बहाल करें।'],
    pt: ['Redefinir Dynamic Island', 'Restaura todas as opções do Dynamic Island para os padrões.'],
    de: ['Dynamic Island zurücksetzen', 'Setzt alle Dynamic Island-Optionen auf die Standardwerte zurück.'],
    fr: ['Réinitialiser Dynamic Island', 'Restaure toutes les options du Dynamic Island par défaut.'],
    es: ['Restablecer Dynamic Island', 'Restaura todas las opciones de Dynamic Island a sus valores predeterminados.']
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
    defaultValue: 'false'
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
    defaultValue: '0',
    options: [
      { value: '0', label: 'opt.never' },
      { value: '3', label: 'opt.3sec' },
      { value: '5', label: 'opt.5sec' },
      { value: '10', label: 'opt.10sec' }
    ]
  },

  {
    id: 'lyrics.textSize',
    category: 'lyrics',
    group: 'grp.lyricsDisplay',
    title: 'Font size',
    description: 'Adjust lyrics text size.',
    widget: 'segmented',
    defaultValue: '0',
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
    defaultValue: 'false'
  },
  {
    id: 'lyrics.blurOthers',
    category: 'lyrics',
    group: 'grp.lyricsDisplay',
    title: 'Blur inactive lines',
    description: 'Full performance mode only.',
    widget: 'toggle',
    defaultValue: 'false'
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
    defaultValue: '250',
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

  {
    id: 'content.country',
    category: 'content',
    group: 'grp.regionLanguage',
    title: 'Content region',
    description: 'Catalog region for Home, Charts, and Search recommendations.',
    widget: 'select',
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

  {
    id: 'integrations.jam.enabled',
    category: 'backup',
    group: 'grp.syncPhone',
    title: 'Allow Velora Jam (Wi-Fi remote)',
    description: 'Grant permission to turn Velora Jam on or off from the top bar.',
    widget: 'toggle',
    defaultValue: 'false'
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
    title: 'Closed style',
    description: 'Default shape when Dynamic Island is closed (Hidden shrinks into a top-edge line unless active lyrics playback is running).',
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
    id: 'dynamicIsland.coverStyle',
    category: 'dynamic-island',
    group: 'grp.islandContent',
    title: 'Album art style',
    description: 'Choose square or circular cover art inside Dynamic Island.',
    widget: 'segmented',
    defaultValue: 'square',
    options: [
      { value: 'square', label: 'opt.square' },
      { value: 'circle', label: 'opt.circle' }
    ],
    visibleWhen: { id: 'dynamicIsland.showCover', equals: 'true' }
  },
  {
    id: 'dynamicIsland.coverSpin',
    category: 'dynamic-island',
    group: 'grp.islandContent',
    title: 'Rotate circular album art',
    description: 'Spin circular album artwork while music is playing.',
    widget: 'toggle',
    defaultValue: 'true',
    visibleWhen: { id: 'dynamicIsland.coverStyle', equals: 'circle' },
    disabledWhen: { id: 'dynamicIsland.showCover', equals: 'false' }
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
    id: 'dynamicIsland.showClosedProgress',
    category: 'dynamic-island',
    group: 'grp.islandContent',
    title: 'Show circle when closed',
    description: 'Progress ring on the right when closed.',
    widget: 'toggle',
    defaultValue: 'true',
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' },
    disabledWhen: { id: 'dynamicIsland.showProgress', equals: 'false' }
  },
  {
    id: 'dynamicIsland.closedCoverSpin',
    category: 'dynamic-island',
    group: 'grp.islandContent',
    title: 'Rotate cover when closed',
    description: 'Spin small song artwork when Dynamic Island is closed and playing.',
    widget: 'toggle',
    defaultValue: 'false',
    visibleWhen: { id: 'dynamicIsland.showClosedCover', equals: 'true' },
    disabledWhen: { id: 'dynamicIsland.showCover', equals: 'false' }
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
    title: 'Hide completely in fullscreen apps',
    description: 'Completely hide Dynamic Island during fullscreen games or videos.',
    widget: 'toggle',
    defaultValue: 'true',
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' }
  },
  {
    id: 'dynamicIsland.idleHideSec',
    category: 'dynamic-island',
    group: 'grp.islandScreen',
    title: 'Auto-hide when idle',
    description: 'Shrink into a thin top-edge line when untouched (paused during active lyrics playback).',
    widget: 'combobox',
    defaultValue: '0',
    min: 0,
    max: 3600,
    unit: 's',
    options: [
      { value: '0', label: 'opt.off' },
      { value: '5', label: '5s' },
      { value: '15', label: '15s' },
      { value: '30', label: '30s' }
    ],
    visibleWhen: { id: 'dynamicIsland.enabled', equals: 'true' },
    disabledWhen: { id: 'dynamicIsland.compactStyle', equals: 'TERSEMBUNYI' }
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
  const [title, description] = pair ? (pair[code] ?? pair.en) : [item.title, item.description];
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
