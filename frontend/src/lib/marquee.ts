import type { Action } from 'svelte/action';

export interface VeloraMarqueeParams {
  text?: string;
  disabled?: boolean;
}

/**
 * Svelte action that applies the Dynamic Island marquee animation
 * (`dynamic_island_windows.go`) to an overflowing single-line text element.
 * Attach to the `overflow-hidden` wrapper element whose first child (or
 * `[data-marquee-text]` element) is `inline-block whitespace-nowrap`.
 */
export const veloraMarquee: Action<HTMLElement, VeloraMarqueeParams | undefined> = (
  wrap,
  initialParams = {}
) => {
  let params = initialParams;
  let anim: Animation | null = null;
  let rafId = 0;

  const stop = () => {
    if (rafId) {
      cancelAnimationFrame(rafId);
      rafId = 0;
    }
    anim?.cancel();
    anim = null;
  };

  const startOrUpdate = () => {
    anim?.cancel();
    anim = null;
    if (params?.disabled) return;

    const reducedMotion =
      typeof window !== 'undefined' &&
      window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (reducedMotion) return;

    const text =
      (wrap.querySelector('[data-marquee-text]') as HTMLElement | null) ??
      (wrap.firstElementChild as HTMLElement | null);
    if (!text) return;

    const overflow = Math.ceil(text.scrollWidth - wrap.clientWidth);
    if (overflow <= 2) return;

    // Samakan persis dengan animasi marquee Dynamic Island (dynamic_island_windows.go)
    const shift = overflow + 4;
    const scrollDur = Math.max(1.4, shift / 28.0);
    const pauseStart = 2.2;
    const pauseEnd = 2.0;
    const fadeOutDur = 0.24;
    const fadeInDur = 0.24;
    const cycle = pauseStart + scrollDur + pauseEnd + fadeOutDur + fadeInDur;

    const t1 = pauseStart / cycle;
    const t2 = (pauseStart + scrollDur) / cycle;
    const t3 = (pauseStart + scrollDur + pauseEnd) / cycle;
    const t4 = (pauseStart + scrollDur + pauseEnd + fadeOutDur) / cycle;

    anim = text.animate(
      [
        { offset: 0, transform: 'translateX(0px)', opacity: 1, easing: 'linear' },
        { offset: t1, transform: 'translateX(0px)', opacity: 1, easing: 'linear' },
        { offset: t2, transform: `translateX(-${shift}px)`, opacity: 1, easing: 'linear' },
        { offset: t3, transform: `translateX(-${shift}px)`, opacity: 1, easing: 'ease' },
        { offset: Math.max(t3 + 0.0001, t4 - 0.0001), transform: `translateX(-${shift}px)`, opacity: 0 },
        { offset: t4, transform: 'translateX(0px)', opacity: 0, easing: 'ease' },
        { offset: 1, transform: 'translateX(0px)', opacity: 1 }
      ],
      {
        duration: cycle * 1000,
        iterations: Infinity
      }
    );
  };

  const schedule = () => {
    stop();
    startOrUpdate();
    rafId = requestAnimationFrame(() => {
      rafId = 0;
      startOrUpdate();
    });
  };

  const ro = new ResizeObserver(schedule);
  ro.observe(wrap);
  const firstChild = wrap.firstElementChild;
  if (firstChild) ro.observe(firstChild);

  schedule();

  return {
    update(newParams = {}) {
      const changed =
        newParams?.text !== params?.text || newParams?.disabled !== params?.disabled;
      params = newParams;
      if (changed) {
        schedule();
      }
    },
    destroy() {
      stop();
      ro.disconnect();
    }
  };
};
