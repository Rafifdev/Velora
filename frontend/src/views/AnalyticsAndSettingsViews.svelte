<script lang="ts">
  import { formatDuration, resolveCoverUrl } from '../lib/api';
  import { svelteMotionEnter } from '../lib/motion';
  import {
    BUILTIN_PLUGINS,
    KEY_SHORTCUTS_LIST,
    NUCLEAR_APP_TABS,
    NUCLEAR_SETTINGS,
    NUCLEAR_THEMES,
    getPrefBool,
    getPrefValue,
    type SettingItemDef
  } from '../lib/nuclearPrefs';
  import type { FireStreakStatus, ListeningStats, PlayHistoryItem, Track, ViewRoute } from '../lib/types';
  import Icon from '../components/Icon.svelte';

  interface Props {
    mode: 'history' | 'listening-stats' | 'fire-streak' | 'preferences';
    history: PlayHistoryItem[];
    listeningStats: ListeningStats;
    fireStreak: FireStreakStatus;
    preferences: Record<string, string>;
    tracks: Track[];
    mediaBaseUrl: string;
    accentColor: string;
    activePrefTab?: string;
    prefSearch?: string;
    onPlayTrack: (track: Track, list: Track[]) => void;
    onClearHistory: () => void;
    onSavePreferences: (prefs: Record<string, string>) => void;
    onClearCache: () => void;
    onNavigate?: (route: ViewRoute) => void;
  }

  let {
    mode,
    history,
    listeningStats,
    fireStreak,
    preferences,
    tracks,
    mediaBaseUrl,
    accentColor: _accentColor,
    activePrefTab = $bindable('appearance'),
    prefSearch = $bindable(''),
    onPlayTrack,
    onClearHistory,
    onSavePreferences,
    onClearCache,
    onNavigate
  }: Props = $props();

  let savedToast = $state('');
  let recordingShortcutId = $state<string | null>(null);
  let shortcutConflict = $state('');
  let copiedKey = $state('');
  let historyTab = $state<'recent' | 'stats'>('recent');
  let historyFilter = $state('');

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

  const visibleSettingsForCategory = $derived.by(() => {
    const q = prefSearch.trim().toLowerCase();
    return NUCLEAR_SETTINGS.filter((s) => {
      if (q) {
        return (
          s.title.toLowerCase().includes(q) ||
          s.description.toLowerCase().includes(q) ||
          s.category.toLowerCase().includes(q)
        );
      }
      return s.category === activePrefTab;
    });
  });

  const maxHourMinutes = $derived(
    Math.max(1, ...listeningStats.timeOfDay.map((b) => b.minutes))
  );
  const maxDayMinutes = $derived(
    Math.max(1, ...listeningStats.dayOfWeek.map((d) => d.minutes))
  );

  function updateSetting(key: string, value: string) {
    onSavePreferences({ [key]: value });
    savedToast = 'Preference saved';
    setTimeout(() => (savedToast = ''), 1800);
  }

  function applyThemePreset(theme: (typeof NUCLEAR_THEMES)[number]) {
    onSavePreferences({
      'theme.seed': theme.seed,
      'theme.dark': theme.dark ? 'true' : 'false',
      'theme.active.id': theme.id,
      accent_color: theme.accent
    });
    savedToast = `Theme "${theme.name}" applied`;
    setTimeout(() => (savedToast = ''), 2000);
  }

  function handleShortcutKeyDown(e: KeyboardEvent, cmdId: string) {
    if (recordingShortcutId !== cmdId) return;
    e.preventDefault();
    e.stopPropagation();
    const parts: string[] = [];
    if (e.ctrlKey || e.metaKey) parts.push('Ctrl');
    if (e.altKey) parts.push('Alt');
    if (e.shiftKey) parts.push('Shift');
    const keyName = e.key === ' ' ? 'Space' : e.key.length === 1 ? e.key.toUpperCase() : e.key;
    if (!['Control', 'Shift', 'Alt', 'Meta'].includes(keyName)) {
      parts.push(keyName);
      const combo = parts.join('+');
      const dup = KEY_SHORTCUTS_LIST.find(
        (k) => k.id !== cmdId && getPrefValue(preferences, `shortcut.${k.id}`) === combo
      );
      if (dup) {
        shortcutConflict = `This shortcut is already used by "${dup.label}"`;
      } else {
        shortcutConflict = '';
        updateSetting(`shortcut.${cmdId}`, combo);
      }
      recordingShortcutId = null;
    }
  }

  function resetAllShortcuts() {
    const patch: Record<string, string> = {};
    for (const sc of KEY_SHORTCUTS_LIST) {
      patch[`shortcut.${sc.id}`] = sc.defaultKey;
    }
    onSavePreferences(patch);
    shortcutConflict = '';
    savedToast = 'All key shortcuts reset to defaults';
    setTimeout(() => (savedToast = ''), 2000);
  }

  function copyInfoValue(id: string, val: string) {
    navigator.clipboard?.writeText(val);
    copiedKey = id;
    setTimeout(() => (copiedKey = ''), 1500);
  }

  function historyItemToTrack(item: PlayHistoryItem): Track {
    const found = tracks.find((t) => t.id === item.trackId);
    if (found) return found;
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
      createdAt: item.playedAt || ''
    };
  }

  function playHistoryTrack(item: PlayHistoryItem) {
    const target = historyItemToTrack(item);
    const list = filteredHistory.map(historyItemToTrack);
    onPlayTrack(target, list.length > 0 ? list : [target]);
  }

  function playAllHistory() {
    if (filteredHistory.length === 0) return;
    const list = filteredHistory.map(historyItemToTrack);
    onPlayTrack(list[0], list);
  }

  function isAppTab(tab: string): boolean {
    return NUCLEAR_APP_TABS.some((t) => t.id === tab);
  }
