<script lang="ts">
  import { fade } from 'svelte/transition';
  import { api, resolveCoverUrl, isYTVideoThumb } from '../lib/api';
  import type { Artist, FireStreakStatus, Track, ViewRoute, YTMusicTrack } from '../lib/types';
  import Icon from './Icon.svelte';

  interface Props {
    currentRoute: ViewRoute;
    canGoBack: boolean;
    canGoForward: boolean;
    searchQuery: string;
    fireStreak: FireStreakStatus;
    isDark: boolean;
    accentColor: string;
    tracks?: Track[];
    artists?: Artist[];
    ytResults?: YTMusicTrack[];
    ytLoading?: boolean;
    mediaBaseUrl?: string;
    onBack: () => void;
    onForward: () => void;
    onNavigate: (route: ViewRoute) => void;
    onSearch: (query: string) => void;
    onSuggestSearch?: (query: string) => void;
    onPlayLocalTrack?: (track: Track) => void;
    onPlayYTMusic?: (track: YTMusicTrack) => void;
    onToggleFavoriteArtist?: (artistId: number) => void;
    onToggleTheme: () => void;
    onOpenStreakModal: () => void;
  }

  let {
    currentRoute,
    canGoBack,
    canGoForward,
    searchQuery,
    fireStreak,
    isDark,
    tracks = [],
    artists = [],
    ytResults = [],
    ytLoading = false,
    mediaBaseUrl = '',
    onBack,
    onForward,
    onNavigate,
    onSearch,
    onSuggestSearch,
    onPlayLocalTrack,
    onPlayYTMusic,
    onToggleFavoriteArtist,
    onToggleTheme,
    onOpenStreakModal
  }: Props = $props();

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
          cover: resolveCoverUrl(t.coverUrl, mediaBaseUrl),
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
        cover: resolveCoverUrl(yt.coverUrl, mediaBaseUrl),
        rank: computeSongRank(yt.title, yt.artist, normalizedQuery)
      });
    }

    out.sort((a, b) => a.rank - b.rank || a.title.localeCompare(b.title));
    return out.slice(0, 8);
  });

  // Maksimal 4 sugestion judul lagu
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

  // Maksimal 25 sugestion artist
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
          avatar: resolveCoverUrl(a.avatarUrl, mediaBaseUrl),
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
          ? resolveCoverUrl(localMatch.avatarUrl, mediaBaseUrl)
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
  const dropdownVisible = $derived(
    suggestionsOpen &&
      localQuery.trim().length > 0 &&
      normalizedQuery.length > 0 &&
      (hasAnySuggestions || (!isDebouncing && !ytLoading))
  );

  function submitSearch(q: string) {
    if (debounceTimer) clearTimeout(debounceTimer);
    localQuery = q;
    debouncedQuery = q;
    suggestionsOpen = false;
    onSearch(q);
  }

  function handleInput(e: Event) {
    const val = (e.target as HTMLInputElement).value;
    localQuery = val;
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
    if (e.key === 'Enter') {
      e.preventDefault();
      submitSearch(localQuery);
    } else if (e.key === 'Escape') {
      suggestionsOpen = false;
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

<!-- Borderless Spotify Top App Bar -->
<header
  class="wails-drag relative z-40 h-16 pl-3 pr-0 flex items-center justify-between gap-4 select-none shrink-0 transition-colors duration-300"
  style="background-color: var(--md-sys-color-surface-container-lowest);"
>
  <!-- Left: Navigation History Circular Icon Buttons -->
  <div class="flex items-center gap-3 shrink-0">
    <div class="wails-no-drag flex items-center gap-1">
      <button
        type="button"
        onclick={onBack}
        disabled={!canGoBack}
        aria-label="Kembali"
        class="m3-icon-btn w-10 h-10 rounded-full flex items-center justify-center disabled:opacity-35 disabled:pointer-events-none cursor-pointer"
      >
        <Icon name="caret-left" weight="bold" size={18} />
      </button>
      <button
        type="button"
        onclick={onForward}
        disabled={!canGoForward}
        aria-label="Maju"
        class="m3-icon-btn w-10 h-10 rounded-full flex items-center justify-center disabled:opacity-35 disabled:pointer-events-none cursor-pointer"
      >
        <Icon name="caret-right" weight="bold" size={18} />
      </button>
    </div>
  </div>

  <!-- Center: 1:1 Spotify Home Circle Button + Docked Pill Search Bar -->
  <div class="wails-no-drag flex-1 max-w-xl mx-auto flex items-center gap-2">
    <button
      type="button"
      onclick={() => onNavigate({ name: 'home' })}
      title="Home"
      aria-label="Home"
      class="w-12 h-12 rounded-full flex items-center justify-center shrink-0 cursor-pointer transition-all duration-200 hover:scale-105 hover:brightness-125"
      style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);"
    >
      <Icon name="house" weight={currentRoute.name === 'home' ? 'fill' : 'regular'} size={22} />
    </button>

    <div bind:this={searchContainerEl} class="relative flex-1">
      <div
        class="w-full h-12 px-3.5 flex items-center gap-3 transition-colors duration-200 {dropdownVisible
          ? 'rounded-t-[24px] rounded-b-none'
          : 'rounded-full has-[input:hover]:brightness-110'}"
        style="background-color: var(--md-sys-color-surface-container-high);"
      >
        <Icon
          name="magnifying-glass"
          weight="regular"
          size={22}
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
          placeholder="Apa yang ingin kamu putar?"
          class="w-full h-full bg-transparent text-[14px] font-normal outline-none border-none"
          style="color: var(--md-sys-color-on-surface);"
        />
        {#if localQuery.length > 0}
          <button
            type="button"
            onclick={clearSearch}
            title="Hapus pencarian"
            aria-label="Hapus pencarian"
            class="shrink-0 flex items-center justify-center cursor-pointer transition-opacity hover:opacity-80"
            style="color: var(--md-sys-color-on-surface);"
          >
            <Icon name="x" weight="regular" size={20} />
          </button>
        {/if}
        <span class="w-[1px] h-6 shrink-0" style="background-color: rgba(114, 114, 114, 0.55);"></span>
        <button
          type="button"
          onclick={onOpenStreakModal}
          title="Fire Streak Hari Ini ({fireStreak.currentStreak} Hari)"
          aria-label="Fire Streak {fireStreak.currentStreak}"
          class="h-8 px-2.5 rounded-full flex items-center gap-1.5 shrink-0 text-[13px] font-bold cursor-pointer transition-all duration-200 hover:brightness-125"
          style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);"
        >
          <Icon name="flame" weight="fill" size={18} style="color: var(--md-sys-color-tertiary);" />
          <span class="tabular-nums">{fireStreak.currentStreak}</span>
        </button>
      </div>

      {#if dropdownVisible}
        <div
          transition:fade={{ duration: 130 }}
          class="absolute left-0 right-0 top-full w-full rounded-b-[24px] shadow-2xl overflow-hidden z-50 border-t"
          style="background-color: var(--md-sys-color-surface-container-high); border-color: rgba(128, 128, 128, 0.16);"
        >
          <div class="relative">
            <div class="max-h-[440px] overflow-y-auto overscroll-contain p-3 pb-10 space-y-0.5">

            <!-- 1. Sugestion Judul Lagu (Maksimal 4) -->
            {#if titleSuggestions.length > 0}
              {#each titleSuggestions as title (title)}
                <button
                  type="button"
                  onclick={() => submitSearch(title)}
                  class="w-full flex items-center gap-3 px-2 py-1.5 rounded-lg text-[14px] text-left cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
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

            <!-- 2. Campuran Lagu & Artist (tanpa kategori, pixel-aligned) -->
            {#if mixedSuggestions.length > 0}
              {#each mixedSuggestions as entry (entry.entryType === 'song' ? 's:' + entry.item.kind + ':' + entry.item.title + ':' + entry.item.artist : 'a:' + entry.artist.name)}
                {#if entry.entryType === 'song'}
                  {@const item = entry.item}
                  <button
                    type="button"
                    onclick={() => {
                      suggestionsOpen = false;
                      if (item.kind === 'local') {
                        onPlayLocalTrack?.(item.track);
                      } else {
                        onPlayYTMusic?.(item.track);
                      }
                    }}
                    class="w-full flex items-center gap-3 px-2 py-1.5 rounded-lg text-left cursor-pointer transition-colors hover:bg-[var(--md-sys-color-state-hover)]"
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
                    <Icon
                      name="plus"
                      weight="bold"
                      size={16}
                      class="shrink-0 opacity-75"
                      style="color: var(--md-sys-color-on-surface-variant);"
                    />
                  </button>
                {:else}
                  {@const art = entry.artist}
                  <div class="w-full flex items-center gap-2 px-2 py-1.5 rounded-lg transition-colors hover:bg-[var(--md-sys-color-state-hover)]">
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
                          Artist
                        </div>
                      </div>
                    </button>
                    <button
                      type="button"
                      title={art.isFavorite ? 'Hapus dari Favorit' : 'Tambahkan ke Favorit'}
                      aria-label={art.isFavorite ? 'Hapus dari Favorit' : 'Tambahkan ke Favorit'}
                      onclick={(e) => {
                        e.stopPropagation();
                        if (art.id !== null) {
                          onToggleFavoriteArtist?.(art.id);
                        } else {
                          const key = art.name.toLowerCase();
                          transientFavArtists = {
                            ...transientFavArtists,
                            [key]: !transientFavArtists[key]
                          };
                        }
                      }}
                      class="w-7 h-7 rounded-full flex items-center justify-center shrink-0 cursor-pointer transition-transform hover:scale-110"
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
                  Tidak ada saran ditemukan. Tekan <span class="font-semibold">Enter</span> untuk cari "{localQuery.trim()}".
                </div>
              {/if}
            </div>
            <!-- Gradasi hitam di bagian bawah card seperti panel musik sisi kanan/kiri -->
            <div
              class="pointer-events-none absolute inset-x-0 bottom-0 h-14"
              style={isDark
                ? 'background: linear-gradient(to bottom, transparent 0%, rgba(0, 0, 0, 0.45) 55%, rgba(0, 0, 0, 0.88) 100%);'
                : 'background: linear-gradient(to bottom, transparent 0%, var(--md-sys-color-surface-container-high) 92%);'}
            ></div>
          </div>
        </div>
      {/if}
    </div>
  </div>

  <!-- Right: Taller Unified Controls (44px Action Circles) + Native Windows Caption Controls -->
  <div class="wails-no-drag flex items-center gap-1.5 shrink-0 self-stretch">
    <button
      type="button"
      onclick={() => onNavigate({ name: 'preferences' })}
      title="Nuclear Jam Remote"
      aria-label="Remote QR & Integrations"
      class="w-11 h-11 rounded-full flex items-center justify-center cursor-pointer transition-all duration-200 hover:brightness-125"
      style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface-variant);"
    >
      <Icon name="qr-code" size={19} />
    </button>

    <button
      type="button"
      onclick={onToggleTheme}
      title={isDark ? 'Switch to Material Light' : 'Switch to Material Dark'}
      aria-label="Toggle Theme"
      class="w-11 h-11 rounded-full flex items-center justify-center cursor-pointer transition-all duration-200 hover:brightness-125"
      style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface-variant);"
    >
      <Icon name={isDark ? 'sun' : 'moon'} weight="bold" size={19} />
    </button>

    <div class="flex items-stretch self-stretch ml-2">
      <button
        type="button"
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
  </div>
</header>
