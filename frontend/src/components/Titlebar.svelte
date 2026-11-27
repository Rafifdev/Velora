<script lang="ts">
  import { fade, scale } from 'svelte/transition';
  import { api, isJamRemote, resolveCoverUrl, isYTVideoThumb } from '../lib/api';
  import type { Artist, FireStreakStatus, JamStatus, Track, ViewRoute, YTMusicTrack } from '../lib/types';
  import { tr, type LangCode } from '../lib/i18n';
  import DownloadProgressIcon from './DownloadProgressIcon.svelte';
  import Icon from './Icon.svelte';
  import JamModal from './JamModal.svelte';
  import ModalDialog from './ModalDialog.svelte';

  interface Props {
    currentRoute: ViewRoute;
    canGoBack: boolean;
    canGoForward: boolean;
    searchQuery: string;
    fireStreak: FireStreakStatus;
    accentColor: string;
    tracks?: Track[];
    artists?: Artist[];
    ytResults?: YTMusicTrack[];
    ytLoading?: boolean;
    mediaBaseUrl?: string;
    zoomLevel?: number;
    zoomBadgeVisible?: boolean;
    showStreak?: boolean;
    lang?: LangCode;
    onBack: () => void;
    onForward: () => void;
    onNavigate: (route: ViewRoute) => void;
    onSearch: (query: string) => void;
    onSuggestSearch?: (query: string) => void;
    onPlayLocalTrack?: (track: Track) => void;
    onPlayYTMusic?: (track: YTMusicTrack) => void;
    onToggleFavoriteArtist?: (artistId: number) => void;
    onToggleFavoriteYTMusicArtist?: (payload: {
      externalId: string;
      name: string;
      avatarUrl: string;
    }) => void;
    onOpenStreakModal: () => void;
    onOpenJam?: () => void;
    onCloseJam?: () => void;
    onToggleJam?: () => void;
    jamStatus?: JamStatus;
    jamOpen?: boolean;
    jamLoading?: boolean;
    jamRunning?: boolean;
    jamConnected?: boolean;
    jamPrefEnabled?: boolean;
    activeDownloadsCount?: number;
    downloadProgressPct?: number;
    saveSearches?: boolean;
    onOpenJamSettings?: () => void;
    onResetZoom?: () => void;
  }

  let {
    currentRoute,
    canGoBack,
    canGoForward,
    searchQuery,
    fireStreak,
    tracks = [],
    artists = [],
    ytResults = [],
    ytLoading = false,
    mediaBaseUrl = '',
    zoomLevel = 1,
    zoomBadgeVisible = false,
    showStreak = true,
    lang = 'en',
    onBack,
    onForward,
    onNavigate,
    onSearch,
    onSuggestSearch,
    onPlayLocalTrack,
    onPlayYTMusic,
    onToggleFavoriteArtist,
    onToggleFavoriteYTMusicArtist,
    onOpenStreakModal,
    onOpenJam,
    onCloseJam,
    onToggleJam,
    jamStatus,
    jamOpen = false,
    jamLoading = false,
    jamRunning = false,
    jamConnected = false,
    jamPrefEnabled = false,
    activeDownloadsCount = 0,
    downloadProgressPct = 0,
    saveSearches = true,
    onOpenJamSettings,
    onResetZoom
  }: Props = $props();

  let jamAnchorEl: HTMLElement | undefined = $state();

  $effect(() => {
    if (!jamOpen) return;
    const onPointerDown = (e: PointerEvent) => {
      if (jamAnchorEl && !jamAnchorEl.contains(e.target as Node)) {
        onCloseJam?.();
      }
    };
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onCloseJam?.();
    };
    document.addEventListener('pointerdown', onPointerDown, true);
    document.addEventListener('keydown', onKeyDown, true);
    return () => {
      document.removeEventListener('pointerdown', onPointerDown, true);
      document.removeEventListener('keydown', onKeyDown, true);
    };
  });

  const SEARCH_DEBOUNCE_MS = 300;

  let localQuery = $state('');
  let debouncedQuery = $state('');
  let debounceTimer: ReturnType<typeof setTimeout> | null = null;
  let isMaximized = $state(false);
  let suggestionsOpen = $state(false);
  let searchContainerEl: HTMLDivElement | undefined = $state();

  async function syncMaximizedState() {
    isMaximized = await api.windowIsMaximized();
  }

  async function handleToggleMaximize() {
    isMaximized = !isMaximized;
    api.windowMaximize();
    setTimeout(syncMaximizedState, 120);
  }

  $effect(() => {
    localQuery = searchQuery;
    debouncedQuery = searchQuery;
  });

  $effect(() => {
    void syncMaximizedState();
    const handleOutsideClick = (ev: MouseEvent) => {
      if (searchContainerEl && !searchContainerEl.contains(ev.target as Node)) {
        suggestionsOpen = false;
      }
    };
    window.addEventListener('resize', syncMaximizedState);
    window.addEventListener('pointerdown', handleOutsideClick);
    return () => {
      if (debounceTimer) clearTimeout(debounceTimer);
      window.removeEventListener('resize', syncMaximizedState);
      window.removeEventListener('pointerdown', handleOutsideClick);
    };
  });

  const isDebouncing = $derived(localQuery.trim() !== debouncedQuery.trim());
  const normalizedQuery = $derived(debouncedQuery.trim().toLowerCase());

  function splitHighlight(text: string, query: string): Array<{ text: string; match: boolean }> {
    const q = query.trim();
    if (!q) return [{ text, match: false }];
    const idx = text.toLowerCase().indexOf(q.toLowerCase());
    if (idx === -1) return [{ text, match: false }];
    const parts: Array<{ text: string; match: boolean }> = [];
    if (idx > 0) parts.push({ text: text.slice(0, idx), match: false });
    parts.push({ text: text.slice(idx, idx + q.length), match: true });
    if (idx + q.length < text.length) {
      parts.push({ text: text.slice(idx + q.length), match: false });
    }
    return parts;
  }

  function prefixRank(text: string, query: string): number {
    const lower = text.trim().toLowerCase();
    if (!lower) return 5;
    if (lower === query) return 0;
    if (lower.startsWith(query)) return 1;
    if (lower.split(/[\s\-_()[\]]+/).some((w) => w.startsWith(query))) return 2;
    if (lower.includes(query)) return 3;
    return 4;
  }

  type SongSuggestion =
    | { kind: 'local'; track: Track; title: string; artist: string; cover: string; rank: number }
    | { kind: 'yt'; track: YTMusicTrack; title: string; artist: string; cover: string; rank: number };

  function computeSongRank(title: string, artist: string, query: string): number {
    const tRank = prefixRank(title, query);
    if (tRank <= 2) return tRank;
    const aRank = prefixRank(artist, query);
    if (aRank <= 2) return aRank + 0.5;
    return Math.min(tRank, aRank + 0.5);
  }

  const songSuggestions = $derived.by<SongSuggestion[]>(() => {
    if (!normalizedQuery) return [];
    const out: SongSuggestion[] = [];
    const seen = new Set<string>();

    for (const t of tracks) {
      if (
        t.title.toLowerCase().includes(normalizedQuery) ||
        t.artistName.toLowerCase().includes(normalizedQuery)
      ) {
        const key = `${t.title.toLowerCase()}::${t.artistName.toLowerCase()}`;
        if (seen.has(key)) continue;
        seen.add(key);
        out.push({
          kind: 'local',
          track: t,
          title: t.title,
          artist: t.artistName,
          cover: resolveCoverUrl(t.coverUrl, mediaBaseUrl, 96),
          rank: computeSongRank(t.title, t.artistName, normalizedQuery)
        });
      }
    }

    for (const yt of ytResults) {
      const key = `${yt.title.toLowerCase()}::${yt.artist.toLowerCase()}`;
      if (seen.has(key)) continue;
      seen.add(key);
      out.push({
        kind: 'yt',
        track: yt,
        title: yt.title,
        artist: yt.artist,
        cover: resolveCoverUrl(yt.coverUrl, mediaBaseUrl, 96),
        rank: computeSongRank(yt.title, yt.artist, normalizedQuery)
      });
    }

    out.sort((a, b) => a.rank - b.rank || a.title.localeCompare(b.title));
    return out.slice(0, 8);
  });

  const titleSuggestions = $derived.by<string[]>(() => {
    if (!normalizedQuery) return [];
    const titles: string[] = [];
    const seen = new Set<string>();
    for (const s of songSuggestions) {
      const clean = s.title.trim();
      const key = clean.toLowerCase();
      if (!clean || seen.has(key)) continue;
      seen.add(key);
      titles.push(clean);
      if (titles.length >= 4) break;
    }
    return titles;
  });

  let transientFavArtists = $state<Record<string, boolean>>({});

  const artistSuggestions = $derived.by<Array<{ id: number | null; name: string; avatar: string; isFavorite: boolean; rank: number }>>(() => {
    if (!normalizedQuery) return [];
    const out: Array<{ id: number | null; name: string; avatar: string; isFavorite: boolean; rank: number }> = [];
    const seen = new Set<string>();

    for (const a of artists) {
      const name = a.name.trim();
      const key = name.toLowerCase();
      if (!name || seen.has(key)) continue;
      if (key.includes(normalizedQuery)) {
        seen.add(key);
        out.push({
          id: a.id,
          name,
          avatar: resolveCoverUrl(a.avatarUrl, mediaBaseUrl, 96),
          isFavorite: a.isFavorite,
          rank: prefixRank(name, normalizedQuery)
        });
      }
    }

    for (const s of songSuggestions) {
      const parts = s.artist
        .split(/[,&•]|feat\.|ft\./i)
        .map((p) => p.trim())
        .filter(Boolean);
      for (const p of parts) {
        const key = p.toLowerCase();
        if (seen.has(key)) continue;
        seen.add(key);
        const localMatch = artists.find((a) => a.name.toLowerCase() === key);
        const avatar = localMatch?.avatarUrl
          ? resolveCoverUrl(localMatch.avatarUrl, mediaBaseUrl, 96)
          : s.cover;
        out.push({
          id: localMatch?.id ?? null,
          name: p,
          avatar,
          isFavorite: localMatch ? localMatch.isFavorite : Boolean(transientFavArtists[key]),
          rank: prefixRank(p, normalizedQuery)
        });
      }
    }

    out.sort((a, b) => a.rank - b.rank || a.name.localeCompare(b.name));
    return out.slice(0, 25);
  });

  type MixedSuggestion =
    | { entryType: 'song'; item: SongSuggestion; rank: number; label: string }
    | { entryType: 'artist'; artist: { id: number | null; name: string; avatar: string; isFavorite: boolean; rank: number }; rank: number; label: string };

  const mixedSuggestions = $derived.by<MixedSuggestion[]>(() => {
    const merged: MixedSuggestion[] = [
      ...songSuggestions.map((item) => ({
        entryType: 'song' as const,
        item,
        rank: item.rank,
        label: item.title
      })),
      ...artistSuggestions.map((artist) => ({
        entryType: 'artist' as const,
        artist,
        rank: artist.rank,
        label: artist.name
      }))
    ];
    merged.sort((a, b) => a.rank - b.rank || a.label.localeCompare(b.label));
    return merged;
  });

  const hasAnySuggestions = $derived(
    titleSuggestions.length > 0 || songSuggestions.length > 0 || artistSuggestions.length > 0
  );
  const totalDropdownItems = $derived(titleSuggestions.length + mixedSuggestions.length);
  const dropdownVisible = $derived(
    suggestionsOpen &&
      localQuery.trim().length > 0 &&
      normalizedQuery.length > 0 &&
      (hasAnySuggestions || (!isDebouncing && !ytLoading))
  );

  let activeSuggestionIdx = $state(-1);
  let dropdownListEl: HTMLDivElement | undefined = $state();

  $effect(() => {
    if (!dropdownVisible || totalDropdownItems === 0) {
      activeSuggestionIdx = -1;
    } else if (activeSuggestionIdx >= totalDropdownItems) {
      activeSuggestionIdx = totalDropdownItems - 1;
    }
  });

  function scrollActiveSuggestionIntoView(idx: number) {
    requestAnimationFrame(() => {
      const el = dropdownListEl?.querySelector(`[data-search-idx="${idx}"]`) as HTMLElement | null;
      el?.scrollIntoView({ block: 'nearest' });
    });
  }

  function activateSuggestionAt(idx: number) {
    if (idx < 0 || idx >= totalDropdownItems) return;
    if (idx < titleSuggestions.length) {
      submitSearch(titleSuggestions[idx]);
      return;
    }
    const entry = mixedSuggestions[idx - titleSuggestions.length];
    if (!entry) return;
    suggestionsOpen = false;
    activeSuggestionIdx = -1;
    if (entry.entryType === 'song') {
      if (entry.item.kind === 'local') {
        onPlayLocalTrack?.(entry.item.track);
      } else {
        onPlayYTMusic?.(entry.item.track);
      }
    } else {
      if (entry.artist.id !== null) {
        onNavigate({ name: 'artist-detail', artistId: entry.artist.id });
      } else {
        submitSearch(entry.artist.name);
      }
    }
  }

  let jamDisabledAlert = $state(false);

  function submitSearch(q: string) {
    if (debounceTimer) clearTimeout(debounceTimer);
    localQuery = q;
    debouncedQuery = q;
    suggestionsOpen = false;
    activeSuggestionIdx = -1;
    const trimmed = q.trim();
    if (trimmed) {
      if (saveSearches) {
        try {
          const prev = JSON.parse(localStorage.getItem('velora.recentSearches') || '[]') as string[];
          const next = [trimmed, ...prev.filter((item) => item.toLowerCase() !== trimmed.toLowerCase())].slice(0, 8);
          localStorage.setItem('velora.recentSearches', JSON.stringify(next));
        } catch {
        }
      } else {
        try {
          localStorage.removeItem('velora.recentSearches');
        } catch {
        }
      }
    }
    onSearch(q);
  }

  function handleInput(e: Event) {
    const val = (e.target as HTMLInputElement).value;
    localQuery = val;
    activeSuggestionIdx = -1;
    if (debounceTimer) clearTimeout(debounceTimer);
    if (!val.trim()) {
      debouncedQuery = '';
      suggestionsOpen = false;
      return;
    }
    suggestionsOpen = true;
    debounceTimer = setTimeout(() => {
      debouncedQuery = val;
      onSuggestSearch?.(val);
    }, SEARCH_DEBOUNCE_MS);
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'ArrowDown') {
      if (dropdownVisible && totalDropdownItems > 0) {
        e.preventDefault();
        activeSuggestionIdx = activeSuggestionIdx < totalDropdownItems - 1 ? activeSuggestionIdx + 1 : 0;
        scrollActiveSuggestionIntoView(activeSuggestionIdx);
      }
      return;
    }
    if (e.key === 'ArrowUp') {
      if (dropdownVisible && totalDropdownItems > 0) {
        e.preventDefault();
        activeSuggestionIdx = activeSuggestionIdx > 0 ? activeSuggestionIdx - 1 : totalDropdownItems - 1;
        scrollActiveSuggestionIntoView(activeSuggestionIdx);
      }
      return;
    }
    if (e.key === 'Enter') {
      e.preventDefault();
      if (dropdownVisible && activeSuggestionIdx >= 0 && activeSuggestionIdx < totalDropdownItems) {
        activateSuggestionAt(activeSuggestionIdx);
      } else {
        submitSearch(localQuery);
      }
    } else if (e.key === 'Escape') {
      suggestionsOpen = false;
      activeSuggestionIdx = -1;
    }
  }

  function clearSearch() {
    if (debounceTimer) clearTimeout(debounceTimer);
    localQuery = '';
    debouncedQuery = '';
    suggestionsOpen = false;
    if (currentRoute.name === 'search') {
      onSearch('');
    }
  }
