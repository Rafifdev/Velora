import { animate, spring, stagger } from 'motion';

export const SPRING_PANEL = spring(0.32, 0.14);
export const SPRING_CARD = spring(0.28, 0.18);
export const SPRING_SNAPPY = spring(0.22, 0.25);

export function animatePanelWidth(node: HTMLElement, widthPx: number, animateMotion = true) {
  if (!node) return;
  if (!animateMotion) {
    node.style.width = `${widthPx}px`;
    return;
  }
  return animate(
    node,
    { width: `${widthPx}px` },
    {
      duration: 0.34,
      ease: [0.22, 1, 0.36, 1]
    }
  );
}

export function animateCardEntrance(node: HTMLElement, delayIndex = 0) {
  if (!node) return;
  return animate(
    node,
    {
      opacity: [0, 1],
      y: [12, 0],
      scale: [0.98, 1]
    },
    {
      duration: 0.36,
      delay: delayIndex * 0.045,
      ease: [0.22, 1, 0.36, 1]
    }
  );
}

export function animateViewSwap(node: HTMLElement) {
  if (!node) return;
  return animate(
    node,
    {
      opacity: [0, 1],
      y: [8, 0]
    },
    {
      duration: 0.26,
      ease: [0.22, 1, 0.36, 1]
    }
  );
}

export function animateStaggerChildren(container: HTMLElement, selector = '[data-motion-item]') {
  if (!container) return;
  const items = Array.from(container.querySelectorAll(selector));
  if (items.length === 0) return;
  return animate(
    items.slice(0, 24),
    {
      opacity: [0, 1],
      y: [10, 0]
    },
    {
      duration: 0.3,
      delay: stagger(0.025),
      ease: [0.22, 1, 0.36, 1]
    }
  );
}

export function svelteMotionEnter(
  node: HTMLElement,
  params: { delay?: number; y?: number; scale?: number; duration?: number } = {}
) {
  const { delay = 0, y = 6, scale = 1, duration = 0.26 } = params;
  node.style.opacity = '0';
  node.style.willChange = 'transform, opacity';
  const anim = animate(
    node,
    {
      opacity: [0, 1],
      y: [y, 0],
      scale: [scale, 1]
    },
    {
      duration,
      delay,
      ease: [0.22, 1, 0.36, 1]
    }
  );
  void anim.finished.then(() => {
    node.style.opacity = '';
    node.style.willChange = '';
  });
  return {
    destroy() {
      anim.stop();
    }
  };
}