</script>

{#snippet m3Switch(checked: boolean, label: string, onToggle: () => void)}
  <button
    type="button"
    role="switch"
    aria-checked={checked}
    aria-label={label}
    onclick={onToggle}
    class="w-[52px] h-8 rounded-full relative transition-colors duration-200 cursor-pointer shrink-0 flex items-center"
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

{#snippet settingControl(item: SettingItemDef)}
  {@const val = getPrefValue(preferences, item.id)}
  {@const checked = val === 'true'}
  <div class="m3-card p-4 rounded-lg flex flex-wrap items-center justify-between gap-4">
    <div class="min-w-0 flex-1 space-y-0.5">
      <div class="text-sm font-semibold" style="color: var(--md-sys-color-on-surface);">{item.title}</div>
      <div class="text-xs leading-relaxed" style="color: var(--md-sys-color-on-surface-variant);">
        {item.description}
      </div>
    </div>

    <div class="shrink-0 flex items-center gap-2">
      {#if item.widget === 'toggle'}
        {@render m3Switch(checked, item.title, () => updateSetting(item.id, checked ? 'false' : 'true'))}
      {:else if item.widget === 'select'}
        <select
          value={val}
          onchange={(e) => updateSetting(item.id, (e.target as HTMLSelectElement).value)}
          class="m3-input h-9 px-3.5 rounded-md text-xs font-medium outline-none cursor-pointer"
        >
          {#each item.options || [] as opt (opt.value)}
            <option value={opt.value}>{opt.label}</option>
          {/each}
        </select>
      {:else if item.widget === 'slider'}
        <div class="w-44 space-y-1.5">
          <div class="flex items-center justify-between text-[10px] font-medium" style="color: var(--md-sys-color-on-surface-variant);">
            <span>{item.startLabel || item.min}</span>
            <span class="tabular-nums font-semibold" style="color: var(--md-sys-color-on-surface);">{val}</span>
            <span>{item.endLabel || item.max}</span>
          </div>
          <input
            type="range"
            min={item.min}
            max={item.max}
            step={item.step}
            value={Number(val)}
            oninput={(e) => updateSetting(item.id, (e.target as HTMLInputElement).value)}
            class="velora-range w-full"
            style="background-color: var(--md-sys-color-surface-container-highest);"
          />
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
        <div class="flex items-center gap-1.5">
          <code
            class="px-3 py-1.5 rounded-md text-[11px] font-mono select-all"
            style="background-color: var(--md-sys-color-surface-container-highest); color: var(--md-sys-color-on-surface);"
          >
            {val}
          </code>
          <button
            type="button"
            onclick={() => copyInfoValue(item.id, val)}
            title="Copy URL"
            class="m3-icon-btn-tonal w-8 h-8 flex items-center justify-center cursor-pointer"
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
{/snippet}

{#if mode === 'history'}
  <div
    class="relative h-full w-full overflow-y-auto rounded-xl select-none flex flex-col"
    style="background-color: var(--md-sys-color-surface-container-low);"
    use:svelteMotionEnter={{ y: 8 }}
  >
    <!-- Hero Banner -->
    <section
      class="relative z-10 px-6 sm:px-7 pt-7 pb-7"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-secondary) 26%, var(--md-sys-color-secondary-container)) 0%, color-mix(in srgb, var(--md-sys-color-secondary-container) 74%, var(--md-sys-color-surface-container-low)) 100%);"
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
            <h1 class="text-3xl font-extrabold tracking-tight text-[var(--md-sys-color-on-surface)]">History</h1>
            <p class="text-xs text-[var(--md-sys-color-on-surface-variant)] tabular-nums">
              {history.length} recent streams • {listeningStats.totalMinutes} min listened
            </p>
          </div>
        </div>

        <!-- Segmented Tabs -->
        <div class="mb-[4px] inline-flex items-center gap-1 p-1 rounded-full bg-[var(--md-sys-color-surface-container-high)]">
          <button
            type="button"
            onclick={() => (historyTab = 'recent')}
            class="px-4 py-1.5 rounded-full text-xs font-semibold transition-colors cursor-pointer {historyTab === 'recent'
              ? 'bg-[var(--md-sys-color-on-surface)] text-[var(--md-sys-color-surface)]'
              : 'text-[var(--md-sys-color-on-surface-variant)] hover:text-[var(--md-sys-color-on-surface)]'}"
          >
            Recent Streams
          </button>
          <button
            type="button"
            onclick={() => (historyTab = 'stats')}
            class="px-4 py-1.5 rounded-full text-xs font-semibold transition-colors cursor-pointer {historyTab === 'stats'
              ? 'bg-[var(--md-sys-color-on-surface)] text-[var(--md-sys-color-surface)]'
              : 'text-[var(--md-sys-color-on-surface-variant)] hover:text-[var(--md-sys-color-on-surface)]'}"
          >
            Statistics & Insights
          </button>
        </div>
      </div>
    </section>

    <div
      class="relative z-10 px-6 sm:px-8 pt-5 pb-10 flex-1 flex flex-col gap-5"
      style="background: linear-gradient(180deg, color-mix(in srgb, var(--md-sys-color-secondary-container) 48%, var(--md-sys-color-surface-container-low)) 0%, color-mix(in srgb, var(--md-sys-color-secondary-container) 28%, var(--md-sys-color-surface-container-low)) 120px, color-mix(in srgb, var(--md-sys-color-secondary-container) 11%, var(--md-sys-color-surface-container-low)) 240px, var(--md-sys-color-surface-container-low) 340px);"
    >

    {#if historyTab === 'recent'}
      <!-- Action & Filter Toolbar -->
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex items-center gap-2.5">
          <button
            type="button"
            onclick={playAllHistory}
            disabled={filteredHistory.length === 0}
            class="h-9 px-4 rounded-full text-xs font-bold flex items-center gap-1.5 cursor-pointer transition-transform active:scale-95 disabled:opacity-40"
            style="background-color: var(--md-sys-color-tertiary); color: var(--md-sys-color-on-tertiary);"
          >
            <Icon name="play" weight="fill" size={14} /> Play All
          </button>
          {#if history.length > 0}
            <button
              type="button"
              onclick={onClearHistory}
              class="h-9 px-3.5 rounded-full text-xs font-medium text-rose-300 flex items-center gap-1.5 cursor-pointer transition-colors hover:bg-rose-500/20"
              style="background-color: rgba(244, 63, 94, 0.14);"
            >
              <Icon name="trash" size={14} /> Clear History
            </button>
          {/if}
        </div>

        <div class="relative w-full sm:w-64">
          <span class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-[var(--md-sys-color-on-surface-variant)] opacity-75">
            <Icon name="magnifying-glass" size={14} />
          </span>
          <input
            type="text"
            bind:value={historyFilter}
            placeholder="Filter history by title, artist..."
            class="w-full h-9 pl-9 pr-3 rounded-lg text-xs text-[var(--md-sys-color-on-surface)] placeholder:text-[var(--md-sys-color-on-surface-variant)] placeholder:opacity-65 bg-[var(--md-sys-color-surface-container-high)] focus:bg-[var(--md-sys-color-surface-container-highest)] outline-none transition-colors"
          />
        </div>
      </div>

      {#if history.length === 0 || filteredHistory.length === 0}
        <div class="flex-1 flex flex-col items-center justify-center text-center py-12 px-4">
          <div class="mb-5" style="color: var(--md-sys-color-on-surface);">
            <Icon name={history.length === 0 ? 'music-note' : 'magnifying-glass'} weight="bold" size={38} />
          </div>
          <h2 class="text-2xl font-extrabold tracking-tight" style="color: var(--md-sys-color-on-surface);">
            {history.length === 0 ? 'Belum ada riwayat putar' : 'Riwayat tidak ditemukan'}
          </h2>
          <p class="mt-2 text-[13px] max-w-sm mx-auto" style="color: var(--md-sys-color-on-surface-variant);">
            {history.length === 0
              ? 'Lagu yang kamu putar akan muncul di sini.'
              : `Tidak ada riwayat yang cocok dengan "${historyFilter.trim()}".`}
          </p>
          <button
            type="button"
            onclick={() => (history.length === 0 ? onNavigate?.({ name: 'home' }) : (historyFilter = ''))}
            class="mt-6 h-10 px-6 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
            style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
          >
            {history.length === 0 ? 'Mulai dengarkan' : 'Reset filter'}
          </button>
        </div>
      {:else}
        <div class="space-y-1">
          {#each filteredHistory as item, idx (item.id)}
            {@const cover = resolveCoverUrl(item.coverUrl, mediaBaseUrl)}
            <div class="group flex items-center justify-between gap-3 px-3 py-2 rounded-lg transition-colors hover:bg-[var(--md-sys-color-state-hover)]">
              <button
                type="button"
                onclick={() => playHistoryTrack(item)}
                class="flex items-center gap-3 min-w-0 flex-1 text-left cursor-pointer"
              >
                <span class="w-5 text-[11px] font-medium text-[var(--md-sys-color-on-surface-variant)] opacity-70 tabular-nums text-center shrink-0">
                  {idx + 1}
                </span>
                {#if cover}
                  <img src={cover} alt={item.title} class="block w-10 h-10 rounded-md object-cover object-center shrink-0" />
                {:else}
                  <div class="w-10 h-10 rounded-md flex items-center justify-center shrink-0 bg-[var(--md-sys-color-surface-container-high)] text-[var(--md-sys-color-on-surface-variant)]">
                    <Icon name="music-note" size={16} />
                  </div>
                {/if}
                <div class="min-w-0 flex-1">
                  <div class="text-[13px] font-semibold truncate text-[var(--md-sys-color-on-surface)] group-hover:text-[var(--md-sys-color-primary)] transition-colors">
                    {item.title}
                  </div>
                  <div class="text-[11px] truncate text-[var(--md-sys-color-on-surface-variant)] mt-0.5">
                    {item.artistName} • {item.albumTitle}
                  </div>
                </div>
              </button>

              <div class="flex items-center gap-3 shrink-0 text-xs tabular-nums text-[var(--md-sys-color-on-surface-variant)]">
                <span class="text-[11px]">{formatDuration(item.durationListenedSec)}</span>
                <span class="hidden md:inline text-[11px] text-[var(--md-sys-color-on-surface-variant)] opacity-70">{item.playedAt}</span>
                <button
                  type="button"
                  onclick={() => playHistoryTrack(item)}
                  aria-label="Putar ulang {item.title}"
                  class="w-8 h-8 rounded-full flex items-center justify-center cursor-pointer transition-transform hover:scale-105 active:scale-95"
                  style="background-color: var(--md-sys-color-tertiary); color: var(--md-sys-color-on-tertiary);"
                >
                  <Icon name="play" weight="fill" size={13} />
                </button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    {:else}
      <!-- Statistics & Insights Tab (Borderless Nuclear Cards) -->
      <div class="grid grid-cols-2 lg:grid-cols-4 gap-3">
        <div class="rounded-lg p-4 bg-[var(--md-sys-color-surface-container)]">
          <div class="text-xs flex items-center gap-1.5 text-[var(--md-sys-color-on-surface-variant)]">
            <Icon name="clock" size={14} /> Total Listening
          </div>
          <div class="text-2xl font-bold mt-1.5 tabular-nums text-[var(--md-sys-color-on-surface)]">{listeningStats.totalMinutes} min</div>
          <div class="text-[11px] mt-1 tabular-nums text-[var(--md-sys-color-on-surface-variant)] opacity-85">
            {listeningStats.totalPlays} plays • {listeningStats.uniqueTracks} unique tracks
          </div>
        </div>
        <div class="rounded-lg p-4 bg-[var(--md-sys-color-surface-container)]">
          <div class="text-xs flex items-center gap-1.5 text-[var(--md-sys-color-on-surface-variant)]">
            <Icon name="sparkle" size={14} /> Peak Listening Hour
          </div>
          <div class="text-2xl font-bold mt-1.5 tabular-nums text-[var(--md-sys-color-on-surface)]">{listeningStats.peakHourLabel}</div>
          <div class="text-[11px] mt-1 text-[var(--md-sys-color-on-surface-variant)] opacity-85">{listeningStats.peakPersona}</div>
        </div>
        <div class="rounded-lg p-4 bg-[var(--md-sys-color-surface-container)]">
          <div class="text-xs flex items-center gap-1.5 text-[var(--md-sys-color-on-surface-variant)]">
            <Icon name="calendar" size={14} /> Most Active Day
          </div>
          <div class="text-2xl font-bold mt-1.5 text-[var(--md-sys-color-on-surface)]">{listeningStats.mostActiveDay}</div>
          <div class="text-[11px] mt-1 text-[var(--md-sys-color-on-surface-variant)] opacity-85">Highest total minutes</div>
        </div>
        <div class="rounded-lg p-4 bg-[var(--md-sys-color-surface-container)]">
          <div class="text-xs flex items-center gap-1.5 text-[var(--md-sys-color-on-surface-variant)]">
            <Icon name="flame" weight="fill" size={14} class="text-orange-400" /> Fire Streak
          </div>
          <div class="text-2xl font-bold text-orange-400 mt-1.5 tabular-nums">{fireStreak.currentStreak} Days</div>
          <div class="text-[11px] mt-1 tabular-nums text-[var(--md-sys-color-on-surface-variant)] opacity-85">Best: {fireStreak.longestStreak} days</div>
        </div>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <section class="rounded-lg p-4 bg-[var(--md-sys-color-surface-container-low)] space-y-2.5">
          <h2 class="text-sm font-semibold text-[var(--md-sys-color-on-surface)]">Time of Day (00:00 – 23:00)</h2>
          <svg viewBox="0 0 720 180" class="w-full h-40">
            {#each listeningStats.timeOfDay as bucket, i (bucket.hour)}
              {@const barH = Math.max(4, (bucket.minutes / maxHourMinutes) * 120)}
              {@const x = i * 29 + 14}
              {@const y = 144 - barH}
              <g>
                <rect
                  x={x}
                  y={y}
                  width="20"
                  height={barH}
                  rx="6"
                  fill={bucket.minutes > 0 ? 'var(--md-sys-color-tertiary)' : 'var(--md-sys-color-surface-container-highest)'}
                />
                {#if i % 3 === 0}
                  <text x={x + 10} y="166" text-anchor="middle" fill="currentColor" opacity="0.55" font-size="10">
                    {bucket.label.slice(0, 2)}
                  </text>
                {/if}
              </g>
            {/each}
          </svg>
        </section>

        <section class="rounded-lg p-4 bg-[var(--md-sys-color-surface-container-low)] space-y-2.5">
          <h2 class="text-sm font-semibold text-[var(--md-sys-color-on-surface)]">Day of Week</h2>
          <svg viewBox="0 0 420 180" class="w-full h-40">
            {#each listeningStats.dayOfWeek as day, i (day.dayIndex)}
              {@const barH = Math.max(6, (day.minutes / maxDayMinutes) * 116)}
              {@const x = i * 56 + 22}
              {@const y = 140 - barH}
              <g>
                <rect
                  x={x}
                  y={y}
                  width="34"
                  height={barH}
                  rx="8"
                  fill={day.minutes > 0 ? 'var(--md-sys-color-tertiary)' : 'var(--md-sys-color-surface-container-highest)'}
                />
                <text x={x + 17} y={y - 6} text-anchor="middle" fill="currentColor" opacity="0.65" font-size="10">
                  {day.minutes}m
                </text>
                <text x={x + 17} y="164" text-anchor="middle" fill="currentColor" font-size="11" font-weight="600">
                  {day.label}
                </text>
              </g>
            {/each}
          </svg>
        </section>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <section class="rounded-lg p-4 bg-[var(--md-sys-color-surface-container-low)] space-y-2.5">
          <h2 class="text-sm font-semibold text-[var(--md-sys-color-on-surface)]">Top Played Tracks</h2>
          {#if !listeningStats.topTracks || listeningStats.topTracks.length === 0}
            <p class="text-xs text-[var(--md-sys-color-on-surface-variant)] opacity-75 py-4">Belum ada data lagu terbanyak diputar.</p>
          {:else}
            <div class="space-y-1">
              {#each listeningStats.topTracks.slice(0, 8) as top, i (top.name + i)}
                <div class="flex items-center justify-between gap-3 px-2.5 py-2 rounded-md bg-[var(--md-sys-color-surface-container)]">
                  <div class="flex items-center gap-2.5 min-w-0">
                    <span class="w-5 text-xs font-bold text-[var(--md-sys-color-primary)] tabular-nums">{i + 1}</span>
                    <div class="min-w-0">
                      <div class="text-xs font-semibold text-[var(--md-sys-color-on-surface)] truncate">{top.name}</div>
                      {#if top.subText}
                        <div class="text-[11px] text-[var(--md-sys-color-on-surface-variant)] opacity-85 truncate">{top.subText}</div>
                      {/if}
                    </div>
                  </div>
                  <div class="text-[11px] text-[var(--md-sys-color-on-surface-variant)] tabular-nums shrink-0">
                    {top.plays}x • {top.minutes}m
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </section>

        <section class="rounded-lg p-4 bg-[var(--md-sys-color-surface-container-low)] space-y-2.5">
          <h2 class="text-sm font-semibold text-[var(--md-sys-color-on-surface)]">Top Artists</h2>
          {#if !listeningStats.topArtists || listeningStats.topArtists.length === 0}
            <p class="text-xs text-[var(--md-sys-color-on-surface-variant)] opacity-75 py-4">Belum ada data artis terbanyak diputar.</p>
          {:else}
            <div class="space-y-1">
              {#each listeningStats.topArtists.slice(0, 8) as artist, i (artist.name + i)}
                <div class="flex items-center justify-between gap-3 px-2.5 py-2 rounded-md bg-[var(--md-sys-color-surface-container)]">
                  <div class="flex items-center gap-2.5 min-w-0">
                    <span class="w-5 text-xs font-bold text-[var(--md-sys-color-primary)] tabular-nums">{i + 1}</span>
                    <div class="min-w-0">
                      <div class="text-xs font-semibold text-[var(--md-sys-color-on-surface)] truncate">{artist.name}</div>
                      {#if artist.subText}
                        <div class="text-[11px] text-[var(--md-sys-color-on-surface-variant)] opacity-85 truncate">{artist.subText}</div>
                      {/if}
                    </div>
                  </div>
                  <div class="text-[11px] text-[var(--md-sys-color-on-surface-variant)] tabular-nums shrink-0">
                    {artist.plays}x • {artist.minutes}m
                  </div>
                </div>
              {/each}
            </div>
          {/if}
        </section>
      </div>
    {/if}
    </div>
  </div>
{:else if mode === 'listening-stats'}
  <div class="p-6 space-y-6" use:svelteMotionEnter={{ y: 8 }}>
    <div>
      <h1 class="text-2xl font-semibold tracking-tight" style="color: var(--md-sys-color-on-surface);">
        Listening Stats
      </h1>
      <p class="text-xs mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
        Statistik kebiasaan mendengarkan musik
      </p>
    </div>

    <div class="grid grid-cols-2 lg:grid-cols-4 gap-3.5">
      <div class="m3-card p-4 rounded-lg">
        <div class="text-xs flex items-center gap-1.5" style="color: var(--md-sys-color-on-surface-variant);">
          <Icon name="clock" size={14} /> Total Waktu
        </div>
        <div class="text-2xl font-semibold mt-1 tabular-nums">{listeningStats.totalMinutes} mnt</div>
        <div class="text-[11px] mt-0.5 tabular-nums" style="color: var(--md-sys-color-on-surface-variant);">
          {listeningStats.totalPlays} plays • {listeningStats.uniqueTracks} lagu
        </div>
      </div>
      <div class="m3-card p-4 rounded-lg">
        <div class="text-xs flex items-center gap-1.5" style="color: var(--md-sys-color-on-surface-variant);">
          <Icon name="sparkle" size={14} /> Jam Puncak
        </div>
        <div class="text-2xl font-semibold mt-1 tabular-nums">{listeningStats.peakHourLabel}</div>
        <div class="text-[11px] mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
          {listeningStats.peakPersona}
        </div>
      </div>
      <div class="m3-card p-4 rounded-lg">
        <div class="text-xs flex items-center gap-1.5" style="color: var(--md-sys-color-on-surface-variant);">
          <Icon name="calendar" size={14} /> Hari Aktif
        </div>
        <div class="text-2xl font-semibold mt-1">{listeningStats.mostActiveDay}</div>
        <div class="text-[11px] mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
          Berdasarkan total menit
        </div>
      </div>
      <div class="m3-card p-4 rounded-lg">
        <div class="text-xs flex items-center gap-1.5" style="color: var(--md-sys-color-on-surface-variant);">
          <Icon name="flame" weight="fill" size={14} class="text-orange-400" /> Fire Streak
        </div>
        <div class="text-2xl font-semibold text-orange-400 mt-1 tabular-nums">{fireStreak.currentStreak} Hari</div>
        <div class="text-[11px] mt-0.5 tabular-nums" style="color: var(--md-sys-color-on-surface-variant);">
          Rekor: {fireStreak.longestStreak} hari
        </div>
      </div>
    </div>

    <section class="m3-card p-5 rounded-lg space-y-3">
      <h2 class="text-sm font-semibold" style="color: var(--md-sys-color-on-surface);">Time of Day (00:00 – 23:00)</h2>
      <svg viewBox="0 0 720 180" class="w-full h-44">
        {#each listeningStats.timeOfDay as bucket, i (bucket.hour)}
          {@const barH = Math.max(4, (bucket.minutes / maxHourMinutes) * 120)}
          {@const x = i * 29 + 14}
          {@const y = 144 - barH}
          <g>
            <rect
              x={x}
              y={y}
              width="20"
              height={barH}
              rx="6"
              fill={bucket.minutes > 0 ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-surface-container-highest)'}
            />
            {#if i % 3 === 0}
              <text x={x + 10} y="166" text-anchor="middle" fill="currentColor" opacity="0.65" font-size="10">
                {bucket.label.slice(0, 2)}
              </text>
            {/if}
          </g>
        {/each}
      </svg>
    </section>

    <section class="m3-card p-5 rounded-lg space-y-3">
      <h2 class="text-sm font-semibold" style="color: var(--md-sys-color-on-surface);">Day of Week</h2>
      <svg viewBox="0 0 420 180" class="w-full h-44">
        {#each listeningStats.dayOfWeek as day, i (day.dayIndex)}
          {@const barH = Math.max(6, (day.minutes / maxDayMinutes) * 116)}
          {@const x = i * 56 + 22}
          {@const y = 140 - barH}
          <g>
            <rect
              x={x}
              y={y}
              width="34"
              height={barH}
              rx="8"
              fill={day.minutes > 0 ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-surface-container-highest)'}
            />
            <text x={x + 17} y={y - 6} text-anchor="middle" fill="currentColor" opacity="0.7" font-size="10">
              {day.minutes}m
            </text>
            <text x={x + 17} y="164" text-anchor="middle" fill="currentColor" font-size="11" font-weight="600">
              {day.label}
            </text>
          </g>
        {/each}
      </svg>
    </section>
  </div>
{:else if mode === 'fire-streak'}
  <div class="p-6 space-y-6" use:svelteMotionEnter={{ y: 8 }}>
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
            Fire Streak Harian
          </h1>
          <p class="text-xs mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
            Target: {fireStreak.goalMinutes} menit atau {fireStreak.goalTracks} lagu per hari
          </p>
        </div>
      </div>
      <div class="flex items-center gap-6">
        <div class="text-center">
          <div class="text-2xl font-semibold text-orange-400 tabular-nums">{fireStreak.currentStreak}</div>
          <div class="text-[11px]" style="color: var(--md-sys-color-on-surface-variant);">Streak Aktif</div>
        </div>
        <div class="text-center">
          <div class="text-2xl font-semibold tabular-nums">{fireStreak.longestStreak}</div>
          <div class="text-[11px]" style="color: var(--md-sys-color-on-surface-variant);">Rekor</div>
        </div>
      </div>
    </div>

    <div class="m3-card p-5 rounded-lg">
      <h2 class="text-sm font-semibold mb-3" style="color: var(--md-sys-color-on-surface);">
        Kalender Streak 30 Hari Terakhir
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
  <!-- 1:1 NUCLEAR PREFERENCES MAIN PAGE (Matched with Home & M3 Design Tokens) -->
  <div class="h-full min-h-0 flex flex-col overflow-hidden" use:svelteMotionEnter={{ y: 8 }}>
    <div class="flex-1 min-h-0 overflow-y-auto overscroll-contain p-6 space-y-5">
      {#if !prefSearch.trim() && isAppTab(activePrefTab)}
        {#if activePrefTab === 'shortcuts'}
          <div class="w-full flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 class="text-2xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
                Key Shortcuts
              </h2>
              <p class="text-xs mt-1" style="color: var(--md-sys-color-on-surface-variant);">
                Click any shortcut to rebind key combinations
              </p>
            </div>
            <button type="button" onclick={resetAllShortcuts} class="m3-btn-tonal px-4 py-2 text-xs cursor-pointer">
              Reset all to defaults
            </button>
          </div>

          {#if shortcutConflict}
            <div
              class="w-full p-3.5 rounded-lg text-xs font-medium"
              style="background-color: var(--md-sys-color-error-container); color: var(--md-sys-color-on-error-container);"
            >
              {shortcutConflict}
            </div>
          {/if}

          {#each ['Playback', 'General'] as section (section)}
            <div class="w-full space-y-2 pt-2">
              <div
                class="text-[11px] font-semibold uppercase tracking-wider"
                style="color: var(--md-sys-color-on-surface-variant);"
              >
                {section}
              </div>
              {#each KEY_SHORTCUTS_LIST.filter((k) => k.section === section) as sc (sc.id)}
                {@const curKey = getPrefValue(preferences, `shortcut.${sc.id}`) || sc.defaultKey}
                <div class="m3-card px-4 py-3 rounded-lg flex items-center justify-between gap-3">
                  <span class="text-sm font-medium" style="color: var(--md-sys-color-on-surface);">{sc.label}</span>
                  <div class="flex items-center gap-2">
                    <button
                      type="button"
                      onclick={() => (recordingShortcutId = recordingShortcutId === sc.id ? null : sc.id)}
                      onkeydown={(e) => handleShortcutKeyDown(e, sc.id)}
                      class="px-3.5 py-1.5 text-xs font-mono cursor-pointer {recordingShortcutId === sc.id
                        ? 'm3-btn-filled'
                        : 'm3-btn-tonal'}"
                    >
                      {recordingShortcutId === sc.id ? 'Press a key combination...' : curKey}
                    </button>
                    <button
                      type="button"
                      onclick={() => updateSetting(`shortcut.${sc.id}`, sc.defaultKey)}
                      class="m3-btn-outlined px-3 py-1 text-xs cursor-pointer"
                    >
                      Reset
                    </button>
                  </div>
                </div>
              {/each}
            </div>
          {/each}
        {:else if activePrefTab === 'app-plugins'}
          <div class="w-full">
            <h2 class="text-2xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">Plugins</h2>
            <p class="text-xs mt-1" style="color: var(--md-sys-color-on-surface-variant);">
              Active metadata, stream & discovery plugins
            </p>
          </div>
          <div class="w-full space-y-2.5">
            {#each BUILTIN_PLUGINS as plugin (plugin.id)}
              {@const enabled = getPrefValue(preferences, `plugin.${plugin.id}.enabled`) !== 'false'}
              <div class="m3-card p-4 rounded-lg flex items-center justify-between gap-4">
                <div class="space-y-0.5 min-w-0">
                  <div class="flex items-center gap-2">
                    <span class="text-sm font-semibold" style="color: var(--md-sys-color-on-surface);">
                      {plugin.name}
                    </span>
                    <span
                      class="px-2 py-0.5 rounded-full text-[10px] font-semibold"
                      style="background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);"
                    >
                      v{plugin.version}
                    </span>
                  </div>
                  <div class="text-xs" style="color: var(--md-sys-color-on-surface-variant);">
                    {plugin.description} • {plugin.author}
                  </div>
                </div>
                {@render m3Switch(enabled, `Toggle ${plugin.name}`, () =>
                  updateSetting(`plugin.${plugin.id}.enabled`, enabled ? 'false' : 'true')
                )}
              </div>
            {/each}
          </div>
        {:else if activePrefTab === 'themes'}
          <div class="w-full">
            <h2 class="text-2xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
              Material You Themes
            </h2>
            <p class="text-xs mt-1" style="color: var(--md-sys-color-on-surface-variant);">
              Choose a Google Material Design 3 tonal palette and seed color for Velora
            </p>
          </div>
          <div class="w-full grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3.5">
            {#each NUCLEAR_THEMES as th (th.id)}
              {@const isSelected = getPrefValue(preferences, 'theme.active.id') === th.id}
              <button
                type="button"
                onclick={() => applyThemePreset(th)}
                class="m3-card p-4 rounded-lg text-left cursor-pointer space-y-3"
                style={isSelected ? `box-shadow: 0 0 0 2px var(--md-sys-color-primary);` : ''}
              >
                <div class="h-14 rounded-md flex overflow-hidden">
                  <div class="w-1/4 h-full" style="background-color: {th.previewPanel};"></div>
                  <div class="flex-1 h-full p-2.5 flex items-center gap-2" style="background-color: {th.previewBg};">
                    <span class="w-6 h-6 rounded-full shrink-0" style="background-color: {th.accent};"></span>
                    <span class="h-3 flex-1 rounded-full" style="background-color: {th.previewPanel};"></span>
                  </div>
                </div>
                <div>
                  <div class="text-sm font-semibold flex items-center justify-between" style="color: var(--md-sys-color-on-surface);">
                    <span>{th.name}</span>
                    {#if isSelected}
                      <Icon name="check-circle" weight="fill" size={18} style="color: var(--md-sys-color-primary);" />
                    {/if}
                  </div>
                  <div class="text-xs mt-0.5" style="color: var(--md-sys-color-on-surface-variant);">
                    {th.description}
                  </div>
                </div>
              </button>
            {/each}
          </div>
        {:else}
          <div class="w-full">
            <h2 class="text-2xl font-bold tracking-tight" style="color: var(--md-sys-color-on-surface);">
              What's New
            </h2>
            <p class="text-xs mt-1" style="color: var(--md-sys-color-on-surface-variant);">Velora Release Notes</p>
          </div>
          <div class="w-full m3-card p-5 rounded-lg space-y-2.5 text-xs leading-relaxed">
            <div class="text-sm font-semibold" style="color: var(--md-sys-color-on-surface);">
              Velora v2.0 — Material Design 3 Workspace
            </div>
            <ul class="list-disc pl-5 space-y-1" style="color: var(--md-sys-color-on-surface-variant);">
              <li>Google Material Design 3 tonal surfaces, dynamic color seeds, elevated cards, and pill navigation.</li>
              <li>Dynamic resizable Left Menu & Right Studio Panel with smooth morphing animations.</li>
              <li>Right Panel with 3 dedicated cards: Album Lagu, Synchronized Lirik Lagu, and Antrean berikutnya.</li>
              <li>Phosphor Icons + Motion One spring physics across all interactions.</li>
              <li>
                Full Preferences covering Appearance, General, History, Integrations (MCP, MPD, Jam, Discord),
                Playback, Plugins & Updates.
              </li>
            </ul>
          </div>
        {/if}
      {:else}
        <div class="w-full flex items-center justify-between">
          <div>
            <h2 class="text-2xl font-bold capitalize tracking-tight" style="color: var(--md-sys-color-on-surface);">
              {prefSearch.trim() ? `Search results for "${prefSearch}"` : activePrefTab}
            </h2>
            <p class="text-xs mt-1" style="color: var(--md-sys-color-on-surface-variant);">
              Changes apply and persist automatically
            </p>
          </div>
        </div>

        <div class="w-full space-y-2.5">
          {#if visibleSettingsForCategory.length === 0}
            <div class="m3-card p-6 rounded-lg text-xs text-center" style="color: var(--md-sys-color-on-surface-variant);">
              No matching preferences found.
            </div>
          {:else}
            {#each visibleSettingsForCategory as item (item.id)}
              {@render settingControl(item)}
            {/each}
          {/if}
        </div>
      {/if}
    </div>
  </div>
{/if}
