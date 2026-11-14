<script lang="ts">
  import Icon from './Icon.svelte';

  interface Props {
    open: boolean;
    initialIsLocal: boolean;
    onClose: () => void;
    onSubmit: (name: string, description: string, isLocal: boolean, coverColor: string) => void;
  }

  let { open, initialIsLocal, onClose, onSubmit }: Props = $props();

  let dialogEl: HTMLDialogElement | undefined = $state();
  let name = $state('');
  let description = $state('');
  let isLocal = $state(true);
  let coverColor = $state('#6750A4');

  const colors = ['#6750A4', '#0B57D0', '#146C2E', '#9B4052', '#EA580C', '#006874'];

  $effect(() => {
    if (open) {
      isLocal = initialIsLocal;
      name = '';
      description = '';
    }
  });

  $effect(() => {
    if (!dialogEl) return;
    if (open && !dialogEl.open) {
      dialogEl.showModal();
    } else if (!open && dialogEl.open) {
      dialogEl.close();
    }
  });

  function handleFormSubmit(e: Event) {
    e.preventDefault();
    if (!name.trim()) return;
    onSubmit(name.trim(), description.trim(), isLocal, coverColor);
    dialogEl?.close();
  }
</script>

<!-- Google Material Design 3 Dialog (28px corner radius) -->
<dialog
  bind:this={dialogEl}
  onclose={() => onClose()}
  closedby="any"
  aria-labelledby="create-playlist-title"
  class="m-auto w-full max-w-md rounded-lg shadow-2xl p-6"
  style="background-color: var(--md-sys-color-surface-container-high); color: var(--md-sys-color-on-surface);"
>
  <div class="flex items-center justify-between mb-5">
    <h2 id="create-playlist-title" class="text-lg font-semibold">
      {isLocal ? 'Buat Local Playlist' : 'Buat Playlist Baru'}
    </h2>
    <button
      type="button"
      onclick={() => dialogEl?.close()}
      aria-label="Tutup"
      class="m3-icon-btn w-9 h-9 flex items-center justify-center cursor-pointer"
    >
      <Icon name="x" size={16} />
    </button>
  </div>

  <form onsubmit={handleFormSubmit} class="space-y-4">
    <div>
      <label for="pl-name" class="block text-xs font-medium mb-1.5" style="color: var(--md-sys-color-on-surface-variant);">
        Nama Playlist
      </label>
      <input
        id="pl-name"
        type="text"
        bind:value={name}
        required
        placeholder="Misal: Coding Midnight Mix"
        class="m3-input w-full h-11 px-4 rounded-md text-sm outline-none"
      />
    </div>

    <div>
      <label for="pl-desc" class="block text-xs font-medium mb-1.5" style="color: var(--md-sys-color-on-surface-variant);">
        Deskripsi
      </label>
      <textarea
        id="pl-desc"
        rows="2"
        bind:value={description}
        placeholder="Deskripsi singkat playlist..."
        class="m3-input w-full px-4 py-2.5 rounded-md text-sm outline-none resize-none"
      ></textarea>
    </div>

    <div class="flex items-center justify-between">
      <span class="text-xs font-medium" style="color: var(--md-sys-color-on-surface-variant);">Tipe Playlist</span>
      <div class="flex gap-2">
        <button
          type="button"
          onclick={() => (isLocal = true)}
          class="px-3.5 py-1.5 rounded-lg text-xs font-medium cursor-pointer {isLocal ? 'm3-chip-selected' : 'm3-chip'}"
        >
          Local Playlist
        </button>
        <button
          type="button"
          onclick={() => (isLocal = false)}
          class="px-3.5 py-1.5 rounded-lg text-xs font-medium cursor-pointer {!isLocal ? 'm3-chip-selected' : 'm3-chip'}"
        >
          YouTube Music
        </button>
      </div>
    </div>

    <div>
      <span class="block text-xs font-medium mb-2" style="color: var(--md-sys-color-on-surface-variant);">Warna Cover</span>
      <div class="flex items-center gap-2.5">
        {#each colors as c (c)}
          <button
            type="button"
            onclick={() => (coverColor = c)}
            aria-label="Pilih warna {c}"
            class="w-7 h-7 rounded-full transition cursor-pointer {coverColor === c
              ? 'ring-2 ring-white scale-110'
              : 'opacity-70 hover:opacity-100'}"
            style="background-color: {c};"
          ></button>
        {/each}
      </div>
    </div>

    <div class="flex justify-end gap-2 pt-3">
      <button
        type="button"
        onclick={() => dialogEl?.close()}
        class="m3-btn-outlined px-5 py-2 text-xs cursor-pointer"
      >
        Batal
      </button>
      <button
        type="submit"
        class="m3-btn-filled px-5 py-2 text-xs cursor-pointer"
      >
        Simpan
      </button>
    </div>
  </form>
</dialog>
