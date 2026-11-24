<script lang="ts">
  import { fade, fly, scale } from 'svelte/transition';
  import { api, formatDuration, resolveCoverUrl } from '../lib/api';
  import {
    PREFERENCE_APP_TABS,
    PREFERENCE_CATEGORIES,
    PREFERENCE_SETTINGS,
    getPrefBool,
    getPrefValue,
    localizeItem,
    type PreferenceItemDef
  } from '../lib/preferences';
  import { getPrefLang, tr } from '../lib/i18n';
  import type { DailyStreakRow, DownloadProgress, FireStreakStatus, ListeningStats, PlayHistoryItem, Playlist, Track, UpdateCheckResult, ViewRoute } from '../lib/types';
  import CoverImage from '../components/CoverImage.svelte';
  import EmptyState from '../components/EmptyState.svelte';
  import EqIndicator from '../components/EqIndicator.svelte';
  import Icon from '../components/Icon.svelte';
  import ModalDialog from '../components/ModalDialog.svelte';
  import StatsChart from '../components/StatsChart.svelte';
  import TrackActionMenu from '../components/TrackActionMenu.svelte';

  interface Props {
    mode: 'history' | 'listening-stats' | 'fire-streak' | 'preferences';
    history: PlayHistoryItem[];
    listeningStats: ListeningStats;
    fireStreak: FireStreakStatus;
    preferences: Record<string, string>;
    tracks: Track[];
    playlists?: Playlist[];
    downloadStates?: Record<number, DownloadProgress>;
    currentTrackId?: number | null;
    currentHistoryEntryId?: number | null;
    currentExternalId?: string | null;
    currentTrackTitle?: string | null;
    currentTrackArtist?: string | null;
    isPlaying?: boolean;
    mediaBaseUrl: string;
    accentColor: string;
    isMobile?: boolean;
    activePrefTab?: string;
    prefSearch?: string;
    onPlayTrack: (track: Track, list: Track[]) => void;
    onTogglePlay?: () => void;
    onPlayNext?: (track: Track) => void;
    onAddToQueue?: (track: Track) => void;
    onAddToPlaylist?: (playlistId: number, trackId: number) => void;
    onToggleFavorite?: (trackId: number) => void;
    onDownloadTrack?: (track: Track) => void;
    onRemoveDownload?: (trackId: number) => void;
    onOpenArtist?: (artistId: number) => void;
    onOpenTrackAlbum?: (track: Track) => void;
    onCopyLink?: (track: Track) => void;
    onClearHistory: () => void;
    onSavePreferences: (prefs: Record<string, string>) => void;
    onClearCache: () => Promise<number>;
    onNavigate?: (route: ViewRoute) => void;
    onBackFromPreferences?: () => void;
  }

  let {
    mode,
    history,
    listeningStats,
    fireStreak,
    preferences,
    tracks,
    playlists = [],
    downloadStates = {},
    currentTrackId = null,
    currentHistoryEntryId = null,
    currentExternalId = null,
    currentTrackTitle = null,
    currentTrackArtist = null,
    isPlaying = false,
    mediaBaseUrl,
    accentColor: _accentColor,
    isMobile = false,
    activePrefTab = $bindable('appearance'),
    prefSearch = $bindable(''),
    onPlayTrack,
    onTogglePlay,
    onPlayNext,
    onAddToQueue,
    onAddToPlaylist,
    onToggleFavorite,
    onDownloadTrack,
    onRemoveDownload,
    onOpenArtist,
    onOpenTrackAlbum,
    onCopyLink,
    onClearHistory,
    onSavePreferences,
    onClearCache,
    onNavigate,
    onBackFromPreferences
  }: Props = $props();

  const isDesktopApp =
    typeof window !== 'undefined' &&
    Boolean((window as unknown as { go?: { main?: { App?: unknown } } }).go?.main?.App);
  const allowDynamicIsland = $derived(!isMobile && isDesktopApp);
  const mobilePrefTabs = $derived([
    ...(allowDynamicIsland ? PREFERENCE_CATEGORIES : PREFERENCE_CATEGORIES.filter((c) => c.id !== 'dynamic-island')),
    ...PREFERENCE_APP_TABS
  ]);

  $effect(() => {
    if (!allowDynamicIsland && activePrefTab === 'dynamic-island') {
      activePrefTab = 'appearance';
    }
  });

  let copiedKey = $state('');
  let cacheToast = $state('');
  let openSelectId = $state<string | null>(null);
  let openSelectFlipUp = $state(false);
  let clearingCache = $state(false);
  let confirmClearCache = $state(false);
  let historyTab = $state<'recent' | 'stats'>('recent');
  let historyFilter = $state('');
  let openHistoryActionId = $state<number | null>(null);
  let confirmClearHistory = $state(false);
  let confirmResetIsland = $state(false);
  let showShortcutsModal = $state(false);
  let showReleaseNotesModal = $state(false);
  let showSupportModal = $state(false);
  let showFeedbackModal = $state(false);
  let feedbackTitle = $state('');
  let feedbackDetail = $state('');
  let feedbackScreenshotFile = $state<File | null>(null);
  let feedbackScreenshotUrl = $state('');
  let feedbackSubmitting = $state(false);
  let feedbackSubmitted = $state(false);
  let feedbackError = $state('');
  let feedbackFileInputEl = $state<HTMLInputElement | null>(null);
  let dataDirPath = $state('');
  let supportQrDataUrl = $state('');
  const SUPPORT_TRIBE_URL = 'https://sociabuzz.com/yoapipp/tribe';

  const initToday = new Date();
  const currentYear = initToday.getFullYear();
  const currentMonth = initToday.getMonth() + 1;
  const currentDay = initToday.getDate();
  let calYear = $state(currentYear);
  let calMonth = $state(currentMonth);
  let monthlyStreakRows = $state<DailyStreakRow[]>([]);
  let monthlyStreakKey = $state('');
  let selectedStatsDate = $state('');
  let dateFilteredStats = $state<ListeningStats | null>(null);
  let statsLoading = $state(false);
  let initialStatsLoaded = $state(false);
  let statsReqSeq = 0;

  const displayedStats = $derived(
    selectedStatsDate && dateFilteredStats ? dateFilteredStats : listeningStats
  );
  const todayDateStr = `${currentYear}-${String(currentMonth).padStart(2, '0')}-${String(currentDay).padStart(2, '0')}`;

  async function selectStatsDate(dateStr: string) {
    if (dateStr > todayDateStr) return;
    if (selectedStatsDate === dateStr) {
      selectedStatsDate = '';
      dateFilteredStats = null;
      return;
    }
    const seq = ++statsReqSeq;
    selectedStatsDate = dateStr;
    const res = await api.getListeningStatsByDate(dateStr);
    if (seq === statsReqSeq) {
      dateFilteredStats = res;
    }
  }

  function clearSelectedStatsDate() {
    selectedStatsDate = '';
    dateFilteredStats = null;
  }

  const isCurrentCalMonth = $derived(calYear === currentYear && calMonth === currentMonth);
  const currentMonthIdx = currentYear * 12 + (currentMonth - 1);
  const minAllowedMonthIdx = $derived.by(() => {
    const defaultMin = currentMonthIdx - 2;
    let earliest = currentMonthIdx;
    for (const r of fireStreak.last30Days ?? []) {
      if (!r.streakMet) continue;
      const ry = Number(r.date.slice(0, 4));
      const rm = Number(r.date.slice(5, 7)) - 1;
      const idx = ry * 12 + rm;
      if (idx < earliest) earliest = idx;
    }
    return Math.min(defaultMin, earliest);
  });
  const canGoPrevCalMonth = $derived(calYear * 12 + (calMonth - 1) > minAllowedMonthIdx);
  let chartPeriod = $state<'week' | 'month' | 'year'>('week');
  const CHART_PERIOD_OPTIONS = [
    { value: 'week' as const, label: 'Minggu ini' },
    { value: 'month' as const, label: 'Bulan ini' },
    { value: 'year' as const, label: 'Tahun ini' }
  ];

  const monthlyStreakStats = $derived.by(() => {
    const key = `${currentYear}-${String(currentMonth).padStart(2, '0')}`;
    const daysInMonth = new Date(currentYear, currentMonth, 0).getDate();
    const byDate = new Map<string, DailyStreakRow>();
    for (const r of fireStreak.last30Days ?? []) {
      byDate.set(r.date, r);
    }
    if (monthlyStreakKey === key) {
      for (const r of monthlyStreakRows) {
        byDate.set(r.date, r);
      }
    }
    let metDays = 0;
    let brokenDays = 0;
    let hadStreak = false;
    const firstOfMonth = `${key}-01`;
    for (const r of byDate.values()) {
      if (r.date < firstOfMonth && r.streakMet) {
        hadStreak = true;
        break;
      }
    }
    for (let d = 1; d <= currentDay; d++) {
      const dateStr = `${key}-${String(d).padStart(2, '0')}`;
      const row = byDate.get(dateStr);
      const sec = row?.secondsListened ?? 0;
      const met = row?.streakMet ?? false;
      if (met) {
        metDays++;
        hadStreak = true;
      } else if (d < currentDay && sec === 0 && hadStreak) {
        brokenDays++;
        hadStreak = false;
      }
    }
    const pct = Math.min(100, Math.round((metDays / Math.max(1, daysInMonth)) * 100));
    const remainingRecoveries = Math.max(0, 3 - brokenDays);
    return { metDays, daysInMonth, pct, remainingRecoveries };
  });

  function prevCalMonth() {
    if (!canGoPrevCalMonth) return;
    if (calMonth === 1) {
      calYear -= 1;
      calMonth = 12;
    } else {
      calMonth -= 1;
    }
  }

  function nextCalMonth() {
    if (isCurrentCalMonth) return;
    if (calMonth === 12) {
      calYear += 1;
      calMonth = 1;
    } else {
      calMonth += 1;
    }
  }

  $effect(() => {
    if (mode !== 'listening-stats') {
      initialStatsLoaded = false;
      return;
    }
    if (!initialStatsLoaded) {
      const seq = ++statsReqSeq;
      statsLoading = true;
      const targetDate = selectedStatsDate;
      const call = targetDate ? api.getListeningStatsByDate(targetDate) : api.getListeningStats();
      void call
        .then((res) => {
          if (seq !== statsReqSeq) return;
          if (targetDate) {
            dateFilteredStats = res;
          }
        })
        .catch(() => {})
        .finally(() => {
          if (seq === statsReqSeq) {
            initialStatsLoaded = true;
            statsLoading = false;
          }
        });
    }
  });

  $effect(() => {
    if (mode !== 'listening-stats') return;
    const y = calYear;
    const m = calMonth;
    const key = `${y}-${String(m).padStart(2, '0')}`;
    const _todaySec = fireStreak.todaySeconds;
    const _todayTracks = fireStreak.todayTracks;
    void api.getMonthlyStreak(y, m).then((rows) => {
      if (calYear === y && calMonth === m) {
        monthlyStreakRows = rows;
        monthlyStreakKey = key;
      }
    });
  });

  const calendarView = $derived.by(() => {
    const y = calYear;
    const m = calMonth;
    const key = `${y}-${String(m).padStart(2, '0')}`;
    const daysInMonth = new Date(y, m, 0).getDate();
    const firstWeekday = (new Date(y, m - 1, 1).getDay() + 6) % 7;
    const prevMonthDaysCount = new Date(y, m - 1, 0).getDate();
    const prevDays: number[] = [];
    for (let i = firstWeekday - 1; i >= 0; i--) {
      prevDays.push(prevMonthDaysCount - i);
    }
    const trailingCount = Math.max(0, 42 - (prevDays.length + daysInMonth));
    const nextDays: number[] = [];
    for (let i = 1; i <= trailingCount; i++) {
      nextDays.push(i);
    }

    const byDate = new Map<string, DailyStreakRow>();
    for (const r of fireStreak.last30Days ?? []) {
      byDate.set(r.date, r);
    }
    if (monthlyStreakKey === key) {
      for (const r of monthlyStreakRows) {
        byDate.set(r.date, r);
      }
    }

    const firstOfMonth = `${key}-01`;
    let hadActiveStreakBefore = false;
    for (const r of byDate.values()) {
      if (r.date < firstOfMonth && r.streakMet) {
        hadActiveStreakBefore = true;
        break;
      }
    }

    const days: Array<DailyStreakRow & { dayNum: number; isToday: boolean; isFuture: boolean; streakBroken: boolean }> = [];
    for (let d = 1; d <= daysInMonth; d++) {
      const dateStr = `${key}-${String(d).padStart(2, '0')}`;
      const existing = byDate.get(dateStr);
      const isToday = y === currentYear && m === currentMonth && d === currentDay;
      const isFuture = y === currentYear && m === currentMonth && d > currentDay;
      const secondsListened = existing?.secondsListened ?? 0;
      const streakMet = existing?.streakMet ?? false;
      const streakBroken = !isFuture && !isToday && !streakMet && secondsListened === 0 && hadActiveStreakBefore;
      if (streakMet) {
        hadActiveStreakBefore = true;
      } else if (streakBroken) {
        hadActiveStreakBefore = false;
      }
      days.push({
        date: dateStr,
        secondsListened,
        tracksCount: existing?.tracksCount ?? 0,
        streakMet,
        dayNum: d,
        isToday,
        isFuture,
        streakBroken
      });
    }

    const numericLabel = `${String(m).padStart(2, '0')}/${y}`;
    return { prevDays, days, nextDays, numericLabel };
  });
  let updateStatusText = $state('');
  let checkingUpdates = $state(false);
  let updateResult = $state<UpdateCheckResult | null>(null);
  let showUpdateModal = $state(false);
  let shakingParentId = $state<string | null>(null);
  let shakeTimer: ReturnType<typeof setTimeout> | undefined;

  function triggerParentShake(parentId: string) {
    if (shakeTimer) clearTimeout(shakeTimer);
    shakingParentId = null;
    document.getElementById(`pref-row-${parentId}`)?.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
    requestAnimationFrame(() => {
      shakingParentId = parentId;
      shakeTimer = setTimeout(() => {
        shakingParentId = null;
      }, 420);
    });
  }

  function setFeedbackScreenshot(file: File | null) {
    if (feedbackScreenshotUrl) {
      URL.revokeObjectURL(feedbackScreenshotUrl);
    }
    feedbackScreenshotFile = file;
    feedbackScreenshotUrl = file ? URL.createObjectURL(file) : '';
  }

  function handleFeedbackFileChange(e: Event) {
    const input = e.currentTarget as HTMLInputElement;
    const file = input.files?.[0] ?? null;
    if (file && file.type.startsWith('image/')) {
      setFeedbackScreenshot(file);
    }
  }

  function resetFeedbackForm() {
    feedbackTitle = '';
    feedbackDetail = '';
    setFeedbackScreenshot(null);
    if (feedbackFileInputEl) feedbackFileInputEl.value = '';
    feedbackSubmitted = false;
    feedbackError = '';
  }

  function fileToDataURL(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = () => resolve(typeof reader.result === 'string' ? reader.result : '');
      reader.onerror = () => reject(new Error('Gagal membaca file gambar'));
      reader.readAsDataURL(file);
    });
  }

  async function submitFeedbackForm() {
    const title = feedbackTitle.trim();
    const detail = feedbackDetail.trim();
    if (!title || !detail || feedbackSubmitting) return;
    feedbackSubmitting = true;
    feedbackError = '';
    try {
      let dataUrl = '';
      let fileName = '';
      if (feedbackScreenshotFile) {
        fileName = feedbackScreenshotFile.name;
        dataUrl = await fileToDataURL(feedbackScreenshotFile);
      }
      await api.submitFeedback(title, detail, fileName, dataUrl);
      feedbackSubmitted = true;
    } catch (err) {
      feedbackError = err instanceof Error ? err.message : String(err || 'Gagal mengirim masukan');
    } finally {
      feedbackSubmitting = false;
    }
  }

  $effect(() => {
    if (mode === 'preferences' && isAppTab(activePrefTab)) {
      if (!dataDirPath) {
        void api.getDataDir().then((dir) => {
          if (dir) dataDirPath = dir;
        });
      }
      if (!supportQrDataUrl) {
        void api.generateQRDataURL(SUPPORT_TRIBE_URL).then((url) => {
          if (url) supportQrDataUrl = url;
        });
      }
    }
  });

  async function handleCheckForUpdates() {
    if (checkingUpdates) return;
    checkingUpdates = true;
    updateStatusText = tr(lang, 'nav.checkingUpdates');
    try {
      const res = await api.checkForUpdates();
      updateResult = res;
      if (res.hasUpdate) {
        updateStatusText = `${tr(lang, 'nav.updateAvailableBadge')} v${res.latestVersion}`;
        showUpdateModal = true;
      } else {
        updateStatusText = `${tr(lang, 'nav.upToDate')} (v${res.currentVersion})`;
      }
    } catch {
      updateStatusText = tr(lang, 'nav.updateCheckFailed');
    } finally {
      checkingUpdates = false;
    }
  }

  const filteredHistory = $derived.by(() => {
    const q = historyFilter.trim().toLowerCase();
    if (!q) return history;
    return history.filter(
      (h) =>
        h.title.toLowerCase().includes(q) ||
        h.artistName.toLowerCase().includes(q) ||
        h.albumTitle.toLowerCase().includes(q)
    );
  });

  const effectivePrefs = $derived(preferences);
  const lang = $derived(getPrefLang(effectivePrefs));

  const visibleSettingsForCategory = $derived.by(() => {
    const q = prefSearch.trim().toLowerCase();
    return PREFERENCE_SETTINGS.map((s) => localizeItem(s, lang))
      .filter((s) => {
        if (!allowDynamicIsland && s.category === 'dynamic-island') {
          return false;
        }
        if (q) {
          return (
            s.title.toLowerCase().includes(q) ||
            s.description.toLowerCase().includes(q) ||
            s.group.toLowerCase().includes(q) ||
            s.category.toLowerCase().includes(q)
          );
        }
        return s.category === activePrefTab;
      });
  });

  const groupedVisibleSettings = $derived.by(() => {
    const groups: { name: string; items: PreferenceItemDef[] }[] = [];
    for (const item of visibleSettingsForCategory) {
      let grp = groups.find((g) => g.name === item.group);
      if (!grp) {
        grp = { name: item.group, items: [] };
        groups.push(grp);
      }
      grp.items.push(item);
    }
    return groups;
  });

  const activeCategoryTitle = $derived.by(() => {
    const cat = PREFERENCE_CATEGORIES.find((c) => c.id === activePrefTab);
    return cat ? tr(lang, cat.titleKey, cat.title) : activePrefTab;
  });

  function formatListeningDuration(minutes: number): string {
    const totalSeconds = Math.max(0, Math.round((minutes || 0) * 60));
    if (totalSeconds < 60) {
      return `${totalSeconds} ${lang === 'en' ? 's' : 'dtk'}`;
    }
    const mins = Math.round((minutes || 0) * 10) / 10;
    const displayMin = Number.isInteger(mins) ? String(mins) : mins.toFixed(1);
    return `${displayMin} ${tr(lang, 'nav.minutesShort')}`;
  }

  const maxTopTrackPlays = $derived(
    Math.max(1, ...(displayedStats.topTracks ?? []).map((t) => t.plays))
  );
  const maxTopArtistPlays = $derived(
    Math.max(1, ...(displayedStats.topArtists ?? []).map((a) => a.plays))
  );

  const todayHourlySeries = $derived.by(() => {
    const hrs = displayedStats.todayHours ?? displayedStats.timeOfDay ?? [];
    const labels = hrs.map((h) => h.label.slice(0, 2));
    const values = hrs.map((h) => h.minutes);
    const plays = hrs.map((h) => h.plays);
    let peakIdx = -1;
    let peakVal = 0;
    for (let i = 0; i < values.length; i++) {
      if (values[i] > peakVal) {
        peakVal = values[i];
        peakIdx = i;
      }
    }
    const totalMin = Math.round(values.reduce((a, b) => a + b, 0) * 10) / 10;
    return { labels, values, plays, peakIdx, totalMin };
  });

  const periodRangeSeries = $derived.by(() => {
    const src =
      chartPeriod === 'year'
        ? (displayedStats.yearSeries ?? [])
        : chartPeriod === 'month'
          ? (displayedStats.monthSeries ?? [])
          : (displayedStats.weekSeries ?? []);
    const labels = src.map((d) => d.label);
    const values = src.map((d) => d.minutes);
    const plays = src.map((d) => d.plays);
    let highlightIndex = -1;
    let maxVal = 0;
    for (let i = 0; i < values.length; i++) {
      if (values[i] > maxVal) {
        maxVal = values[i];
        highlightIndex = i;
      }
    }
    const totalMin = Math.round(values.reduce((a, b) => a + b, 0) * 10) / 10;
    return { labels, values, plays, highlightIndex, totalMin };
  });

  const SHORT_DAY_KEYS = ['nav.dayMon', 'nav.dayTue', 'nav.dayWed', 'nav.dayThu', 'nav.dayFri', 'nav.daySat', 'nav.daySun'] as const;
  const FULL_DAY_KEYS = ['nav.dayMonFull', 'nav.dayTueFull', 'nav.dayWedFull', 'nav.dayThuFull', 'nav.dayFriFull', 'nav.daySatFull', 'nav.daySunFull'] as const;

  function localizeShortDay(dayIndex: number, fallback: string): string {
    const key = SHORT_DAY_KEYS[dayIndex];
    return key ? tr(lang, key, fallback) : fallback;
  }

  function localizeFullDayName(raw: string): string {
    if (!raw || raw === '-') return '-';
    const map: Record<string, (typeof FULL_DAY_KEYS)[number]> = {
      senin: 'nav.dayMonFull',
      selasa: 'nav.dayTueFull',
      rabu: 'nav.dayWedFull',
      kamis: 'nav.dayThuFull',
      jumat: 'nav.dayFriFull',
      sabtu: 'nav.daySatFull',
      minggu: 'nav.daySunFull'
    };
    const key = map[raw.trim().toLowerCase()];
    return key ? tr(lang, key, raw) : raw;
  }

  function updateSetting(key: string, value: string) {
    if (getPrefValue(preferences, key) === value) return;
    onSavePreferences({ [key]: value });
  }

  function formatClearedBytes(bytes: number): string {
    if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / (1024 * 1024)).toFixed(2)} MB`;
  }

  function formatWatchedAt(raw: string): string {
    if (!raw) return '-';
    const normalized = raw.includes('T') ? raw : raw.replace(' ', 'T');
    const d = new Date(normalized);
    if (Number.isNaN(d.getTime())) return raw;
    const datePart = new Intl.DateTimeFormat(lang === 'id' ? 'id-ID' : lang, {
      day: '2-digit',
      month: 'short',
      year: 'numeric'
    }).format(d);
    const hh = String(d.getHours()).padStart(2, '0');
    const mm = String(d.getMinutes()).padStart(2, '0');
    return `${datePart}, ${hh}:${mm}`;
  }

  async function executeClearCache() {
    if (clearingCache) return;
    clearingCache = true;
    try {
      const freed = Number(await onClearCache()) || 0;
      const msg = freed > 0
        ? `${tr(lang, 'nav.cacheCleared')} (${formatClearedBytes(freed)})`
        : tr(lang, 'nav.cacheAlreadyClean');
      cacheToast = msg;
      setTimeout(() => {
        if (cacheToast === msg) cacheToast = '';
      }, 3000);
    } finally {
      clearingCache = false;
    }
  }

  async function handleActionClick(itemId: string) {
    if (itemId === 'dynamicIsland.reset') {
      confirmResetIsland = true;
      return;
    }
    if (itemId === 'storage.clearCache') {
      confirmClearCache = true;
      return;
    }
    if (itemId === 'about.shortcuts') {
      showShortcutsModal = true;
      return;
    }
    if (itemId === 'about.dataFolder') {
      await api.openDataFolder();
    }
  }

  function executeResetIsland() {
    const resetPatch: Record<string, string> = {};
    for (const s of PREFERENCE_SETTINGS) {
      if (s.category === 'dynamic-island' && s.id !== 'dynamicIsland.enabled' && s.widget !== 'action') {
        resetPatch[s.id] = s.defaultValue;
      }
    }
    onSavePreferences(resetPatch);
  }

  function copyInfoValue(id: string, val: string) {
    navigator.clipboard?.writeText(val);
    copiedKey = id;
    setTimeout(() => (copiedKey = ''), 1500);
  }

  function historyItemToTrack(item: PlayHistoryItem): Track {
    const byId = tracks.find((t) => t.id === item.trackId);
    if (byId) return { ...byId, historyEntryId: item.id };
    const targetTitle = normHistoryTitle(item.title);
    const targetArtist = (item.artistName || '').trim().toLowerCase();
    const byMeta = targetTitle
      ? tracks.find(
          (t) =>
            normHistoryTitle(t.title) === targetTitle &&
            (!targetArtist || t.artistName.trim().toLowerCase().includes(targetArtist))
        )
      : undefined;
    if (byMeta) return { ...byMeta, historyEntryId: item.id };
    return {
      id: item.trackId,
      title: item.title,
      artistId: 0,
      artistName: item.artistName || 'Unknown Artist',
      albumId: 0,
      albumTitle: item.albumTitle || 'Unknown Album',
      durationSec: item.durationSec || item.durationListenedSec || 180,
      format: item.format || 'OPUS',
      bitrate: 160,
      sourceType: item.sourceType || 'ytmusic',
      filePath: '',
      externalId: '',
      coverUrl: item.coverUrl || '',
      isFavorite: false,
      playCount: 1,
      historyEntryId: item.id,
      createdAt: item.playedAt || ''
    };
  }

  function normHistoryTitle(s: string): string {
    return (s || '')
      .toLowerCase()
      .replace(/\s*[\(\[](?:official|audio|video|mv|lyrics?|hd|4k).*?[\)\]]/gi, '')
      .replace(/\s+/g, ' ')
      .trim();
  }

  function isHistoryTrackActive(tr: Track): boolean {
    if (currentTrackId !== null && tr.id === currentTrackId) return true;
    if (currentExternalId && tr.externalId && tr.externalId === currentExternalId) return true;
    if (currentTrackTitle) {
      const nt = normHistoryTitle(tr.title);
      const nc = normHistoryTitle(currentTrackTitle);
      if (nt && nc && nt === nc) {
        if (!currentTrackArtist) return true;
        const a1 = (tr.artistName || '').toLowerCase();
        const a2 = currentTrackArtist.toLowerCase();
        if (a1.includes(a2) || a2.includes(a1)) return true;
      }
    }
    return false;
  }

  const activeHistoryEntryId = $derived.by<number | null>(() => {
    const matchingIds: number[] = [];
    for (const item of history) {
      if (isHistoryTrackActive(historyItemToTrack(item))) {
        matchingIds.push(item.id);
      }
    }
    if (matchingIds.length === 0) return null;
    if (currentHistoryEntryId !== null && matchingIds.includes(currentHistoryEntryId)) {
      return currentHistoryEntryId;
    }
    return matchingIds[0];
  });

  function playHistoryTrack(item: PlayHistoryItem) {
    const list = history.map(historyItemToTrack);
    const target = list.find((t) => t.historyEntryId === item.id) ?? historyItemToTrack(item);
    onPlayTrack(target, list.length > 0 ? list : [target]);
  }

  function playAllHistory() {
    if (history.length === 0) return;
    const list = history.map(historyItemToTrack);
    onPlayTrack(list[0], list);
  }

  function isAppTab(tab: string): boolean {
    return PREFERENCE_APP_TABS.some((t) => t.id === tab);
  }
</script>

<svelte:window
  onclick={() => (openSelectId = null)}
  onkeydown={(e) => {
    if (e.key === 'Escape' && openSelectId) {
      openSelectId = null;
    }
  }}
/>

{#snippet m3Switch(checked: boolean, label: string, onToggle: () => void)}
  <button
    type="button"
    role="switch"
    aria-checked={checked}
    aria-label={label}
    onclick={onToggle}
    class="w-[52px] h-8 rounded-full relative transition-all duration-200 cursor-pointer shrink-0 flex items-center hover:brightness-110"
    style={checked
      ? 'background-color: var(--md-sys-color-primary);'
      : 'background-color: var(--md-sys-color-surface-container-highest);'}
  >
    <span
      class="rounded-full transition-all duration-200 shadow-xs"
      style={checked
        ? 'width: 22px; height: 22px; transform: translateX(25px); background-color: var(--md-sys-color-on-primary);'
        : 'width: 16px; height: 16px; transform: translateX(6px); background-color: var(--md-sys-color-on-surface-variant);'}
    ></span>
  </button>
{/snippet}

{#snippet settingTitleRow(item: PreferenceItemDef, parentDisabled: boolean, showConflictBadge: boolean)}
  <div class="inline-flex items-center gap-1.5 min-h-[20px]">
    <span
      class="text-sm leading-5 font-semibold {parentDisabled ? 'opacity-45 select-none' : ''}"
      style="color: var(--md-sys-color-on-surface);"
    >{item.title}</span>
    {#if showConflictBadge}
      <span
        class="group/warn relative inline-flex items-center justify-center w-5 h-5 rounded-full shrink-0 cursor-help"
        style="color: var(--md-sys-color-error);"
        aria-label={tr(lang, 'nav.disableFullscreenFirstHint')}
      >
        <Icon name="warning-circle" weight="fill" size={16} />
        <span
          role="tooltip"
          class="pointer-events-none absolute left-1/2 -translate-x-1/2 bottom-full mb-2 px-2.5 py-1.5 rounded-lg text-[11px] leading-snug font-medium whitespace-nowrap opacity-0 scale-95 group-hover/warn:opacity-100 group-hover/warn:scale-100 transition-all duration-150 z-50 shadow-lg border"
          style="background-color: var(--md-sys-color-inverse-surface); color: var(--md-sys-color-inverse-on-surface); border-color: var(--md-sys-color-outline-variant);"
        >
          {tr(lang, 'nav.disableFullscreenFirstHint')}
        </span>
      </span>
    {/if}
  </div>
{/snippet}

{#snippet settingControl(item: PreferenceItemDef)}
  {@const val = getPrefValue(effectivePrefs, item.id)}
  {@const checked = val === 'true'}
  {@const isOpen = openSelectId === item.id}
  {@const disabledByParent = Boolean(item.visibleWhen && getPrefValue(effectivePrefs, item.visibleWhen.id) !== item.visibleWhen.equals)}
  {@const disabledByConflict = Boolean(item.disabledWhen && getPrefValue(effectivePrefs, item.disabledWhen.id) === item.disabledWhen.equals)}
  {@const parentDisabled = disabledByParent || disabledByConflict}
  {@const showConflictBadge = disabledByConflict && !disabledByParent}
  {@const shakeTargetId = disabledByParent ? item.visibleWhen!.id : disabledByConflict ? item.disabledWhen!.id : null}
  {@const isDropdownWidget = item.widget === 'select' || item.widget === 'multi-select' || item.widget === 'combobox' || (item.widget === 'segmented' && (item.options?.length ?? 0) >= 3)}
  {@const isShaking = shakingParentId === item.id}
  <div
    id="pref-row-{item.id}"
    class="py-3.5 first:pt-2 px-1 flex flex-col sm:flex-row sm:items-center justify-between gap-2.5 sm:gap-4 relative {isOpen ? 'z-30' : ''}"
  >
    {#if isDropdownWidget}
      <button
        type="button"
        onclick={(e) => {
          e.stopPropagation();
          if (parentDisabled && shakeTargetId) {
            triggerParentShake(shakeTargetId);
            return;
          }
          if (isOpen) {
            openSelectId = null;
          } else {
            const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
            openSelectFlipUp = window.innerHeight - rect.bottom < 220;
            openSelectId = item.id;
          }
        }}
        class="min-w-0 flex-1 space-y-0.5 text-left {parentDisabled ? 'cursor-not-allowed select-none' : 'cursor-pointer hover:opacity-90 transition-opacity'} {isShaking ? 'velora-shake' : ''}"
      >
        {@render settingTitleRow(item, parentDisabled, showConflictBadge)}
        <div
          class="text-xs leading-relaxed {parentDisabled ? 'opacity-45 select-none' : ''}"
          style="color: var(--md-sys-color-on-surface-variant);"
        >
          {item.description}
        </div>
      </button>
    {:else}
      <div class="min-w-0 flex-1 space-y-0.5 {isShaking ? 'velora-shake' : ''}">
        {@render settingTitleRow(item, parentDisabled, showConflictBadge)}
        <div
          class="text-xs leading-relaxed {parentDisabled ? 'opacity-45 select-none' : ''}"
          style="color: var(--md-sys-color-on-surface-variant);"
        >
          {item.description}
        </div>
      </div>
    {/if}

    <div class="relative shrink-0 flex items-center gap-2 {isShaking ? 'velora-shake' : ''}">
      {#if parentDisabled && shakeTargetId}
        <button
          type="button"
          aria-label={item.title}
          onclick={() => triggerParentShake(shakeTargetId)}
          class="absolute -inset-1 z-20 rounded-xl cursor-not-allowed"
          title={disabledByParent
            ? tr(lang, 'nav.enableMainToggleFirstHint')
            : tr(lang, 'nav.disableFullscreenFirstHint')}
        ></button>
      {/if}
      <div class="flex items-center gap-2 {parentDisabled ? 'opacity-45 pointer-events-none select-none' : ''}">
      {#if item.widget === 'toggle'}
        {@render m3Switch(checked, item.title, () => updateSetting(item.id, checked ? 'false' : 'true'))}
      {:else if item.widget === 'segmented' && (item.options?.length ?? 0) < 3}
        <div
          role="radiogroup"
          aria-label={item.title}
          class="inline-flex p-1 rounded-xl gap-1"
          style="background-color: var(--md-sys-color-surface-container-highest); border: 1px solid var(--md-sys-color-outline-variant);"
        >
          {#each item.options || [] as opt (opt.value)}
            {@const isActive = val === opt.value}
            <button
              type="button"
              role="radio"
              aria-checked={isActive}
              onclick={() => updateSetting(item.id, opt.value)}
              class="px-3.5 py-1.5 rounded-lg text-xs font-medium cursor-pointer transition-all duration-150 {isActive ? 'font-semibold shadow-xs' : 'hover:bg-[var(--md-sys-color-state-hover)]'}"
              style={isActive
                ? 'background-color: var(--md-sys-color-primary); color: var(--md-sys-color-on-primary);'
                : 'color: var(--md-sys-color-on-surface-variant);'}
            >
              {opt.label}
            </button>
          {/each}
        </div>
      {:else if item.widget === 'stepper'}
        {@const curNum = Number(val) || 1}
        {@const minV = item.min ?? 0.7}
        {@const maxV = item.max ?? 1.3}
        {@const stepV = item.step ?? 0.1}
        {@const pct = Math.round(curNum * 100)}
        <div class="flex items-center gap-2">
          {#if pct !== 100}
            <button
              type="button"
              onclick={() => updateSetting(item.id, '1')}
              class="m3-btn-outlined px-2.5 h-9 text-[11px] font-medium cursor-pointer"
            >
              {tr(lang, 'nav.reset')}
            </button>
          {/if}
          <div
            class="inline-flex items-center h-9 rounded-lg overflow-hidden"
            style="background-color: var(--md-sys-color-surface-container-highest); border: 1px solid var(--md-sys-color-outline-variant);"
          >
            <button
              type="button"
              aria-label="Perkecil skala"
              disabled={curNum <= minV + 0.001}
              onclick={() => updateSetting(item.id, String(Math.round(Math.max(minV, curNum - stepV) * 100) / 100))}
              class="w-9 h-full flex items-center justify-center cursor-pointer disabled:opacity-35 disabled:cursor-not-allowed hover:bg-[var(--md-sys-color-state-hover)]"
              style="color: var(--md-sys-color-on-surface);"
            >
              <Icon name="minus" size={14} weight="bold" />
            </button>
            <button
              type="button"
              onclick={() => updateSetting(item.id, '1')}
              title="Reset 100%"
              class="px-3 h-full flex items-center justify-center text-xs font-semibold tabular-nums min-w-[54px] text-center cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
              style="color: var(--md-sys-color-on-surface);"
            >
              {pct}%
            </button>
            <button
              type="button"
              aria-label="Perbesar skala"
              disabled={curNum >= maxV - 0.001}
              onclick={() => updateSetting(item.id, String(Math.round(Math.min(maxV, curNum + stepV) * 100) / 100))}
              class="w-9 h-full flex items-center justify-center cursor-pointer disabled:opacity-35 disabled:cursor-not-allowed hover:bg-[var(--md-sys-color-state-hover)]"
              style="color: var(--md-sys-color-on-surface);"
            >
              <Icon name="plus" size={14} weight="bold" />
            </button>
          </div>
        </div>
      {:else if item.widget === 'action'}
        {@const isDangerAction = item.id === 'storage.clearCache' || item.id === 'dynamicIsland.reset'}
        <button
          type="button"
          disabled={item.id === 'storage.clearCache' && clearingCache}
          onclick={() => handleActionClick(item.id)}
          class={isDangerAction
            ? 'px-4 min-w-[104px] h-9 rounded-full text-xs font-bold text-rose-300 bg-rose-500/15 hover:bg-rose-500/25 inline-flex items-center justify-center cursor-pointer transition-all duration-150 hover:scale-[1.03] active:scale-95 disabled:opacity-75 disabled:cursor-wait'
            : 'm3-btn-tonal px-4 min-w-[104px] h-9 text-xs font-semibold inline-flex items-center justify-center cursor-pointer disabled:opacity-75 disabled:cursor-wait'}
        >
          {#if item.id === 'storage.clearCache' && clearingCache}
            <EqIndicator variant="loader" />
          {:else}
            {item.actionLabel || tr(lang, 'nav.run')}
          {/if}
        </button>
      {:else if item.widget === 'select' || (item.widget === 'segmented' && (item.options?.length ?? 0) >= 3)}
        {@const activeOpt = (item.options || []).find((o) => o.value === val) ?? item.options?.[0]}
        <div class="relative">
          <button
            type="button"
            aria-haspopup="listbox"
            aria-expanded={isOpen}
            onclick={(e) => {
              e.stopPropagation();
              if (isOpen) {
                openSelectId = null;
              } else {
                const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
                openSelectFlipUp = window.innerHeight - rect.bottom < 220;
                openSelectId = item.id;
              }
            }}
            class="h-10 min-w-[168px] px-3.5 rounded-lg text-xs font-medium flex items-center justify-between gap-3 cursor-pointer transition-colors duration-150 hover:bg-[var(--md-sys-color-state-hover)]"
            style="background-color: var(--md-sys-color-surface-container-highest); color: var(--md-sys-color-on-surface); border: 1px solid {isOpen ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-outline-variant)'};"
          >
            <span class="truncate">{activeOpt?.label ?? val}</span>
            <span class="transition-transform duration-200 {isOpen ? 'rotate-180' : ''}" style="color: var(--md-sys-color-on-surface-variant);">
              <Icon name="caret-down" size={14} weight="bold" />
            </span>
          </button>

          {#if isOpen}
            <div
              role="listbox"
              aria-label={item.title}
              in:fly={{ y: openSelectFlipUp ? 6 : -6, duration: 150 }}
              class="absolute right-0 {openSelectFlipUp ? 'bottom-full mb-1.5' : 'top-full mt-1.5'} min-w-[196px] max-h-64 overflow-y-auto py-1.5 rounded-xl shadow-xl z-40"
              style="background-color: var(--md-sys-color-surface-container-high); border: 1px solid var(--md-sys-color-outline-variant);"
            >
              {#each item.options || [] as opt (opt.value)}
                {@const isSelected = opt.value === val}
                <button
                  type="button"
                  role="option"
                  aria-selected={isSelected}
                  onclick={(e) => {
                    e.stopPropagation();
                    updateSetting(item.id, opt.value);
                    openSelectId = null;
                  }}
                  class="w-full px-3.5 py-2.5 text-xs font-medium flex items-center justify-between gap-3 text-left cursor-pointer transition-colors duration-150 {isSelected ? 'font-semibold' : 'hover:bg-[var(--md-sys-color-state-hover)]'}"
                  style={isSelected
                    ? 'background-color: var(--md-sys-color-primary); color: var(--md-sys-color-on-primary);'
                    : 'color: var(--md-sys-color-on-surface);'}
                >
                  <span class="truncate">{opt.label}</span>
                  {#if isSelected}
                    <Icon name="check" size={14} weight="bold" />
                  {/if}
                </button>
              {/each}
            </div>
          {/if}
        </div>
      {:else if item.widget === 'multi-select'}
        {@const selectedCodes = val.split(',').map((s) => s.trim()).filter(Boolean)}
        {@const activeCodes = selectedCodes.length > 0 ? selectedCodes : [item.defaultValue]}
        {@const selectedLabels = (item.options || []).filter((o) => activeCodes.includes(o.value)).map((o) => o.label)}
        {@const triggerText = selectedLabels.length === 0
          ? val
          : selectedLabels.length <= 2
            ? selectedLabels.join(', ')
            : `${selectedLabels.slice(0, 2).join(', ')} +${selectedLabels.length - 2}`}
        <div class="relative">
          <button
            type="button"
            aria-haspopup="listbox"
            aria-expanded={isOpen}
            onclick={(e) => {
              e.stopPropagation();
              if (isOpen) {
                openSelectId = null;
              } else {
                const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
                openSelectFlipUp = window.innerHeight - rect.bottom < 220;
                openSelectId = item.id;
              }
            }}
            class="h-10 min-w-[168px] max-w-[260px] px-3.5 rounded-lg text-xs font-medium flex items-center justify-between gap-3 cursor-pointer transition-colors duration-150 hover:bg-[var(--md-sys-color-state-hover)]"
            style="background-color: var(--md-sys-color-surface-container-highest); color: var(--md-sys-color-on-surface); border: 1px solid {isOpen ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-outline-variant)'};"
          >
            <span class="truncate">{triggerText}</span>
            <span class="transition-transform duration-200 {isOpen ? 'rotate-180' : ''}" style="color: var(--md-sys-color-on-surface-variant);">
              <Icon name="caret-down" size={14} weight="bold" />
            </span>
          </button>

          {#if isOpen}
            <div
              role="listbox"
              aria-multiselectable="true"
              aria-label={item.title}
              in:fly={{ y: openSelectFlipUp ? 6 : -6, duration: 150 }}
              class="absolute right-0 {openSelectFlipUp ? 'bottom-full mb-1.5' : 'top-full mt-1.5'} min-w-[216px] max-h-64 overflow-y-auto py-1.5 rounded-xl shadow-xl z-40"
              style="background-color: var(--md-sys-color-surface-container-high); border: 1px solid var(--md-sys-color-outline-variant);"
            >
              {#each item.options || [] as opt (opt.value)}
                {@const isChecked = activeCodes.includes(opt.value)}
                <button
                  type="button"
                  role="option"
                  aria-selected={isChecked}
                  onclick={(e) => {
                    e.stopPropagation();
                    let next: string[];
                    if (isChecked) {
                      if (activeCodes.length <= 1) return;
                      next = activeCodes.filter((c) => c !== opt.value);
                    } else {
                      next = [...activeCodes, opt.value];
                    }
                    updateSetting(item.id, next.join(','));
                  }}
                  class="w-full px-3.5 py-2.5 text-xs font-medium flex items-center gap-2.5 text-left cursor-pointer transition-colors duration-150 hover:bg-[var(--md-sys-color-state-hover)] {isChecked ? 'font-semibold' : ''}"
                  style="color: var(--md-sys-color-on-surface);"
                >
                  <span
                    class="w-4 h-4 rounded-[4px] shrink-0 flex items-center justify-center transition-colors"
                    style={isChecked
                      ? 'background-color: var(--md-sys-color-primary); color: var(--md-sys-color-on-primary);'
                      : 'border: 1.5px solid var(--md-sys-color-outline);'}
                  >
                    {#if isChecked}
                      <Icon name="check" size={11} weight="bold" />
                    {/if}
                  </span>
                  <span class="truncate">{opt.label}</span>
                </button>
              {/each}
            </div>
          {/if}
        </div>
      {:else if item.widget === 'combobox'}
        {@const minV = item.min ?? 0}
        {@const maxV = item.max ?? 3600}
        {@const unitLabel = tr(lang, 'nav.secondsUnit')}
        <div class="relative">
          <div
            class="h-10 min-w-[168px] pl-3.5 pr-1.5 rounded-lg text-xs font-medium flex items-center justify-between gap-1.5 transition-colors duration-150"
            style="background-color: var(--md-sys-color-surface-container-highest); color: var(--md-sys-color-on-surface); border: 1px solid {isOpen ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-outline-variant)'};"
          >
            <input
              type="number"
              min={minV}
              max={maxV}
              value={Number(val) || 0}
              aria-label={item.title}
              onclick={(e) => e.stopPropagation()}
              onchange={(e) => {
                const raw = Number((e.target as HTMLInputElement).value);
                const clamped = Number.isFinite(raw) ? Math.min(maxV, Math.max(minV, Math.round(raw))) : minV;
                updateSetting(item.id, String(clamped));
              }}
              onkeydown={(e) => {
                if (e.key === 'Enter') {
                  (e.target as HTMLInputElement).blur();
                  openSelectId = null;
                }
              }}
              class="w-12 bg-transparent text-xs font-semibold tabular-nums outline-none [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
              style="color: var(--md-sys-color-on-surface);"
            />
            <button
              type="button"
              aria-haspopup="listbox"
              aria-expanded={isOpen}
              aria-label={item.title}
              onclick={(e) => {
                e.stopPropagation();
                if (isOpen) {
                  openSelectId = null;
                } else {
                  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
                  openSelectFlipUp = window.innerHeight - rect.bottom < 220;
                  openSelectId = item.id;
                }
              }}
              class="flex-1 h-7 px-1.5 flex items-center justify-end gap-2 cursor-pointer"
              style="color: var(--md-sys-color-on-surface-variant);"
            >
              <span class="text-[11px] font-medium truncate">
                {Number(val) === 0 ? tr(lang, 'opt.off') : unitLabel}
              </span>
              <span class="transition-transform duration-200 {isOpen ? 'rotate-180' : ''} shrink-0">
                <Icon name="caret-down" size={14} weight="bold" />
              </span>
            </button>
          </div>

          {#if isOpen}
            <div
              role="listbox"
              aria-label={item.title}
              in:fly={{ y: openSelectFlipUp ? 6 : -6, duration: 150 }}
              class="absolute right-0 {openSelectFlipUp ? 'bottom-full mb-1.5' : 'top-full mt-1.5'} min-w-[168px] max-h-64 overflow-y-auto py-1.5 rounded-xl shadow-xl z-40"
              style="background-color: var(--md-sys-color-surface-container-high); border: 1px solid var(--md-sys-color-outline-variant);"
            >
              {#each item.options || [] as opt (opt.value)}
                {@const isSelected = opt.value === String(Number(val) || 0)}
                {@const optLabel = opt.value === '0' ? tr(lang, 'opt.off') : `${opt.value} ${unitLabel}`}
                <button
                  type="button"
                  role="option"
                  aria-selected={isSelected}
                  onclick={(e) => {
                    e.stopPropagation();
                    updateSetting(item.id, opt.value);
                    openSelectId = null;
                  }}
                  class="w-full px-3.5 py-2.5 text-xs font-medium flex items-center justify-between gap-3 text-left cursor-pointer transition-colors duration-150 {isSelected ? 'font-semibold' : 'hover:bg-[var(--md-sys-color-state-hover)]'}"
                  style={isSelected
                    ? 'background-color: var(--md-sys-color-primary); color: var(--md-sys-color-on-primary);'
                    : 'color: var(--md-sys-color-on-surface);'}
                >
                  <span class="truncate">{optLabel}</span>
                  {#if isSelected}
                    <Icon name="check" size={14} weight="bold" />
                  {/if}
                </button>
              {/each}
            </div>
          {/if}
        </div>
      {:else if item.widget === 'slider'}
        {@const minV = item.min ?? 0}
        {@const maxV = item.max ?? 100}
        {@const rawNum = Number(val)}
        {@const numVal = Number.isFinite(rawNum) ? Math.min(maxV, Math.max(minV, rawNum)) : minV}
        {@const span = Math.max(1, maxV - minV)}
        {@const pct = ((numVal - minV) / span) * 100}
        {@const snaps = item.snapPoints ?? []}
        <div class="{snaps.length > 0 ? 'w-60' : 'w-44'} space-y-1.5">
          <div class="flex items-center {snaps.length > 0 ? 'justify-center' : 'justify-between'} text-[10px] font-medium" style="color: var(--md-sys-color-on-surface-variant);">
            {#if snaps.length > 0}
              <span class="tabular-nums text-[11px] font-bold" style="color: var(--md-sys-color-on-surface);">
                {numVal}{item.unit || ''}
              </span>
            {:else}
              <span>{item.startLabel || minV}</span>
              <span class="tabular-nums font-semibold" style="color: var(--md-sys-color-on-surface);">{val}</span>
              <span>{item.endLabel || maxV}</span>
            {/if}
          </div>

          <div class="relative flex items-center">
            <input
              type="range"
              min={minV}
              max={maxV}
              step={item.step ?? 1}
              value={numVal}
              oninput={(e) => {
                const inputEl = e.target as HTMLInputElement;
                let next = Number(inputEl.value);
                if (snaps.length > 0) {
                  const snapMatch = snaps.find((sp) => Math.abs(next - sp) <= 3);
                  if (snapMatch !== undefined) {
                    next = snapMatch;
                    inputEl.value = String(next);
                  }
                }
                updateSetting(item.id, String(next));
              }}
              class="velora-range {snaps.length > 0 ? 'velora-range-blue' : ''} w-full relative z-10"
              style="background: linear-gradient(to right, var(--md-sys-color-tertiary) 0%, var(--md-sys-color-tertiary) {pct}%, var(--md-sys-color-surface-container-highest) {pct}%, var(--md-sys-color-surface-container-highest) 100%);"
            />
          </div>

          {#if snaps.length > 0}
            <div class="relative h-6 select-none">
              {#each snaps as sp (sp)}
                {@const spPct = ((sp - minV) / span) * 100}
                {@const isCurrent = numVal === sp}
                {@const isPassed = numVal >= sp}
                <button
                  type="button"
                  onclick={() => updateSetting(item.id, String(sp))}
                  class="absolute top-0 -translate-x-1/2 flex flex-col items-center gap-0.5 cursor-pointer group"
                  style="left: calc({spPct}% + {7 - spPct * 0.14}px);"
                  title="{sp}{item.unit || ''}"
                >
                  <span
                    class="w-[1.5px] h-2 rounded-full transition-colors"
                    style="background-color: {isPassed ? 'var(--md-sys-color-tertiary)' : 'var(--md-sys-color-outline)'};"
                  ></span>
                  <span
                    class="text-[9px] tabular-nums transition-colors {isCurrent ? 'font-bold' : 'font-medium opacity-70 group-hover:opacity-100'}"
                    style="color: {isCurrent ? 'var(--md-sys-color-on-surface)' : 'var(--md-sys-color-on-surface-variant)'};"
                  >
                    {sp}{item.unit || ''}
                  </span>
                </button>
              {/each}
            </div>
          {/if}
        </div>
      {:else if item.widget === 'number-input'}
        <div class="flex items-center gap-1.5">
          <input
            type="number"
            min={item.min}
            max={item.max}
            step={item.step}
            value={Number(val)}
            onchange={(e) => updateSetting(item.id, (e.target as HTMLInputElement).value)}
            class="m3-input w-24 h-9 px-3 rounded-md text-xs font-semibold text-right tabular-nums outline-none"
          />
          {#if item.unit}
            <span class="text-xs font-medium" style="color: var(--md-sys-color-on-surface-variant);">{item.unit}</span>
          {/if}
        </div>
      {:else if item.widget === 'info'}
        {@const jamDisabled = item.id === 'integrations.jam.remoteUrl' && getPrefValue(effectivePrefs, 'integrations.jam.enabled') !== 'true'}
        {@const infoVal = jamDisabled ? 'http://xxx.xxx.xxx.xxx:8899/jam' : val}
        <div class="flex items-center gap-1.5">
          <code
            class="px-3 py-1.5 rounded-md text-[11px] font-mono min-w-[160px] min-h-[28px] inline-flex items-center {jamDisabled ? 'opacity-45 select-none' : 'select-all'}"
            style="background-color: var(--md-sys-color-surface-container-highest); color: var(--md-sys-color-on-surface);"
          >
            {infoVal}
          </code>
          <button
            type="button"
            disabled={jamDisabled || !infoVal}
            onclick={() => !jamDisabled && copyInfoValue(item.id, infoVal)}
            title={tr(lang, 'nav.copy')}
            class="m3-icon-btn-tonal w-8 h-8 flex items-center justify-center {jamDisabled || !infoVal ? 'opacity-40 cursor-not-allowed' : 'cursor-pointer'}"
          >
            <Icon name={copiedKey === item.id ? 'check' : 'copy'} weight="bold" size={14} />
          </button>
        </div>
      {:else}
        <input
          type="text"
          value={val}
          onchange={(e) => updateSetting(item.id, (e.target as HTMLInputElement).value)}
          class="m3-input h-9 px-3.5 rounded-md text-xs outline-none"
        />
      {/if}
      </div>
    </div>
  </div>
{/snippet}

{#if mode === 'history'}
  <div
    class="relative h-full w-full overflow-y-auto rounded-xl select-none flex flex-col"
    style="background-color: var(--md-sys-color-surface-container-low);"
    onclick={() => (openHistoryActionId = null)}
    role="presentation"
  >
    <section
      class="relative z-10 px-6 sm:px-7 pt-7 pb-7"
      style="background: linear-gradient(180deg, var(--md-sys-color-surface-container-highest) 0%, color-mix(in srgb, var(--md-sys-color-surface-container-high) 74%, var(--md-sys-color-surface-container-low)) 100%);"
    >
      <div class="flex flex-wrap items-end justify-between gap-5">
        <div class="flex flex-wrap items-end gap-5">
          <div
            class="w-28 h-28 rounded-lg shrink-0 shadow-[0_12px_36px_rgba(0,0,0,0.42)] flex items-center justify-center"
            style="background: linear-gradient(135deg, color-mix(in srgb, var(--md-sys-color-secondary) 55%, var(--md-sys-color-primary-container)) 0%, var(--md-sys-color-secondary) 55%, var(--md-sys-color-on-secondary-container) 100%);"
          >
            <Icon name="clock-counter-clockwise" weight="bold" size={44} class="text-white drop-shadow-sm" />
          </div>
          <div class="space-y-1.5 min-w-[200px] mb-[4px]">
            <div class="text-[11px] font-bold uppercase tracking-wider text-[var(--md-sys-color-secondary)]">
              Playback Archive
            </div>
            <h1 class="text-3xl font-bold tracking-tight text-[var(--md-sys-color-on-surface)]">{tr(lang, 'nav.history')}</h1>
            <p class="text-xs text-[var(--md-sys-color-on-surface-variant)] tabular-nums">
              {history.length} {tr(lang, 'nav.recentStreams')} • {formatListeningDuration(listeningStats.totalMinutes)}
            </p>
          </div>
        </div>

        <div class="mb-[4px] flex items-center gap-2.5">
          <button
            type="button"
            onclick={playAllHistory}
            disabled={history.length === 0}
            class="h-9 min-w-[116px] px-4 rounded-full text-xs font-bold flex items-center justify-center gap-1.5 cursor-pointer transition-transform hover:scale-[1.03] active:scale-95 disabled:opacity-40"
            style="background-color: var(--md-sys-color-tertiary); color: var(--md-sys-color-on-tertiary);"
          >
            <Icon name="play" weight="fill" size={14} /> {tr(lang, 'nav.playAll')}
          </button>
          {#if history.length > 0}
            <button
              type="button"
              onclick={() => (confirmClearHistory = true)}
              class="h-9 min-w-[116px] px-4 rounded-full text-xs font-bold text-rose-300 bg-rose-500/15 hover:bg-rose-500/25 flex items-center justify-center gap-1.5 cursor-pointer transition-all duration-150 hover:scale-[1.03] active:scale-95"
            >
              <Icon name="trash" size={14} /> {tr(lang, 'nav.clear')}
            </button>
          {/if}
        </div>
      </div>
    </section>

    <div
      class="relative z-10 px-6 sm:px-8 pt-5 pb-10 flex-1 flex flex-col gap-5"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-surface-container-high) 48%, var(--md-sys-color-surface-container-low)) 0%, color-mix(in srgb, var(--md-sys-color-surface-container-high) 24%, var(--md-sys-color-surface-container-low)) 140px, var(--md-sys-color-surface-container-low) 320px);"
    >

      {#if history.length === 0}
        <EmptyState
          icon="music-note"
          iconWeight="bold"
          iconSize={38}
          title={tr(lang, 'nav.noHistoryTitle')}
          description={tr(lang, 'nav.noHistoryDesc')}
        >
          <button
            type="button"
            onclick={() => onNavigate?.({ name: 'home' })}
            class="mt-6 h-10 px-6 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
            style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
          >
            {tr(lang, 'nav.exploreSongs')}
          </button>
        </EmptyState>
      {:else}
        <div class="space-y-1">
          <div
            class="-mx-2 grid grid-cols-[32px_minmax(0,1fr)_52px_96px] md:grid-cols-[32px_minmax(0,1.6fr)_minmax(100px,0.6fr)_minmax(150px,0.75fr)_56px_96px] items-center gap-4 px-3 py-2 text-[11px] font-semibold uppercase tracking-wider"
            style="color: var(--md-sys-color-on-surface-variant);"
          >
            <span class="text-center">#</span>
            <span>{tr(lang, 'nav.titleCol')}</span>
            <span class="hidden md:block">{tr(lang, 'nav.listenedCol')}</span>
            <span class="hidden md:block">{tr(lang, 'nav.watchedCol')}</span>
            <span class="flex justify-end pr-1">
              <Icon name="clock" size={14} />
            </span>
            <span></span>
          </div>

          <div class="space-y-0.5" role="list">
            {#each history as item, idx (item.id)}
              {@const trItem = historyItemToTrack(item)}
              {@const cover = resolveCoverUrl(trItem.coverUrl || item.coverUrl, mediaBaseUrl)}
              {@const active = activeHistoryEntryId === item.id}
              {@const menuOpen = openHistoryActionId === item.id}
              {@const dlState = downloadStates[trItem.id]}
              {@const isDlActive = dlState && (dlState.status === 'queued' || dlState.status === 'downloading')}
              <div
                role="listitem"
                class="velora-track-row group -mx-2 grid grid-cols-[32px_minmax(0,1fr)_52px_96px] md:grid-cols-[32px_minmax(0,1.6fr)_minmax(100px,0.6fr)_minmax(150px,0.75fr)_56px_96px] items-center gap-4 px-3 py-2 rounded-lg transition-colors duration-150 {active
                  ? 'bg-[var(--md-sys-color-state-hover)]'
                  : 'hover:bg-[var(--md-sys-color-state-hover)]'}"
                style="color: var(--md-sys-color-on-surface);"
              >
                <div class="flex items-center justify-center">
                  {#if active && isPlaying}
                    <EqIndicator class="group-hover:hidden" />
                    <button
                      type="button"
                      onclick={() => (onTogglePlay ? onTogglePlay() : playHistoryTrack(item))}
                      aria-label="{tr(lang, 'nav.pause')} {item.title}"
                      class="hidden group-hover:flex items-center justify-center cursor-pointer text-[var(--md-sys-color-tertiary)]"
                    >
                      <Icon name="pause" weight="fill" size={14} />
                    </button>
                  {:else if active && !isPlaying}
                    <button
                      type="button"
                      onclick={() => (onTogglePlay ? onTogglePlay() : playHistoryTrack(item))}
                      aria-label="{tr(lang, 'nav.play')} {item.title}"
                      class="flex items-center justify-center cursor-pointer text-[var(--md-sys-color-tertiary)]"
                    >
                      <Icon name="play" weight="fill" size={14} />
                    </button>
                  {:else}
                    <span class="text-xs font-medium tabular-nums group-hover:hidden opacity-65">
                      {idx + 1}
                    </span>
                    <button
                      type="button"
                      onclick={() => playHistoryTrack(item)}
                      aria-label="{tr(lang, 'nav.play')} {item.title}"
                      class="hidden group-hover:flex items-center justify-center cursor-pointer text-[var(--md-sys-color-tertiary)]"
                    >
                      <Icon name="play" weight="fill" size={14} />
                    </button>
                  {/if}
                </div>

                <div class="flex items-center gap-3 min-w-0 text-left">
                  <button
                    type="button"
                    onclick={() => (active && onTogglePlay ? onTogglePlay() : playHistoryTrack(item))}
                    aria-label="{tr(lang, 'nav.play')} {item.title}"
                    class="shrink-0 cursor-pointer"
                  >
                    <div class="w-10 h-10 rounded-md overflow-hidden shrink-0">
                      <CoverImage src={cover} fallbackIcon="vinyl-record" iconSize={16} />
                    </div>
                  </button>
                  <div class="min-w-0 flex-1">
                    {#if onOpenTrackAlbum}
                      <button
                        type="button"
                        onclick={() => onOpenTrackAlbum(trItem)}
                        class="block max-w-full text-left text-xs font-bold line-clamp-2 whitespace-normal leading-snug hover:underline cursor-pointer {active ? 'text-[var(--md-sys-color-primary)]' : ''}"
                      >
                        {item.title}
                      </button>
                    {:else}
                      <button
                        type="button"
                        onclick={() => (active && onTogglePlay ? onTogglePlay() : playHistoryTrack(item))}
                        class="block max-w-full text-left text-xs font-bold line-clamp-2 whitespace-normal leading-snug cursor-pointer {active ? 'text-[var(--md-sys-color-primary)]' : ''}"
                      >
                        {item.title}
                      </button>
                    {/if}
                    <div class="text-[11px] truncate opacity-75 mt-0.5">
                      {item.artistName}{item.albumTitle ? ` • ${item.albumTitle}` : ''}
                    </div>
                  </div>
                </div>

                <span class="hidden md:block text-xs tabular-nums opacity-70 truncate">
                  {formatDuration(item.durationListenedSec)}
                </span>

                <span class="hidden md:block text-xs tabular-nums opacity-70 truncate" title={item.playedAt}>
                  {formatWatchedAt(item.playedAt)}
                </span>

                <span class="text-right text-xs tabular-nums opacity-70 pr-1">
                  {formatDuration(item.durationSec)}
                </span>

                <div class="flex items-center justify-end gap-1 relative">
                  {#if onDownloadTrack || onRemoveDownload}
                    {#if isDlActive}
                      <span
                        class="inline-flex items-center justify-center w-7 h-7 text-[10px] font-bold tabular-nums text-[var(--md-sys-color-primary)]"
                        title="{tr(lang, 'nav.downloading')} ({Math.round(dlState.progress || 0)}%)"
                      >
                        {Math.round(dlState.progress || 0)}%
                      </span>
                    {:else if trItem.isDownloaded}
                      <button
                        type="button"
                        onclick={() => onRemoveDownload?.(trItem.id)}
                        title={tr(lang, 'nav.savedOfflineRemoveHint')}
                        aria-label="{tr(lang, 'nav.removeDownload')} {item.title}"
                        class="m3-icon-btn w-7 h-7 flex items-center justify-center transition cursor-pointer text-[var(--md-sys-color-primary)] opacity-90 hover:opacity-100"
                      >
                        <Icon name="check-circle" weight="fill" size={15} />
                      </button>
                    {:else if onDownloadTrack}
                      <button
                        type="button"
                        onclick={() => onDownloadTrack(trItem)}
                        title={tr(lang, 'nav.downloadOffline')}
                        aria-label="{tr(lang, 'nav.downloadOffline')} {item.title}"
                        class="m3-icon-btn w-7 h-7 flex items-center justify-center opacity-0 group-hover:opacity-100 transition cursor-pointer"
                      >
                        <Icon name="download" size={15} />
                      </button>
                    {/if}
                  {/if}

                  {#if onToggleFavorite}
                    <button
                      type="button"
                      onclick={() => onToggleFavorite(trItem.id)}
                      aria-label="{tr(lang, 'nav.likeSong')} {item.title}"
                      class="m3-icon-btn velora-heart-btn w-7 h-7 flex items-center justify-center cursor-pointer {trItem.isFavorite
                        ? 'is-loved opacity-100 text-[var(--md-sys-color-tertiary)]'
                        : 'opacity-0 group-hover:opacity-100'}"
                    >
                      <Icon name="heart" weight={trItem.isFavorite ? 'fill' : 'regular'} size={15} />
                    </button>
                  {/if}
                  <TrackActionMenu
                    track={trItem}
                    open={menuOpen}
                    {playlists}
                    {lang}
                    triggerClass={menuOpen ? 'opacity-100' : 'opacity-0 group-hover:opacity-100 transition'}
                    onToggleOpen={() => (openHistoryActionId = menuOpen ? null : item.id)}
                    onClose={() => (openHistoryActionId = null)}
                    {onPlayNext}
                    {onAddToQueue}
                    {onAddToPlaylist}
                    {onToggleFavorite}
                    {onDownloadTrack}
                    {onRemoveDownload}
                    {onOpenArtist}
                    {onOpenTrackAlbum}
                    {onCopyLink}
                  />
                </div>
              </div>
            {/each}
          </div>
        </div>
      {/if}
    </div>
  </div>
{:else if mode === 'listening-stats'}
  <div
    class="relative h-full w-full overflow-y-auto rounded-xl select-none flex flex-col"
    style="background-color: var(--md-sys-color-surface-container-low);"
  >
    <section
      class="relative z-10 px-6 sm:px-8 pt-6 pb-6"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-primary) 24%, var(--md-sys-color-primary-container)) 0%, color-mix(in srgb, var(--md-sys-color-primary-container) 56%, var(--md-sys-color-surface-container-low)) 100%);"
    >
      <h1 class="text-3xl font-bold tracking-tight text-[var(--md-sys-color-on-surface)] mb-5">
        {tr(lang, 'nav.stats')}
      </h1>
      {#if statsLoading}
        <div class="grid grid-cols-2 lg:grid-cols-4 gap-3">
          {#each [0, 1, 2, 3] as skIdx (`stat-card-sk-${skIdx}`)}
            <div class="p-4 rounded-xl bg-[var(--md-sys-color-surface-container)]/90 space-y-2.5 shadow-xs animate-pulse">
              <div class="h-3 w-24 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
              <div class="h-7 w-20 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
              <div class="h-2.5 w-16 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
            </div>
          {/each}
        </div>
      {:else}
        <div class="grid grid-cols-2 lg:grid-cols-4 gap-3">
          <div class="p-4 rounded-xl bg-[var(--md-sys-color-surface-container)]/90 space-y-1 shadow-xs">
            <div class="text-[11px] font-semibold" style="color: var(--md-sys-color-on-surface-variant);">
              Waktu dengar
            </div>
            <div class="text-2xl font-bold tabular-nums" style="color: var(--md-sys-color-on-surface);">
              {formatListeningDuration(displayedStats.totalMinutes)}
            </div>
            <div class="text-[11px] opacity-75" style="color: var(--md-sys-color-on-surface-variant);">
              total
            </div>
          </div>

          <div class="p-4 rounded-xl bg-[var(--md-sys-color-surface-container)]/90 space-y-1 shadow-xs">
            <div class="text-[11px] font-semibold" style="color: var(--md-sys-color-on-surface-variant);">
              Putaran
            </div>
            <div class="text-2xl font-bold tabular-nums" style="color: var(--md-sys-color-on-surface);">
              {displayedStats.totalPlays}
            </div>
            <div class="text-[11px] opacity-75" style="color: var(--md-sys-color-on-surface-variant);">
              kali diputar
            </div>
          </div>

          <div class="p-4 rounded-xl bg-[var(--md-sys-color-surface-container)]/90 space-y-1 shadow-xs">
            <div class="text-[11px] font-semibold" style="color: var(--md-sys-color-on-surface-variant);">
              Lagu berbeda
            </div>
            <div class="text-2xl font-bold tabular-nums" style="color: var(--md-sys-color-on-surface);">
              {displayedStats.uniqueTracks}
            </div>
            <div class="text-[11px] opacity-75" style="color: var(--md-sys-color-on-surface-variant);">
              lagu
            </div>
          </div>

          <div class="p-4 rounded-xl bg-[var(--md-sys-color-surface-container)]/90 space-y-1 shadow-xs">
            <div class="text-[11px] font-semibold" style="color: var(--md-sys-color-on-surface-variant);">
              Hari aktif
            </div>
            <div class="text-2xl font-bold tabular-nums" style="color: var(--md-sys-color-on-surface);">
              {displayedStats.activeDays ?? 0}
            </div>
            <div class="text-[11px] opacity-75" style="color: var(--md-sys-color-on-surface-variant);">
              hari ada yang diputar
            </div>
          </div>
        </div>
      {/if}
    </section>

    <div
      class="relative z-10 px-6 sm:px-8 pt-2 pb-10 flex-1 flex flex-col gap-5"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-primary-container) 30%, var(--md-sys-color-surface-container-low)) 0%, var(--md-sys-color-surface-container-low) 180px);"
    >
      {#if statsLoading}
        <section class="p-5 rounded-xl bg-[var(--md-sys-color-surface-container)] animate-pulse flex flex-col md:flex-row md:items-center justify-between gap-5">
          <div class="flex items-center gap-4">
            <div class="w-14 h-14 rounded-2xl bg-[var(--md-sys-color-surface-container-highest)] shrink-0"></div>
            <div class="space-y-2">
              <div class="h-3 w-20 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
              <div class="h-6 w-36 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
            </div>
          </div>
          <div class="flex-1 max-w-md grid grid-cols-3 gap-2.5">
            {#each [0, 1, 2] as fSk (`fire-sk-${fSk}`)}
              <div class="h-14 rounded-xl bg-[var(--md-sys-color-surface-container-highest)]"></div>
            {/each}
          </div>
        </section>

        <div class="grid grid-cols-1 lg:grid-cols-12 gap-4">
          <section class="lg:col-span-7 p-5 rounded-xl bg-[var(--md-sys-color-surface-container)] space-y-4 animate-pulse">
            <div class="flex items-center justify-between">
              <div class="h-3.5 w-28 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
              <div class="h-3.5 w-16 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
            </div>
            <div class="h-44 w-full rounded-lg bg-[var(--md-sys-color-surface-container-highest)]"></div>
          </section>
          <section class="lg:col-span-5 p-5 rounded-xl bg-[var(--md-sys-color-surface-container)] space-y-4 animate-pulse">
            <div class="flex items-center justify-between">
              <div class="h-3.5 w-32 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
              <div class="h-3.5 w-20 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
            </div>
            <div class="h-44 w-full rounded-lg bg-[var(--md-sys-color-surface-container-highest)]"></div>
          </section>
        </div>

        <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {#each [0, 1] as colIdx (`top-sk-${colIdx}`)}
            <section class="p-5 rounded-xl bg-[var(--md-sys-color-surface-container)] space-y-3 animate-pulse">
              <div class="h-3.5 w-28 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
              <div class="h-[248px] space-y-3 overflow-hidden">
                {#each [0, 1, 2, 3, 4] as rowIdx (`top-row-sk-${colIdx}-${rowIdx}`)}
                  <div class="flex items-center gap-3">
                    <div class="w-4 h-3 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
                    <div class="w-10 h-10 {colIdx === 1 ? 'rounded-full' : 'rounded-lg'} bg-[var(--md-sys-color-surface-container-highest)] shrink-0"></div>
                    <div class="flex-1 space-y-2">
                      <div class="flex justify-between gap-4">
                        <div class="h-3 w-36 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
                        <div class="h-3 w-16 rounded bg-[var(--md-sys-color-surface-container-highest)]"></div>
                      </div>
                      <div class="h-1.5 w-full rounded-full bg-[var(--md-sys-color-surface-container-highest)]"></div>
                    </div>
                  </div>
                {/each}
              </div>
            </section>
          {/each}
        </div>
      {:else if !selectedStatsDate && displayedStats.totalPlays === 0}
        <EmptyState
          icon="chart-bar"
          iconWeight="fill"
          iconSize={38}
          title={tr(lang, 'nav.noStatsTitle')}
          description={tr(lang, 'nav.noStatsDesc')}
        >
          <button
            type="button"
            onclick={() => onNavigate?.({ name: 'home', filter: 'musik' })}
            class="mt-6 h-10 px-6 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
            style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
          >
            {tr(lang, 'nav.exploreSongs')}
          </button>
        </EmptyState>
      {:else}
        <section
          class="relative overflow-hidden p-5 rounded-xl bg-[var(--md-sys-color-surface-container)] flex flex-col lg:flex-row lg:items-center justify-between gap-6"
        >
          <div class="flex items-center gap-4 min-w-0 flex-1">
            <div class="relative w-16 h-16 flex items-center justify-center shrink-0">
              <div
                class="velora-flame-glow absolute inset-0 rounded-full blur-md pointer-events-none"
                style="background: radial-gradient(circle, rgba(255,145,50,0.45) 0%, rgba(255,95,30,0) 72%);"
              ></div>
              <div
                class="velora-flame-glow absolute -bottom-0.5 left-1/2 -translate-x-1/2 w-11 h-2.5 rounded-full blur-xs pointer-events-none"
                style="background: radial-gradient(ellipse, rgba(255,160,60,0.8) 0%, rgba(255,90,20,0) 75%);"
              ></div>
              <svg viewBox="0 0 64 64" class="relative z-10 w-14 h-14 drop-shadow-[0_6px_12px_rgba(255,120,35,0.42)] overflow-visible" aria-hidden="true">
                <defs>
                  <linearGradient id="veloraFlameOuter" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stop-color="#FFD166" />
                    <stop offset="45%" stop-color="#FF7A2F" />
                    <stop offset="100%" stop-color="#D9381E" />
                  </linearGradient>
                  <linearGradient id="veloraFlameMid" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stop-color="#FFE89E" />
                    <stop offset="100%" stop-color="#FF9638" />
                  </linearGradient>
                  <linearGradient id="veloraFlameCore" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" stop-color="#FFFBE0" />
                    <stop offset="100%" stop-color="#FFD166" />
                  </linearGradient>
                </defs>
                <g class="velora-flame-anim">
                  <path
                    fill="url(#veloraFlameOuter)"
                    d="M34.5 5C34.5 5 49 17.5 49 34.5C49 46 41.2 55 31.8 55C21.8 55 14.5 46.5 14.5 35.8C14.5 26.8 19.4 20.2 23.4 16.5C23.1 21.8 25.5 25.2 27.8 26.3C26.9 19 30 11 34.5 5Z"
                  />
                  <path
                    fill="url(#veloraFlameMid)"
                    d="M33 17C33 17 43 26.2 43 37.5C43 45.8 38 52 31.6 52C24.8 52 20 46 20 38.2C20 31.5 23.8 26.8 26.6 24.4C26.4 28.2 28.1 30.6 29.8 31.4C29.1 26.2 30.8 21 33 17Z"
                  />
                  <path
                    fill="url(#veloraFlameCore)"
                    d="M32 30C32 30 37.5 35.5 37.5 41.8C37.5 46.8 34.8 50 31.4 50C27.8 50 25 46.8 25 42C25 38 27.2 34.8 29 33.4C28.9 35.5 29.9 36.8 30.8 37.3C30.4 34.5 31.1 32 32 30Z"
                  />
                </g>
                <circle class="velora-ember-1" cx="22" cy="24" r="1.8" fill="#FFD166" />
                <circle class="velora-ember-2" cx="42" cy="21" r="1.5" fill="#FF9F43" />
                <circle class="velora-ember-3" cx="32" cy="14" r="1.4" fill="#FFE89E" />
              </svg>
            </div>

            <div class="min-w-0 flex-1 space-y-2">
              <div class="text-xs font-bold uppercase tracking-wider" style="color: var(--md-sys-color-on-surface-variant);">
                Percikan Api
              </div>
              <div class="flex items-baseline gap-2">
                <span class="text-3xl font-extrabold tabular-nums text-orange-400">{fireStreak.currentStreak}</span>
                <span class="text-xs font-medium" style="color: var(--md-sys-color-on-surface-variant);">
                  hari berturut-turut
                </span>
              </div>
              <div class="space-y-1 w-full">
                <div class="w-full h-2 rounded-full overflow-hidden bg-[var(--md-sys-color-surface-container-highest)]">
                  <div
                    class="h-full rounded-full transition-all duration-300"
                    style="width: {monthlyStreakStats.pct}%; background: linear-gradient(90deg, #ff7a2f, #ffb347);"
                  ></div>
                </div>
                <div class="flex items-center justify-between text-[10px] tabular-nums" style="color: var(--md-sys-color-on-surface-variant); opacity: 0.78;">
                  <span>Progres bulan ini</span>
                  <span>{monthlyStreakStats.metDays}/{monthlyStreakStats.daysInMonth} hari</span>
                </div>
              </div>
            </div>
          </div>

          <div class="grid grid-cols-3 gap-2.5 lg:w-[370px] shrink-0">
            <div class="p-3 rounded-xl bg-[var(--md-sys-color-surface-container-high)] text-center space-y-0.5">
              <div class="text-base font-bold tabular-nums" style="color: var(--md-sys-color-on-surface);">
                {fireStreak.longestStreak}
              </div>
              <div class="text-[11px]" style="color: var(--md-sys-color-on-surface-variant);">
                Terpanjang
              </div>
            </div>
            <div class="p-3 rounded-xl bg-[var(--md-sys-color-surface-container-high)] text-center space-y-0.5">
              <div class="text-base font-bold tabular-nums" style="color: var(--md-sys-color-on-surface);">
                {listeningStats.activeDays ?? 0}
              </div>
              <div class="text-[11px]" style="color: var(--md-sys-color-on-surface-variant);">
                Total hari
              </div>
            </div>
            <div class="p-3 rounded-xl bg-[var(--md-sys-color-surface-container-high)] text-center space-y-0.5">
              <div class="text-base font-bold tabular-nums" style="color: var(--md-sys-color-on-surface);">
                {monthlyStreakStats.remainingRecoveries}/3
              </div>
              <div class="text-[11px]" style="color: var(--md-sys-color-on-surface-variant);">
                Pemulihan
              </div>
            </div>
          </div>
        </section>

        <div class="grid grid-cols-1 lg:grid-cols-12 gap-4">
          <section class="lg:col-span-7 p-5 rounded-xl bg-[var(--md-sys-color-surface-container)] space-y-3">
            <div class="flex items-center justify-between">
              <h2 class="text-xs font-bold uppercase tracking-wider" style="color: var(--md-sys-color-on-surface-variant);">
                {selectedStatsDate ? selectedStatsDate : 'Hari ini'}
              </h2>
              <span class="text-xs font-semibold tabular-nums" style="color: var(--md-sys-color-primary);">
                {formatListeningDuration(todayHourlySeries.totalMin)}
              </span>
            </div>
            <StatsChart
              type="line"
              labels={todayHourlySeries.labels}
              values={todayHourlySeries.values}
              secondaryValues={todayHourlySeries.plays}
              unitLabel={tr(lang, 'nav.minutesShort')}
              secondaryUnitLabel={tr(lang, 'nav.playsLower')}
              highlightIndex={todayHourlySeries.peakIdx}
            />
          </section>

          <section class="lg:col-span-5 p-5 rounded-xl bg-[var(--md-sys-color-surface-container)] space-y-3">
            <div class="flex items-center justify-between gap-2">
              <div class="relative">
                <button
                  type="button"
                  aria-haspopup="listbox"
                  aria-expanded={openSelectId === 'stats-chart-period'}
                  onclick={(e) => {
                    e.stopPropagation();
                    openSelectFlipUp = false;
                    openSelectId = openSelectId === 'stats-chart-period' ? null : 'stats-chart-period';
                  }}
                  class="h-8 min-w-[132px] px-3 rounded-lg text-xs font-semibold flex items-center justify-between gap-2 cursor-pointer transition-colors duration-150 hover:bg-[var(--md-sys-color-state-hover)]"
                  style="background-color: var(--md-sys-color-surface-container-highest); color: var(--md-sys-color-on-surface); border: 1px solid {openSelectId === 'stats-chart-period' ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-outline-variant)'};"
                >
                  <span class="truncate">{(CHART_PERIOD_OPTIONS.find((o) => o.value === chartPeriod) ?? CHART_PERIOD_OPTIONS[0]).label}</span>
                  <span class="transition-transform duration-200 {openSelectId === 'stats-chart-period' ? 'rotate-180' : ''}" style="color: var(--md-sys-color-on-surface-variant);">
                    <Icon name="caret-down" size={13} weight="bold" />
                  </span>
                </button>
                {#if openSelectId === 'stats-chart-period'}
                  <div
                    role="listbox"
                    class="absolute left-0 top-full mt-1.5 min-w-[148px] p-1 rounded-lg shadow-xl border z-40"
                    style="background-color: var(--md-sys-color-surface-container-high); border-color: var(--md-sys-color-outline-variant);"
                  >
                    {#each CHART_PERIOD_OPTIONS as opt (opt.value)}
                      {@const selected = chartPeriod === opt.value}
                      <button
                        type="button"
                        role="option"
                        aria-selected={selected}
                        onclick={() => {
                          chartPeriod = opt.value;
                          openSelectId = null;
                        }}
                        class="w-full px-3 py-2 rounded-md text-xs text-left flex items-center justify-between cursor-pointer transition-colors {selected ? 'font-semibold' : 'hover:bg-[var(--md-sys-color-state-hover)]'}"
                        style={selected
                          ? 'background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);'
                          : 'color: var(--md-sys-color-on-surface);'}
                      >
                        <span>{opt.label}</span>
                        {#if selected}
                          <Icon name="check" size={13} weight="bold" />
                        {/if}
                      </button>
                    {/each}
                  </div>
                {/if}
              </div>
              <span class="text-xs font-semibold tabular-nums" style="color: var(--md-sys-color-primary);">
                {formatListeningDuration(periodRangeSeries.totalMin)}
              </span>
            </div>
            <StatsChart
              type="bar"
              labels={periodRangeSeries.labels}
              values={periodRangeSeries.values}
              secondaryValues={periodRangeSeries.plays}
              unitLabel={tr(lang, 'nav.minutesShort')}
              secondaryUnitLabel={tr(lang, 'nav.playsLower')}
              highlightIndex={periodRangeSeries.highlightIndex}
            />
          </section>
        </div>

        <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <section class="p-5 rounded-xl bg-[var(--md-sys-color-surface-container)] space-y-3">
            <h2 class="text-xs font-bold uppercase tracking-wider" style="color: var(--md-sys-color-on-surface-variant);">
              {tr(lang, 'nav.topTracks')}
            </h2>
            <div class="h-[248px] overflow-y-auto pr-1">
              {#if (displayedStats.topTracks ?? []).length === 0}
                <EmptyState
                  icon="music-note"
                  iconSize={28}
                  title={tr(lang, 'nav.noStatsTitle')}
                  description={selectedStatsDate || tr(lang, 'nav.noStatsDesc')}
                  titleClass="text-sm font-semibold"
                  descClass="mt-1 text-xs max-w-xs mx-auto opacity-75"
                  class="h-full flex flex-col items-center justify-center text-center px-4"
                />
              {:else}
                <div class="space-y-3">
                  {#each displayedStats.topTracks ?? [] as item, idx (item.name + idx)}
                    {@const pct = Math.max(8, Math.round((item.plays / maxTopTrackPlays) * 100))}
                    {@const imgSrc = resolveCoverUrl(item.imageUrl ?? '', mediaBaseUrl)}
                    <div class="flex items-center gap-3">
                      <span class="w-4 text-xs font-bold tabular-nums opacity-55 shrink-0 text-center">{idx + 1}</span>
                      <div class="w-10 h-10 rounded-lg overflow-hidden shrink-0 bg-[var(--md-sys-color-surface-container-highest)] flex items-center justify-center">
                        {#if imgSrc}
                          <img src={imgSrc} alt={item.name} class="w-full h-full object-cover" />
                        {:else}
                          <Icon name="music-note" size={16} class="opacity-50" />
                        {/if}
                      </div>
                      <div class="min-w-0 flex-1 space-y-1.5">
                        <div class="flex items-center justify-between gap-3 text-xs">
                          <div class="min-w-0 flex items-center gap-1.5">
                            <span class="font-bold truncate" style="color: var(--md-sys-color-on-surface);">{item.name}</span>
                            {#if item.subText}
                              <span class="text-[11px] truncate opacity-65">• {item.subText}</span>
                            {/if}
                          </div>
                          <span class="text-[11px] font-semibold tabular-nums shrink-0" style="color: var(--md-sys-color-on-surface-variant);">
                            {item.plays}×
                          </span>
                        </div>
                        <div class="h-1.5 w-full rounded-full overflow-hidden bg-[var(--md-sys-color-surface-container-highest)]">
                          <div
                            class="h-full rounded-full transition-all duration-300"
                            style="width: {pct}%; background-color: {idx === 0 ? 'var(--md-sys-color-tertiary)' : 'var(--md-sys-color-primary)'};"
                          ></div>
                        </div>
                      </div>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
          </section>

          <section class="p-5 rounded-xl bg-[var(--md-sys-color-surface-container)] space-y-3">
            <h2 class="text-xs font-bold uppercase tracking-wider" style="color: var(--md-sys-color-on-surface-variant);">
              {tr(lang, 'nav.topArtists')}
            </h2>
            <div class="h-[248px] overflow-y-auto pr-1">
              {#if (displayedStats.topArtists ?? []).length === 0}
                <EmptyState
                  icon="user"
                  iconSize={28}
                  title={tr(lang, 'nav.noStatsTitle')}
                  description={selectedStatsDate || tr(lang, 'nav.noStatsDesc')}
                  titleClass="text-sm font-semibold"
                  descClass="mt-1 text-xs max-w-xs mx-auto opacity-75"
                  class="h-full flex flex-col items-center justify-center text-center px-4"
                />
              {:else}
                <div class="space-y-3">
                  {#each displayedStats.topArtists ?? [] as item, idx (item.name + idx)}
                    {@const pct = Math.max(8, Math.round((item.plays / maxTopArtistPlays) * 100))}
                    {@const imgSrc = resolveCoverUrl(item.imageUrl ?? '', mediaBaseUrl)}
                    <div class="flex items-center gap-3">
                      <span class="w-4 text-xs font-bold tabular-nums opacity-55 shrink-0 text-center">{idx + 1}</span>
                      <div class="w-10 h-10 rounded-full overflow-hidden shrink-0 bg-[var(--md-sys-color-surface-container-highest)] flex items-center justify-center">
                        {#if imgSrc}
                          <img src={imgSrc} alt={item.name} class="w-full h-full object-cover" />
                        {:else}
                          <Icon name="user" size={16} class="opacity-50" />
                        {/if}
                      </div>
                      <div class="min-w-0 flex-1 space-y-1.5">
                        <div class="flex items-center justify-between gap-3 text-xs">
                          <span class="font-bold truncate" style="color: var(--md-sys-color-on-surface);">{item.name}</span>
                          <span class="text-[11px] font-semibold tabular-nums shrink-0" style="color: var(--md-sys-color-on-surface-variant);">
                            {item.plays}×
                          </span>
                        </div>
                        <div class="h-1.5 w-full rounded-full overflow-hidden bg-[var(--md-sys-color-surface-container-highest)]">
                          <div
                            class="h-full rounded-full transition-all duration-300"
                            style="width: {pct}%; background-color: {idx === 0 ? 'var(--md-sys-color-tertiary)' : 'var(--md-sys-color-primary)'};"
                          ></div>
                        </div>
                      </div>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
          </section>
        </div>
      {/if}
    </div>
  </div>
{:else if mode === 'fire-streak'}
  <div class="p-6 space-y-6">
    <div class="m3-card-elevated p-6 rounded-lg flex flex-wrap items-center justify-between gap-4">
      <div class="flex items-center gap-4">
        <div
          class="w-14 h-14 rounded-md flex items-center justify-center"
          style="background-color: var(--md-sys-color-tertiary-container); color: var(--md-sys-color-on-tertiary-container);"
        >
          <Icon name="flame" weight="fill" size={30} class="text-orange-400" />
        </div>
        <div>
          <h1 class="text-2xl font-semibold tracking-tight" style="color: var(--md-sys-color-on-surface);">
            {tr(lang, 'nav.dailyStreak')}
          </h1>
          <p class="text-xs mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
            {tr(lang, 'nav.streakTargetDesc')} {fireStreak.goalMinutes} {tr(lang, 'nav.minutes')} {tr(lang, 'nav.orWord')} {fireStreak.goalTracks} {tr(lang, 'nav.songs').toLowerCase()} {tr(lang, 'nav.perDay')}
          </p>
        </div>
      </div>
      <div class="flex items-center gap-6">
        <div class="text-center">
          <div class="text-2xl font-semibold text-orange-400 tabular-nums">{fireStreak.currentStreak}</div>
          <div class="text-[11px]" style="color: var(--md-sys-color-on-surface-variant);">{tr(lang, 'nav.activeStreak')}</div>
        </div>
        <div class="text-center">
          <div class="text-2xl font-semibold tabular-nums">{fireStreak.longestStreak}</div>
          <div class="text-[11px]" style="color: var(--md-sys-color-on-surface-variant);">{tr(lang, 'nav.record')}</div>
        </div>
      </div>
    </div>

    <div class="m3-card p-5 rounded-lg">
      <h2 class="text-sm font-semibold mb-3" style="color: var(--md-sys-color-on-surface);">
        {tr(lang, 'nav.streakCalendar')}
      </h2>
      <div class="grid grid-cols-10 gap-2">
        {#each fireStreak.last30Days as day (day.date)}
          <div
            title="{day.date} • {Math.round(day.secondsListened / 60)} menit • {day.tracksCount} lagu"
            class="h-9 rounded-md flex items-center justify-center text-xs font-semibold tabular-nums transition-colors"
            style={day.streakMet
              ? 'background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);'
              : 'background-color: var(--md-sys-color-surface-container-highest); color: var(--md-sys-color-on-surface-variant);'}
          >
            {day.date.slice(8, 10)}
          </div>
        {/each}
      </div>
    </div>
  </div>
{:else}
  <div class="h-full min-h-0 flex flex-col overflow-hidden relative">
    <div
      class="md:hidden shrink-0 px-4 pt-3 pb-2.5 space-y-2.5 border-b"
      style="background-color: var(--md-sys-color-surface-container-low); border-color: var(--md-sys-color-outline-variant);"
    >
      <div class="flex items-center justify-between gap-2">
        <div class="flex items-center gap-1.5 min-w-0">
          <button
            type="button"
            onclick={() => (onBackFromPreferences ? onBackFromPreferences() : onNavigate?.({ name: 'home' }))}
            title={tr(lang, 'nav.back')}
            aria-label={tr(lang, 'nav.back')}
            class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer shrink-0"
          >
            <Icon name="arrow-left" size={17} />
          </button>
          <span class="text-sm font-bold tracking-tight truncate" style="color: var(--md-sys-color-on-surface);">
            {tr(lang, 'nav.settings')}
          </span>
        </div>
      </div>

      <div class="relative">
        <span
          class="absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none"
          style="color: var(--md-sys-color-on-surface-variant);"
        >
          <Icon name="magnifying-glass" size={14} />
        </span>
        <input
          type="search"
          bind:value={prefSearch}
          placeholder={tr(lang, 'nav.searchSettings')}
          class="m3-input w-full h-9 pl-8 pr-3 rounded-xl text-xs outline-none"
        />
      </div>

      <div class="flex items-center gap-1.5 overflow-x-auto no-scrollbar pb-0.5">
        {#each mobilePrefTabs as tab (tab.id)}
          {@const active = !prefSearch.trim() && activePrefTab === tab.id}
          <button
            type="button"
            onclick={() => {
              prefSearch = '';
              activePrefTab = tab.id;
            }}
            class="h-8 px-3 rounded-full text-xs inline-flex items-center gap-1.5 shrink-0 cursor-pointer transition-colors {active
              ? 'font-semibold'
              : 'font-medium hover:bg-[var(--md-sys-color-state-hover)]'}"
            style={active
              ? 'background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);'
              : 'background-color: var(--md-sys-color-surface-container); color: var(--md-sys-color-on-surface-variant);'}
          >
            <Icon name={tab.icon} weight={active ? 'fill' : 'regular'} size={13} class="shrink-0" />
            <span>{tr(lang, tab.titleKey, tab.title)}</span>
          </button>
        {/each}
      </div>
    </div>

    <div class="flex-1 min-h-0 overflow-y-auto overscroll-contain p-4 sm:p-6 pb-24 sm:pb-20">
      {#key prefSearch.trim() ? '__search__' : activePrefTab}
        <div in:fly={{ y: 10, duration: 200 }} class="space-y-6">
      {#if !prefSearch.trim() && isAppTab(activePrefTab)}
        <div class="w-full space-y-5">
          <div
            class="relative overflow-hidden rounded-2xl p-6 flex items-center justify-between"
            style="background: linear-gradient(120deg, color-mix(in srgb, var(--md-sys-color-primary) 28%, var(--md-sys-color-surface-container-high)) 0%, color-mix(in srgb, var(--md-sys-color-primary) 12%, var(--md-sys-color-surface-container-low)) 55%, var(--md-sys-color-surface-container-low) 100%);"
          >
            <div class="relative z-10 flex items-center gap-5">
              <img src="/velora-logo-dark.png" alt="Velora" class="w-16 h-16 rounded-full object-cover shrink-0" />
              <div class="space-y-1">
                <h2 class="text-2xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
                  Velora
                </h2>
                <p class="text-sm font-medium" style="color: var(--md-sys-color-on-surface-variant);">
                  {tr(lang, 'nav.versionForWindows')}
                </p>
              </div>
            </div>
          </div>

          <div
            class="relative overflow-hidden rounded-2xl p-6 min-h-[172px] flex items-end justify-between gap-4"
            style="background: linear-gradient(300deg, color-mix(in srgb, var(--md-sys-color-primary) 28%, var(--md-sys-color-surface-container-high)) 0%, color-mix(in srgb, var(--md-sys-color-primary) 12%, var(--md-sys-color-surface-container-low)) 55%, var(--md-sys-color-surface-container-low) 100%);"
          >
            <div class="relative z-10 space-y-1">
              <div class="text-3xl sm:text-4xl font-extrabold tracking-tight leading-none" style="color: var(--md-sys-color-on-surface);">
                Yoapipp
              </div>
              <div class="text-sm font-medium" style="color: var(--md-sys-color-on-surface-variant);">
                {tr(lang, 'nav.developer')}
              </div>
            </div>
            <img
              src="/dev-profile.png"
              alt="Yoapipp"
              class="relative z-10 w-28 h-28 rounded-full object-cover shrink-0 shadow-lg"
            />
          </div>

          <div class="space-y-0.5">
            <div
              class="px-3.5 text-[11px] font-semibold uppercase tracking-wider"
              style="color: var(--md-sys-color-on-surface-variant); opacity: 0.78;"
            >
              {tr(lang, 'nav.socialMedia')}
            </div>
            <button
              type="button"
              onclick={() => api.openExternalUrl('https://www.instagram.com/yoapipp/')}
              class="w-full py-3.5 px-3.5 flex items-center justify-between text-left cursor-pointer rounded-xl transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
            >
              <span class="flex items-center gap-3.5 text-sm font-medium" style="color: var(--md-sys-color-on-surface);">
                <Icon name="instagram-logo" weight="fill" size={18} style="color: var(--md-sys-color-on-surface-variant);" />
                Instagram · @yoapipp
              </span>
              <Icon name="arrow-up-right" size={15} style="color: var(--md-sys-color-on-surface-variant);" />
            </button>
            <hr class="border-0 h-px w-full" style="background-color: var(--md-sys-color-outline-variant);" />
            <button
              type="button"
              onclick={() => api.openExternalUrl('https://github.com/Rafifdev')}
              class="w-full py-3.5 px-3.5 flex items-center justify-between text-left cursor-pointer rounded-xl transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
            >
              <span class="flex items-center gap-3.5 text-sm font-medium" style="color: var(--md-sys-color-on-surface);">
                <Icon name="github-logo" weight="fill" size={18} style="color: var(--md-sys-color-on-surface-variant);" />
                GitHub · RafifDev
              </span>
              <Icon name="arrow-up-right" size={15} style="color: var(--md-sys-color-on-surface-variant);" />
            </button>
          </div>

          <button
            type="button"
            onclick={() => (showSupportModal = true)}
            class="w-full rounded-2xl px-4 py-3.5 flex items-center gap-3.5 text-left cursor-pointer transition-transform active:scale-[0.99] hover:brightness-105"
            style="background-color: var(--md-sys-color-primary); color: var(--md-sys-color-on-primary);"
          >
            <Icon name="heart" weight="fill" size={20} class="shrink-0" />
            <div class="min-w-0 flex-1">
              <div class="text-sm font-bold">
                {tr(lang, 'nav.buyCoffee')}
              </div>
              <div class="text-xs opacity-85">
                {tr(lang, 'nav.buyCoffeeDesc')}
              </div>
            </div>
          </button>

          <button
            type="button"
            onclick={() => (showFeedbackModal = true)}
            class="w-full rounded-2xl px-4 py-3.5 flex items-center gap-3.5 text-left cursor-pointer transition-transform active:scale-[0.99] hover:brightness-105"
            style="background-color: var(--md-sys-color-secondary-container); color: var(--md-sys-color-on-secondary-container);"
          >
            <Icon name="chat-circle-text" weight="fill" size={20} class="shrink-0" />
            <div class="min-w-0 flex-1">
              <div class="text-sm font-bold">
                {tr(lang, 'nav.feedbackTitle')}
              </div>
              <div class="text-xs opacity-85">
                {tr(lang, 'nav.feedbackDesc')}
              </div>
            </div>
            <Icon name="caret-right" size={16} class="shrink-0 opacity-75" />
          </button>

          <div class="space-y-0.5">
            <button
              type="button"
              onclick={() => (showReleaseNotesModal = true)}
              class="w-full py-3.5 px-3.5 flex items-center justify-between text-left cursor-pointer rounded-xl transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
            >
              <span class="flex items-center gap-3.5 text-sm font-medium" style="color: var(--md-sys-color-on-surface);">
                <Icon name="seal-warning" weight="fill" size={18} style="color: var(--md-sys-color-on-surface-variant);" />
                {tr(lang, 'nav.releaseNotes')}
              </span>
            </button>
            <hr class="border-0 h-px w-full" style="background-color: var(--md-sys-color-outline-variant);" />
            <button
              type="button"
              disabled={checkingUpdates}
              onclick={() => void handleCheckForUpdates()}
              class="w-full py-3.5 px-3.5 flex items-center justify-between text-left cursor-pointer rounded-xl transition-colors hover:bg-[var(--md-sys-color-state-hover)] disabled:opacity-70 disabled:cursor-wait"
            >
              <span class="flex items-center gap-3.5 text-sm font-medium" style="color: var(--md-sys-color-on-surface);">
                <Icon name="download-simple" size={18} style="color: var(--md-sys-color-on-surface-variant);" />
                {tr(lang, 'nav.checkForUpdates')}
              </span>
              {#if updateStatusText}
                <span class="text-xs font-medium" style="color: var(--md-sys-color-primary);">
                  {updateStatusText}
                </span>
              {/if}
            </button>
            <hr class="border-0 h-px w-full" style="background-color: var(--md-sys-color-outline-variant);" />
            <div class="py-3.5 px-3.5 flex items-center justify-between gap-4">
              <div class="min-w-0 flex-1 space-y-0.5">
                <div class="text-sm font-medium" style="color: var(--md-sys-color-on-surface);">
                  {tr(lang, 'nav.dataFolder')}
                </div>
                {#if dataDirPath}
                  <div class="text-xs truncate" style="color: var(--md-sys-color-on-surface-variant);">
                    {dataDirPath}
                  </div>
                {/if}
              </div>
              <button
                type="button"
                onclick={() => void api.openDataFolder()}
                class="m3-btn-tonal px-4 min-w-[104px] h-9 text-xs font-semibold inline-flex items-center justify-center cursor-pointer shrink-0"
              >
                {tr(lang, 'nav.open')}
              </button>
            </div>
          </div>
        </div>
      {:else}
        <div class="w-full flex items-center justify-between">
          <h2 class="text-2xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
            {prefSearch.trim() ? `${tr(lang, 'nav.searchResults')} "${prefSearch}"` : activeCategoryTitle}
          </h2>
        </div>

        <div class="w-full space-y-4">
          {#if groupedVisibleSettings.length === 0}
            <EmptyState
              icon="magnifying-glass"
              iconSize={40}
              title={tr(lang, 'nav.noMatchingSettings')}
              description={tr(lang, 'nav.noMatchingSettingsDesc')}
              titleClass="text-base font-bold"
              descClass="mt-1 text-xs max-w-sm mx-auto"
              class="min-h-[45vh] flex flex-col items-center justify-center text-center px-4"
            />
          {:else}
            {#each groupedVisibleSettings as grp (grp.name)}
              <div class="space-y-0.5">
                <div
                  class="px-1 text-[11px] font-semibold uppercase tracking-wider"
                  style="color: var(--md-sys-color-on-surface-variant); opacity: 0.78;"
                >
                  {grp.name}
                </div>
                <div class="divide-y divide-[var(--md-sys-color-outline-variant)]">
                  {#each grp.items as item (item.id)}
                    {@render settingControl(item)}
                  {/each}
                </div>
              </div>
            {/each}
          {/if}
        </div>
      {/if}
        </div>
      {/key}
    </div>
    <ModalDialog
      open={showReleaseNotesModal}
      onClose={() => (showReleaseNotesModal = false)}
      ariaLabelledby="release-notes-title"
      maxWidthClass="max-w-lg"
      class="rounded-2xl p-6"
    >
      <div class="flex items-start justify-between gap-4 mb-5">
        <div class="space-y-1">
          <div class="flex items-center gap-2">
            <h2 id="release-notes-title" class="text-lg font-bold tracking-tight">
              {tr(lang, 'nav.releaseNotes')}
            </h2>
            <span
              class="px-2.5 py-0.5 rounded-full text-[11px] font-bold"
              style="background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);"
            >
              v1.0.0
            </span>
          </div>
          <p class="text-xs" style="color: var(--md-sys-color-on-surface-variant);">
            {tr(lang, 'nav.releaseNotesSubtitle')}
          </p>
        </div>
        <button
          type="button"
          onclick={() => (showReleaseNotesModal = false)}
          aria-label={tr(lang, 'nav.close')}
          class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer shrink-0"
        >
          <Icon name="x" size={16} />
        </button>
      </div>

      <div class="space-y-4 max-h-[55vh] overflow-y-auto pr-1 text-xs leading-relaxed">
        <div class="space-y-1.5">
          <div class="font-bold uppercase tracking-wider text-[11px]" style="color: var(--md-sys-color-primary);">
            {tr(lang, 'nav.releaseNotesSec1')}
          </div>
          <ul class="list-disc pl-5 space-y-1" style="color: var(--md-sys-color-on-surface-variant);">
            <li>
              {tr(lang, 'nav.releaseNotesItem1')}
            </li>
            <li>
              {tr(lang, 'nav.releaseNotesItem2')}
            </li>
            <li>
              {tr(lang, 'nav.releaseNotesItem3')}
            </li>
          </ul>
        </div>

        <div class="space-y-1.5">
          <div class="font-bold uppercase tracking-wider text-[11px]" style="color: var(--md-sys-color-primary);">
            {tr(lang, 'nav.releaseNotesSec2')}
          </div>
          <ul class="list-disc pl-5 space-y-1" style="color: var(--md-sys-color-on-surface-variant);">
            <li>
              {tr(lang, 'nav.releaseNotesItem4')}
            </li>
            <li>
              {tr(lang, 'nav.releaseNotesItem5')}
            </li>
          </ul>
        </div>

        <div class="space-y-1.5">
          <div class="font-bold uppercase tracking-wider text-[11px]" style="color: var(--md-sys-color-primary);">
            {tr(lang, 'nav.releaseNotesSec3')}
          </div>
          <ul class="list-disc pl-5 space-y-1" style="color: var(--md-sys-color-on-surface-variant);">
            <li>
              {tr(lang, 'nav.releaseNotesItem6')}
            </li>
          </ul>
        </div>
      </div>
    </ModalDialog>
    <ModalDialog
      open={showUpdateModal}
      onClose={() => (showUpdateModal = false)}
      ariaLabelledby="update-modal-title"
      maxWidthClass="max-w-md"
      class="rounded-2xl p-6"
    >
      <div class="space-y-1.5 mb-4">
        <div class="flex items-center gap-2">
          <h2 id="update-modal-title" class="text-lg font-bold tracking-tight">
            {tr(lang, 'nav.updateAvailableTitle')}
          </h2>
          {#if updateResult}
            <span
              class="px-2.5 py-0.5 rounded-full text-[11px] font-bold"
              style="background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);"
            >
              v{updateResult.latestVersion}
            </span>
          {/if}
        </div>
        {#if updateResult}
          <p class="text-xs" style="color: var(--md-sys-color-on-surface-variant);">
            {tr(lang, 'nav.newVersionAvailable')}: v{updateResult.currentVersion} → v{updateResult.latestVersion}
          </p>
        {/if}
      </div>
      {#if updateResult?.releaseNotes}
        <div
          class="p-3.5 rounded-xl text-xs whitespace-pre-line max-h-48 overflow-y-auto mb-5"
          style="background-color: var(--md-sys-color-surface-container); color: var(--md-sys-color-on-surface-variant);"
        >
          {updateResult.releaseNotes}
        </div>
      {/if}
      <div class="flex items-center justify-end gap-2">
        <button
          type="button"
          onclick={() => (showUpdateModal = false)}
          class="m3-btn-tonal px-4 py-2 text-xs font-semibold rounded-full cursor-pointer"
        >
          {tr(lang, 'nav.later')}
        </button>
        <button
          type="button"
          onclick={() => {
            if (updateResult) {
              api.openExternalUrl(updateResult.downloadUrl || updateResult.releaseUrl);
            }
            showUpdateModal = false;
          }}
          class="m3-btn-filled px-5 py-2 text-xs font-semibold rounded-full cursor-pointer"
        >
          {tr(lang, 'nav.downloadUpdate')}
        </button>
      </div>
    </ModalDialog>
    <ModalDialog
      open={showShortcutsModal}
      onClose={() => (showShortcutsModal = false)}
      ariaLabelledby="shortcuts-modal-title"
      maxWidthClass="max-w-md"
      class="p-6"
    >
      <div class="mb-4">
        <h2 id="shortcuts-modal-title" class="text-lg font-bold tracking-tight">
          {tr(lang, 'nav.keyboardShortcuts')}
        </h2>
      </div>
      <div class="divide-y text-xs" style="border-color: var(--md-sys-color-outline-variant);">
        {#each [
          { keys: 'Space', label: tr(lang, 'nav.scPlayPause') },
          { keys: '← / →', label: tr(lang, 'nav.scSeek') },
          { keys: '↑ / ↓', label: tr(lang, 'nav.scVolume') },
          { keys: 'Ctrl + + / -', label: tr(lang, 'nav.scZoom') },
          { keys: 'Ctrl + 0', label: tr(lang, 'nav.scResetZoom') },
          { keys: 'Tab', label: tr(lang, 'nav.scFocusSearch') }
        ] as sc (sc.keys)}
          <div class="py-2.5 flex items-center justify-between gap-4">
            <span style="color: var(--md-sys-color-on-surface-variant);">{sc.label}</span>
            <kbd
              class="px-2.5 py-1 rounded-md text-[11px] font-semibold font-mono shrink-0"
              style="background-color: var(--md-sys-color-surface-container-highest); color: var(--md-sys-color-on-surface);"
            >
              {sc.keys}
            </kbd>
          </div>
        {/each}
      </div>
      <div class="flex items-center justify-end gap-2 mt-6">
        <button
          type="button"
          onclick={() => (showShortcutsModal = false)}
          class="h-10 px-5 rounded-full bg-[var(--md-sys-color-surface-container)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-xs font-semibold cursor-pointer transition-colors"
        >
          {tr(lang, 'nav.close')}
        </button>
      </div>
    </ModalDialog>

    <ModalDialog
      open={confirmResetIsland}
      onClose={() => (confirmResetIsland = false)}
      ariaLabelledby="reset-island-title"
      ariaDescribedby="reset-island-desc"
      maxWidthClass="max-w-md"
      class="p-6"
    >
      <div class="mb-3">
        <h2 id="reset-island-title" class="text-lg font-bold tracking-tight">
          {tr(lang, 'nav.resetIslandTitle')}
        </h2>
      </div>
      <p id="reset-island-desc" class="text-xs leading-relaxed" style="color: var(--md-sys-color-on-surface-variant);">
        {tr(lang, 'nav.resetIslandDesc')}
      </p>
      <div class="flex items-center justify-end gap-2 mt-6">
        <button
          type="button"
          onclick={() => (confirmResetIsland = false)}
          class="h-10 px-5 rounded-full bg-[var(--md-sys-color-surface-container)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-xs font-semibold cursor-pointer transition-colors"
        >
          {tr(lang, 'nav.cancel')}
        </button>
        <button
          type="button"
          onclick={() => {
            confirmResetIsland = false;
            executeResetIsland();
          }}
          class="h-10 px-7 rounded-full text-xs font-bold text-rose-300 bg-rose-500/15 hover:bg-rose-500/25 cursor-pointer transition-all duration-150 hover:scale-[1.03] active:scale-95"
        >
          {tr(lang, 'nav.reset')}
        </button>
      </div>
    </ModalDialog>

    <ModalDialog
      open={confirmClearCache}
      onClose={() => (confirmClearCache = false)}
      ariaLabelledby="clear-cache-title"
      ariaDescribedby="clear-cache-desc"
      maxWidthClass="max-w-md"
      class="p-6"
    >
      <div class="mb-3">
        <h2 id="clear-cache-title" class="text-lg font-bold tracking-tight">
          {tr(lang, 'nav.clearCacheTitle')}
        </h2>
      </div>
      <p id="clear-cache-desc" class="text-xs leading-relaxed" style="color: var(--md-sys-color-on-surface-variant);">
        {tr(lang, 'nav.clearCacheDesc')}
      </p>
      <div class="flex items-center justify-end gap-2 mt-6">
        <button
          type="button"
          onclick={() => (confirmClearCache = false)}
          class="h-10 px-5 rounded-full bg-[var(--md-sys-color-surface-container)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-xs font-semibold cursor-pointer transition-colors"
        >
          {tr(lang, 'nav.cancel')}
        </button>
        <button
          type="button"
          onclick={() => {
            confirmClearCache = false;
            void executeClearCache();
          }}
          class="h-10 px-7 rounded-full text-xs font-bold text-rose-300 bg-rose-500/15 hover:bg-rose-500/25 cursor-pointer transition-all duration-150 hover:scale-[1.03] active:scale-95"
        >
          {tr(lang, 'nav.clear')}
        </button>
      </div>
    </ModalDialog>

    <ModalDialog
      open={showSupportModal}
      onClose={() => (showSupportModal = false)}
      ariaLabelledby="support-modal-title"
      maxWidthClass="max-w-sm"
      class="rounded-2xl p-6"
    >
      <div class="flex items-start justify-between gap-4 mb-4">
        <div>
          <h2 id="support-modal-title" class="text-base font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
            {tr(lang, 'nav.buyCoffee')}
          </h2>
          <p class="text-xs" style="color: var(--md-sys-color-on-surface-variant);">
            {tr(lang, 'nav.buyCoffeeDesc')}
          </p>
        </div>
        <button
          type="button"
          onclick={() => (showSupportModal = false)}
          aria-label={tr(lang, 'nav.close')}
          class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer shrink-0"
        >
          <Icon name="x" size={15} />
        </button>
      </div>

      <div class="flex flex-col items-center text-center gap-3.5 py-2">
        <div class="bg-white p-3 rounded-2xl shadow-md">
          {#if supportQrDataUrl}
            <img
              src={supportQrDataUrl}
              alt="QR Code SociaBuzz Yoapipp"
              class="w-48 h-48 block rounded-lg"
            />
          {:else}
            <div class="w-48 h-48 rounded-lg bg-neutral-100 flex items-center justify-center text-neutral-500">
              <Icon name="qr-code" size={44} />
            </div>
          {/if}
        </div>
        <p class="text-xs leading-relaxed max-w-[260px]" style="color: var(--md-sys-color-on-surface-variant);">
          {tr(lang, 'nav.scanSupportQrHint')}
        </p>
        <button
          type="button"
          onclick={() => api.openExternalUrl(SUPPORT_TRIBE_URL)}
          class="w-full h-10 px-5 rounded-full text-xs font-bold inline-flex items-center justify-center gap-2 cursor-pointer transition-transform hover:scale-[1.02] active:scale-95"
          style="background-color: var(--md-sys-color-primary); color: var(--md-sys-color-on-primary);"
        >
          <span>{tr(lang, 'nav.openSociaBuzz')}</span>
          <Icon name="arrow-up-right" size={14} />
        </button>
      </div>
    </ModalDialog>

    <ModalDialog
      open={showFeedbackModal}
      onClose={() => {
        showFeedbackModal = false;
        if (feedbackSubmitted) resetFeedbackForm();
      }}
      ariaLabelledby="feedback-modal-title"
      maxWidthClass="max-w-lg"
      class="rounded-2xl p-6"
    >
      <div class="flex items-center justify-between gap-4 mb-4">
        <h2 id="feedback-modal-title" class="text-base font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
          {tr(lang, 'nav.feedbackTitle')}
        </h2>
        <button
          type="button"
          onclick={() => {
            showFeedbackModal = false;
            if (feedbackSubmitted) resetFeedbackForm();
          }}
          aria-label={tr(lang, 'nav.close')}
          class="m3-icon-btn w-8 h-8 flex items-center justify-center cursor-pointer shrink-0"
        >
          <Icon name="x" size={15} />
        </button>
      </div>

      {#if feedbackSubmitted}
        <div in:scale={{ start: 0.95, duration: 240 }} out:fade={{ duration: 140 }}>
          <EmptyState
            icon="check-circle"
            iconWeight="fill"
            iconSize={38}
            title={tr(lang, 'nav.feedbackSentTitle')}
            description={tr(lang, 'nav.feedbackSentDesc')}
            titleClass="text-lg font-bold tracking-tight"
            descClass="mt-1.5 text-xs max-w-sm mx-auto leading-relaxed"
            class="flex flex-col items-center justify-center text-center py-8 px-4"
          />
        </div>
      {:else}
        <form
          in:fade={{ duration: 180 }}
          onsubmit={(e) => {
            e.preventDefault();
            void submitFeedbackForm();
          }}
          class="space-y-4"
        >
          <label class="block space-y-1.5">
            <span class="block text-xs font-semibold" style="color: var(--md-sys-color-on-surface-variant);">
              {tr(lang, 'nav.feedbackSubject')}
            </span>
            <input
              type="text"
              required
              maxlength={140}
              bind:value={feedbackTitle}
              placeholder={tr(lang, 'nav.feedbackSubjectPlaceholder')}
              class="w-full h-10 px-3.5 rounded-xl text-sm outline-none border transition-colors focus:border-[var(--md-sys-color-primary)]"
              style="background-color: var(--md-sys-color-surface-container-highest); color: var(--md-sys-color-on-surface); border-color: color-mix(in srgb, var(--md-sys-color-on-surface) 18%, transparent);"
            />
          </label>

          <label class="block space-y-1.5">
            <span class="block text-xs font-semibold" style="color: var(--md-sys-color-on-surface-variant);">
              {tr(lang, 'nav.feedbackDetail')}
            </span>
            <textarea
              required
              rows={4}
              bind:value={feedbackDetail}
              placeholder={tr(lang, 'nav.feedbackDetailPlaceholder')}
              class="w-full p-3.5 rounded-xl text-sm outline-none border resize-y min-h-[110px] transition-colors focus:border-[var(--md-sys-color-primary)]"
              style="background-color: var(--md-sys-color-surface-container-highest); color: var(--md-sys-color-on-surface); border-color: color-mix(in srgb, var(--md-sys-color-on-surface) 18%, transparent);"
            ></textarea>
          </label>

          <div class="space-y-2">
            <div class="flex items-center justify-between gap-2">
              <span class="text-xs font-semibold" style="color: var(--md-sys-color-on-surface-variant);">
                {tr(lang, 'nav.feedbackScreenshot')}
              </span>
              <span class="text-[11px]" style="color: var(--md-sys-color-on-surface-variant); opacity: 0.78;">
                {tr(lang, 'nav.feedbackScreenshotHint')}
              </span>
            </div>

            <input
              bind:this={feedbackFileInputEl}
              type="file"
              accept="image/*"
              onchange={handleFeedbackFileChange}
              class="hidden"
            />

            {#if feedbackScreenshotUrl && feedbackScreenshotFile}
              <div
                in:scale={{ start: 0.92, duration: 180 }}
                class="relative group w-24 h-24 rounded-xl overflow-hidden border"
                style="border-color: var(--md-sys-color-outline-variant);"
              >
                <img
                  src={feedbackScreenshotUrl}
                  alt="Screenshot preview"
                  class="w-full h-full object-cover"
                />
                <button
                  type="button"
                  onclick={() => {
                    setFeedbackScreenshot(null);
                    if (feedbackFileInputEl) feedbackFileInputEl.value = '';
                  }}
                  aria-label={tr(lang, 'nav.feedbackRemoveImage')}
                  class="absolute top-1.5 right-1.5 w-6 h-6 rounded-full bg-black/70 text-white flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity cursor-pointer hover:bg-black/90"
                >
                  <Icon name="x" size={12} weight="bold" />
                </button>
              </div>
            {:else}
              <button
                type="button"
                onclick={() => feedbackFileInputEl?.click()}
                class="w-full py-3.5 px-4 rounded-xl border-2 border-dashed flex items-center justify-center gap-2 text-xs font-semibold cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
                style="background-color: var(--md-sys-color-surface-container-highest); border-color: color-mix(in srgb, var(--md-sys-color-on-surface) 28%, transparent); color: var(--md-sys-color-on-surface-variant);"
              >
                <Icon name="image" size={18} />
                <span>{tr(lang, 'nav.feedbackPickImage')}</span>
              </button>
            {/if}
          </div>

          {#if feedbackError}
            <p class="text-xs font-medium text-rose-400">
              {feedbackError}
            </p>
          {/if}

          <div class="mt-6">
            <button
              type="submit"
              disabled={!feedbackTitle.trim() || !feedbackDetail.trim() || feedbackSubmitting}
              class="w-full h-10 px-6 rounded-full text-xs font-bold inline-flex items-center justify-center gap-2 cursor-pointer transition-all duration-150 hover:scale-[1.01] active:scale-95 disabled:opacity-55 disabled:cursor-not-allowed disabled:hover:scale-100"
              style="background-color: var(--md-sys-color-primary); color: var(--md-sys-color-on-primary);"
            >
              {#if feedbackSubmitting}
                <Icon name="circle-notch" weight="bold" size={15} class="animate-spin" />
                <span>{tr(lang, 'nav.feedbackSending')}</span>
              {:else}
                <Icon name="paper-plane-tilt" weight="fill" size={14} />
                <span>{tr(lang, 'nav.feedbackSend')}</span>
              {/if}
            </button>
          </div>
        </form>
      {/if}
    </ModalDialog>
    {#if cacheToast}
      <div
        in:fly={{ y: 12, duration: 180 }}
        out:fly={{ y: 12, duration: 150 }}
        class="absolute bottom-5 right-6 px-4 py-2 rounded-full text-xs font-semibold shadow-lg pointer-events-none flex items-center gap-2 z-40"
        style="background-color: var(--md-sys-color-inverse-surface); color: var(--md-sys-color-inverse-on-surface);"
      >
        <Icon name="check-circle" weight="fill" size={15} />
        <span>{cacheToast}</span>
      </div>
    {/if}
  </div>
{/if}

<ModalDialog
  open={confirmClearHistory}
  onClose={() => (confirmClearHistory = false)}
  ariaLabelledby="clear-history-title"
  ariaDescribedby="clear-history-desc"
  maxWidthClass="max-w-md"
  class="p-6"
>
  <div class="mb-3">
    <h2 id="clear-history-title" class="text-lg font-bold tracking-tight">
      {tr(lang, 'nav.clearHistoryTitle')}
    </h2>
  </div>
  <p id="clear-history-desc" class="text-xs leading-relaxed" style="color: var(--md-sys-color-on-surface-variant);">
    {tr(lang, 'nav.clearHistoryDesc')}
  </p>
  <div class="flex items-center justify-end gap-2 mt-6">
    <button
      type="button"
      onclick={() => (confirmClearHistory = false)}
      class="h-10 px-5 rounded-full bg-[var(--md-sys-color-surface-container)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-xs font-semibold cursor-pointer transition-colors"
    >
      {tr(lang, 'nav.cancel')}
    </button>
    <button
      type="button"
      onclick={() => {
        confirmClearHistory = false;
        onClearHistory();
      }}
      class="h-10 px-7 rounded-full text-xs font-bold text-rose-300 cursor-pointer transition-all duration-150 hover:bg-rose-500/25 hover:scale-[1.03] active:scale-95"
      style="background-color: rgba(244, 63, 94, 0.16);"
    >
      {tr(lang, 'nav.clear')}
    </button>
  </div>
</ModalDialog>