</script>

<header
  class="wails-drag relative z-40 hidden sm:flex h-14 pl-3 pr-0 items-center justify-between gap-4 select-none shrink-0 transition-colors duration-300"
  style="background-color: var(--md-sys-color-surface-container-lowest);"
>
  <div class="flex items-center gap-3 shrink-0">
    <div class="wails-no-drag flex items-center gap-1">
      <button
        type="button"
        onclick={onBack}
        disabled={!canGoBack}
        aria-label={tr(lang, 'nav.back')}
        class="m3-icon-btn w-9 h-9 rounded-full flex items-center justify-center disabled:opacity-35 disabled:pointer-events-none cursor-pointer"
      >
        <Icon name="caret-left" weight="bold" size={17} />
      </button>
      <button
        type="button"
        onclick={onForward}
        disabled={!canGoForward}
        aria-label={tr(lang, 'nav.forward')}
        class="m3-icon-btn w-9 h-9 rounded-full flex items-center justify-center disabled:opacity-35 disabled:pointer-events-none cursor-pointer"
      >
        <Icon name="caret-right" weight="bold" size={17} />
      </button>
    </div>
  </div>

  <div class="wails-no-drag flex-1 min-w-0 max-w-xl mx-auto flex items-center gap-2">
    <button
      type="button"
      onclick={() => onNavigate({ name: 'home' })}
      title={tr(lang, 'nav.home')}
      aria-label={tr(lang, 'nav.home')}
      class="hidden sm:flex w-10 h-10 rounded-full items-center justify-center shrink-0 cursor-pointer transition-all duration-200 hover:scale-105 hover:brightness-125"
      style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);"
    >
      <Icon name="house" weight={currentRoute.name === 'home' ? 'fill' : 'regular'} size={19} />
    </button>

    <div bind:this={searchContainerEl} class="relative flex-1">
      <div
        class="w-full h-10 px-3.5 flex items-center gap-2.5 transition-colors duration-200 {dropdownVisible
          ? 'rounded-t-[20px] rounded-b-none'
          : 'rounded-full has-[input:hover]:brightness-110'}"
        style="background-color: var(--md-sys-color-surface-container-high);"
      >
        <Icon
          name="magnifying-glass"
          weight="regular"
          size={19}
          class="shrink-0"
          style="color: var(--md-sys-color-on-surface-variant);"
        />
        <input
          type="search"
          value={localQuery}
          onfocus={() => {
            if (localQuery.trim().length > 0) suggestionsOpen = true;
          }}
          oninput={handleInput}
          onkeydown={handleKeyDown}
          placeholder={tr(lang, 'nav.searchPlaceholder')}
          class="w-full h-full bg-transparent text-[14px] font-normal outline-none border-none"
          style="color: var(--md-sys-color-on-surface);"
        />
        {#if localQuery.length > 0}
          <button
            type="button"
            onclick={clearSearch}
            title={tr(lang, 'nav.clearSearch')}
            aria-label={tr(lang, 'nav.clearSearch')}
            class="shrink-0 flex items-center justify-center cursor-pointer transition-opacity hover:opacity-80"
            style="color: var(--md-sys-color-on-surface);"
          >
            <Icon name="x" weight="regular" size={20} />
          </button>
        {/if}
        {#if showStreak}
          <span class="w-[1px] h-6 shrink-0" style="background-color: rgba(114, 114, 114, 0.55);"></span>
          <button
            type="button"
            onclick={onOpenStreakModal}
            title="{tr(lang, 'nav.dailyStreak')} ({fireStreak.currentStreak} {tr(lang, 'nav.days')})"
            aria-label="{tr(lang, 'nav.dailyStreak')} {fireStreak.currentStreak}"
            class="h-8 px-2.5 rounded-full flex items-center gap-1.5 shrink-0 text-[13px] font-bold cursor-pointer transition-all duration-200 hover:brightness-125"
            style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);"
          >
            <Icon name="flame" weight="fill" size={18} style="color: var(--md-sys-color-tertiary);" />
            <span class="tabular-nums">{fireStreak.currentStreak}</span>
          </button>
        {/if}
      </div>

      {#if dropdownVisible}
        <div
          transition:fade={{ duration: 130 }}
          class="absolute left-0 right-0 top-full w-full rounded-b-[24px] shadow-2xl overflow-hidden z-50 border-t"
          style="background-color: var(--md-sys-color-surface-container-high); border-color: rgba(128, 128, 128, 0.16);"
        >
          <div class="relative">
            <div bind:this={dropdownListEl} class="max-h-[520px] overflow-y-auto overscroll-contain p-3 pb-7 space-y-0.5">

            {#if titleSuggestions.length > 0}
              {#each titleSuggestions as title, tIdx (title)}
                <button
                  type="button"
                  data-search-idx={tIdx}
                  onmouseenter={() => (activeSuggestionIdx = tIdx)}
                  onclick={() => submitSearch(title)}
                  class="w-full flex items-center gap-3 px-2 py-1.5 rounded-lg text-[14px] text-left cursor-pointer transition-colors {activeSuggestionIdx === tIdx
                    ? 'bg-[var(--md-sys-color-state-active)]'
                    : 'hover:bg-[var(--md-sys-color-state-hover)]'}"
                >
                  <div class="w-10 h-8 flex items-center justify-center shrink-0">
                    <Icon
                      name="magnifying-glass"
                      size={18}
                      style="color: var(--md-sys-color-on-surface-variant);"
                    />
                  </div>
                  <span class="truncate min-w-0 flex-1">
                    {#each splitHighlight(title, normalizedQuery) as seg, i (i)}
                      <span
                        class={seg.match ? 'font-bold' : 'font-normal'}
                        style="color: {seg.match
                          ? 'var(--md-sys-color-on-surface)'
                          : 'var(--md-sys-color-on-surface-variant)'};"
                      >{seg.text}</span>
                    {/each}
                  </span>
                </button>
              {/each}
            {/if}

            {#if mixedSuggestions.length > 0}
              {#each mixedSuggestions as entry, mIdx (entry.entryType === 'song' ? 's:' + entry.item.kind + ':' + entry.item.title + ':' + entry.item.artist : 'a:' + entry.artist.name)}
                {@const itemIdx = titleSuggestions.length + mIdx}
                {#if entry.entryType === 'song'}
                  {@const item = entry.item}
                  <button
                    type="button"
                    data-search-idx={itemIdx}
                    onmouseenter={() => (activeSuggestionIdx = itemIdx)}
                    onclick={() => {
                      suggestionsOpen = false;
                      activeSuggestionIdx = -1;
                      if (item.kind === 'local') {
                        onPlayLocalTrack?.(item.track);
                      } else {
                        onPlayYTMusic?.(item.track);
                      }
                    }}
                    class="w-full flex items-center gap-3 px-2 py-1.5 rounded-lg text-left cursor-pointer transition-colors {activeSuggestionIdx === itemIdx
                      ? 'bg-[var(--md-sys-color-state-active)]'
                      : 'hover:bg-[var(--md-sys-color-state-hover)]'}"
                  >
                    <div class="w-10 h-10 rounded-md overflow-hidden shrink-0 bg-[var(--md-sys-color-surface-container-highest)]">
                      {#if item.cover}
                        <img
                          src={item.cover}
                          alt={item.title}
                          class="block w-full h-full object-cover object-center"
                          class:velora-yt-video-thumb={isYTVideoThumb(item.cover)}
                        />
                      {:else}
                        <div class="w-full h-full flex items-center justify-center opacity-50">
                          <Icon name="music-note" size={16} />
                        </div>
                      {/if}
                    </div>
                    <div class="min-w-0 flex-1">
                      <div class="text-[14px] leading-snug truncate">
                        {#each splitHighlight(item.title, normalizedQuery) as seg, i (i)}
                          <span
                            class={seg.match ? 'font-bold' : 'font-medium'}
                            style="color: {seg.match
                              ? 'var(--md-sys-color-on-surface)'
                              : 'var(--md-sys-color-on-surface-variant)'};"
                          >{seg.text}</span>
                        {/each}
                      </div>
                      <div class="text-[12px] leading-snug truncate" style="color: var(--md-sys-color-on-surface-variant);">
                        {item.artist}
                      </div>
                    </div>
                    <span
                      class="w-7 h-7 flex items-center justify-center shrink-0 opacity-75"
                      style="color: var(--md-sys-color-on-surface-variant);"
                    >
                      <Icon name="plus" weight="bold" size={16} />
                    </span>
                  </button>
                {:else}
                  {@const art = entry.artist}
                  <div
                    data-search-idx={itemIdx}
                    onmouseenter={() => (activeSuggestionIdx = itemIdx)}
                    class="w-full flex items-center gap-2 px-2 py-1.5 rounded-lg transition-colors {activeSuggestionIdx === itemIdx
                      ? 'bg-[var(--md-sys-color-state-active)]'
                      : 'hover:bg-[var(--md-sys-color-state-hover)]'}"
                  >
                    <button
                      type="button"
                      onclick={() => {
                        suggestionsOpen = false;
                        if (art.id !== null) {
                          onNavigate({ name: 'artist-detail', artistId: art.id });
                        } else {
                          submitSearch(art.name);
                        }
                      }}
                      class="flex-1 min-w-0 flex items-center gap-3 text-left cursor-pointer"
                    >
                      <div class="w-10 h-10 rounded-full overflow-hidden shrink-0 bg-[var(--md-sys-color-surface-container-highest)]">
                        {#if art.avatar}
                          <img
                            src={art.avatar}
                            alt={art.name}
                            class="block w-full h-full object-cover object-center"
                          />
                        {:else}
                          <div class="w-full h-full flex items-center justify-center opacity-50">
                            <Icon name="user" size={18} />
                          </div>
                        {/if}
                      </div>
                      <div class="min-w-0 flex-1">
                        <div class="text-[14px] leading-snug truncate">
                          {#each splitHighlight(art.name, normalizedQuery) as seg, i (i)}
                            <span
                              class={seg.match ? 'font-bold' : 'font-medium'}
                              style="color: {seg.match
                                ? 'var(--md-sys-color-on-surface)'
                                : 'var(--md-sys-color-on-surface-variant)'};"
                            >{seg.text}</span>
                          {/each}
                        </div>
                        <div class="text-[12px] leading-snug truncate" style="color: var(--md-sys-color-on-surface-variant);">
                          {tr(lang, 'nav.artist')}
                        </div>
                      </div>
                    </button>
                    <button
                      type="button"
                      title={art.isFavorite ? tr(lang, 'nav.removeFromFavorites') : tr(lang, 'nav.addToFavorites')}
                      aria-label={art.isFavorite ? tr(lang, 'nav.removeFromFavorites') : tr(lang, 'nav.addToFavorites')}
                      onclick={(e) => {
                        e.stopPropagation();
                        if (art.id !== null) {
                          onToggleFavoriteArtist?.(art.id);
                        } else if (onToggleFavoriteYTMusicArtist) {
                          onToggleFavoriteYTMusicArtist({
                            externalId: '',
                            name: art.name,
                            avatarUrl: art.avatar
                          });
                        } else {
                          const key = art.name.toLowerCase();
                          transientFavArtists = {
                            ...transientFavArtists,
                            [key]: !transientFavArtists[key]
                          };
                        }
                      }}
                      class="velora-heart-btn w-7 h-7 rounded-full flex items-center justify-center shrink-0 cursor-pointer {art.isFavorite ? 'is-loved' : ''}"
                      style="color: {art.isFavorite ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-on-surface-variant)'};"
                    >
                      <Icon name="heart" weight={art.isFavorite ? 'fill' : 'regular'} size={17} />
                    </button>
                  </div>
                {/if}
              {/each}
            {/if}

              {#if !isDebouncing && !ytLoading && titleSuggestions.length === 0 && songSuggestions.length === 0 && artistSuggestions.length === 0}
                <div class="py-4 px-2 text-xs text-center" style="color: var(--md-sys-color-on-surface-variant);">
                  {tr(lang, 'nav.noSuggestions')} "{localQuery.trim()}".
                </div>
              {/if}
            </div>
            <div
              class="pointer-events-none absolute inset-x-0 bottom-0 h-10"
              style="background: linear-gradient(to bottom, transparent 0%, color-mix(in srgb, var(--md-sys-color-surface-container-high) 35%, transparent) 40%, color-mix(in srgb, var(--md-sys-color-surface-container-high) 78%, transparent) 74%, var(--md-sys-color-surface-container-high) 100%);"
            ></div>
          </div>
        </div>
      {/if}
    </div>
  </div>

  <div class="wails-no-drag relative flex items-center gap-1.5 shrink-0 self-stretch">
    {#if zoomBadgeVisible}
      <button
        type="button"
        transition:scale={{ start: 0.88, duration: 180 }}
        onclick={() => onResetZoom?.()}
        title="Reset Zoom (Ctrl+0)"
        aria-label="Indikator Zoom"
        class="absolute right-full mr-1.5 top-1/2 -translate-y-1/2 h-9 px-3 rounded-full flex items-center justify-center gap-1.5 text-[11px] font-bold leading-none tabular-nums whitespace-nowrap cursor-pointer transition-all duration-200 hover:brightness-125"
        style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface-variant);"
      >
        <Icon
          name={zoomLevel < 1 ? 'magnifying-glass-minus' : zoomLevel > 1 ? 'magnifying-glass-plus' : 'magnifying-glass'}
          size={15}
        />
        <span>{Math.round(zoomLevel * 100)}%</span>
      </button>
    {/if}

    {#if isJamRemote}
      <div
        class="h-9 px-2.5 sm:px-3.5 rounded-full flex items-center gap-1.5 text-xs font-semibold select-none shrink-0"
        style={jamConnected
          ? 'background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);'
          : 'background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface-variant);'}
      >
        <Icon name={jamConnected ? 'check-circle' : 'wifi-high'} weight={jamConnected ? 'fill' : 'regular'} size={15} />
        <span class="hidden sm:inline">{jamConnected ? 'Connected' : 'Connecting'}</span>
      </div>
    {:else}
      <div class="relative" bind:this={jamAnchorEl}>
        <button
          type="button"
          onclick={() => {
            if (!jamPrefEnabled) {
              jamDisabledAlert = true;
              return;
            }
            if (jamOpen) onCloseJam?.();
            else if (onOpenJam) onOpenJam();
            else onNavigate({ name: 'preferences' });
          }}
          title={!jamPrefEnabled
            ? lang === 'en'
              ? 'Velora Jam (Beta) — Permission disabled'
              : 'Velora Jam (Beta) — Izin Nonaktif'
            : jamRunning
              ? lang === 'en'
                ? 'Velora Jam (Beta) — Active'
                : 'Velora Jam (Beta) — Remote (Aktif)'
              : 'Velora Jam (Beta)'}
          aria-label="Remote QR & Integrations"
          class="relative w-9 h-9 rounded-full flex items-center justify-center cursor-pointer transition-all duration-300 ease-out hover:brightness-125 {!jamPrefEnabled ? 'opacity-45' : ''}"
          style={!jamPrefEnabled
            ? 'background-color: var(--md-sys-color-surface-container); color: var(--md-sys-color-on-surface-variant);'
            : jamRunning
              ? 'background-color: var(--md-sys-color-primary-container); color: var(--md-sys-color-on-primary-container);'
              : `background-color: var(--md-sys-color-surface-container-high); color: ${jamOpen ? 'var(--md-sys-color-primary)' : 'var(--md-sys-color-on-surface-variant)'};`}
        >
          <Icon name="qr-code" size={17} />
        </button>

        <ModalDialog
          open={jamDisabledAlert}
          onClose={() => (jamDisabledAlert = false)}
          ariaLabelledby="jam-disabled-title"
          ariaDescribedby="jam-disabled-desc"
          maxWidthClass="max-w-md"
          class="p-6"
        >
          <div class="mb-3">
            <h2 id="jam-disabled-title" class="text-lg font-bold tracking-tight">
              {tr(lang, 'nav.jamPermDisabledTitle')}
            </h2>
          </div>
          <p id="jam-disabled-desc" class="text-xs leading-relaxed" style="color: var(--md-sys-color-on-surface-variant);">
            {tr(lang, 'nav.jamPermDisabledDesc')}
          </p>
          <div class="flex items-center justify-end gap-2 mt-6">
            <button
              type="button"
              onclick={() => (jamDisabledAlert = false)}
              class="h-10 px-5 rounded-full bg-[var(--md-sys-color-surface-container)] hover:bg-[var(--md-sys-color-surface-container-highest)] text-xs font-semibold cursor-pointer transition-colors"
            >
              {tr(lang, 'nav.close')}
            </button>
            <button
              type="button"
              onclick={() => {
                jamDisabledAlert = false;
                if (onOpenJamSettings) onOpenJamSettings();
                else onNavigate({ name: 'preferences' });
              }}
              class="h-10 px-7 rounded-full text-xs font-bold cursor-pointer transition-transform duration-150 hover:scale-[1.03] active:scale-95"
              style="background-color: var(--md-sys-color-on-surface); color: var(--md-sys-color-surface);"
            >
              {tr(lang, 'nav.openSettings')}
            </button>
          </div>
        </ModalDialog>

        {#if jamStatus}
          <JamModal
            open={jamOpen}
            status={jamStatus}
            loading={jamLoading}
            {lang}
            onClose={() => onCloseJam?.()}
            onToggleJam={() => onToggleJam?.()}
          />
        {/if}
      </div>
    {/if}

    <button
      type="button"
      onpointerdown={(e) => e.stopPropagation()}
      onclick={() => onNavigate({ name: 'downloads' })}
      aria-label={tr(lang, 'nav.downloaded')}
      title={tr(lang, 'nav.downloaded')}
      class="velora-press wails-no-drag relative w-9 h-9 rounded-full flex items-center justify-center cursor-pointer transition-colors hover:brightness-110"
      style="background-color: var(--md-sys-color-surface-container-high); color: {currentRoute.name === 'downloads' || activeDownloadsCount > 0
        ? 'var(--md-sys-color-primary)'
        : 'var(--md-sys-color-on-surface-variant)'};"
    >
      <DownloadProgressIcon {activeDownloadsCount} {downloadProgressPct} iconSize={18} />
    </button>

    {#if !isJamRemote}
      <div class="wails-no-drag relative z-[60] flex items-stretch self-stretch ml-2">
        <button
          type="button"
          onpointerdown={(e) => e.stopPropagation()}
          onclick={() => api.windowMinimize()}
          title="Minimize"
          aria-label="Minimize"
          class="win-caption-btn"
        >
          <svg width="10" height="10" viewBox="0 0 10 10" fill="none" aria-hidden="true">
            <line x1="0" y1="5.5" x2="10" y2="5.5" stroke="currentColor" stroke-width="1" />
          </svg>
        </button>
        <button
          type="button"
          onpointerdown={(e) => e.stopPropagation()}
          onclick={handleToggleMaximize}
          title={isMaximized ? 'Restore Down' : 'Maximize'}
          aria-label={isMaximized ? 'Restore Down' : 'Maximize'}
          class="win-caption-btn"
        >
          {#if isMaximized}
            <svg width="10" height="10" viewBox="0 0 10 10" fill="none" aria-hidden="true">
              <rect x="0.5" y="2.5" width="7" height="7" stroke="currentColor" stroke-width="1" />
              <path d="M2.5 2.5V0.5H9.5V7.5H7.5" stroke="currentColor" stroke-width="1" />
            </svg>
          {:else}
            <svg width="10" height="10" viewBox="0 0 10 10" fill="none" aria-hidden="true">
              <rect x="0.5" y="0.5" width="9" height="9" stroke="currentColor" stroke-width="1" />
            </svg>
          {/if}
        </button>
        <button
          type="button"
          onpointerdown={(e) => e.stopPropagation()}
          onclick={() => api.windowClose()}
          title="Close"
          aria-label="Close"
          class="win-caption-btn win-caption-btn-close"
        >
          <svg width="10" height="10" viewBox="0 0 10 10" fill="none" aria-hidden="true">
            <path d="M1 1L9 9M9 1L1 9" stroke="currentColor" stroke-width="1.15" stroke-linecap="butt" />
          </svg>
        </button>
      </div>
    {/if}
  </div>
</header>
