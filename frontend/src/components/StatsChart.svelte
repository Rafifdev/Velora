<script lang="ts">
  import { Chart, registerables } from 'chart.js';

  Chart.register(...registerables);

  interface Props {
    type: 'line' | 'bar';
    labels: string[];
    values: number[];
    secondaryValues?: number[];
    unitLabel: string;
    secondaryUnitLabel?: string;
    highlightIndex?: number;
  }

  let {
    type,
    labels,
    values,
    secondaryValues = [],
    unitLabel,
    secondaryUnitLabel = '',
    highlightIndex = -1
  }: Props = $props();

  let canvasEl = $state<HTMLCanvasElement | undefined>();

  function colorToRgba(rawColor: string, alpha: number, fallbackHex: string): string {
    const col = (rawColor || fallbackHex).trim();
    if (typeof document === 'undefined') return fallbackHex;
    const probe = document.createElement('canvas');
    probe.width = 1;
    probe.height = 1;
    const ctx = probe.getContext('2d');
    if (!ctx) return fallbackHex;
    ctx.clearRect(0, 0, 1, 1);
    ctx.fillStyle = fallbackHex;
    ctx.fillStyle = col;
    ctx.fillRect(0, 0, 1, 1);
    const data = ctx.getImageData(0, 0, 1, 1).data;
    return `rgba(${data[0]}, ${data[1]}, ${data[2]}, ${alpha})`;
  }

  $effect(() => {
    const el = canvasEl;
    if (!el) return;
    const ctx = el.getContext('2d');
    if (!ctx) return;

    const curLabels = [...labels];
    const curValues = [...values];
    const curSecondary = [...secondaryValues];
    const curType = type;
    const curHighlight = highlightIndex;
    const curUnit = unitLabel;
    const curSecUnit = secondaryUnitLabel;

    const styles = getComputedStyle(el);
    const rawPrimary = styles.getPropertyValue('--md-sys-color-primary').trim() || '#F6847E';
    const rawTertiary = styles.getPropertyValue('--md-sys-color-tertiary').trim() || '#A7C8FF';
    const rawTextVar = styles.getPropertyValue('--md-sys-color-on-surface-variant').trim() || '#9CA3AF';
    const rawOutline = styles.getPropertyValue('--md-sys-color-outline-variant').trim() || '#4B5563';
    const rawInvSurf = styles.getPropertyValue('--md-sys-color-inverse-surface').trim() || '#1F2937';
    const rawInvText = styles.getPropertyValue('--md-sys-color-inverse-on-surface').trim() || '#F9FAFB';

    const primarySolid = colorToRgba(rawPrimary, 1, '#F6847E');
    const tertiarySolid = colorToRgba(rawTertiary, 1, '#A7C8FF');
    const tickColor = colorToRgba(rawTextVar, 0.78, '#9CA3AF');
    const gridColor = colorToRgba(rawOutline, 0.24, '#4B5563');
    const mutedBar = colorToRgba(rawOutline, 0.28, '#4B5563');
    const tooltipBg = colorToRgba(rawInvSurf, 0.96, '#1F2937');
    const tooltipFg = colorToRgba(rawInvText, 1, '#F9FAFB');

    const areaGrad = ctx.createLinearGradient(0, 0, 0, el.clientHeight || 168);
    areaGrad.addColorStop(0, colorToRgba(rawPrimary, 0.34, '#F6847E'));
    areaGrad.addColorStop(1, colorToRgba(rawPrimary, 0.0, '#F6847E'));

    const pointRadii = curValues.map((v, i) => {
      if (v <= 0) return 0;
      return i === curHighlight ? 5 : 3;
    });
    const pointColors = curValues.map((_, i) =>
      i === curHighlight ? tertiarySolid : primarySolid
    );

    const barColors = curValues.map((v, i) => {
      if (v <= 0) return mutedBar;
      return i === curHighlight ? tertiarySolid : primarySolid;
    });

    const chart = new Chart(ctx, {
      type: curType,
      data: {
        labels: curLabels,
        datasets: [
          curType === 'line'
            ? {
                data: curValues,
                borderColor: primarySolid,
                backgroundColor: areaGrad,
                fill: true,
                tension: 0.38,
                borderWidth: 2.5,
                pointRadius: pointRadii,
                pointHoverRadius: 6,
                pointBackgroundColor: pointColors,
                pointBorderColor: pointColors
              }
            : {
                data: curValues,
                backgroundColor: barColors,
                borderRadius: 8,
                borderSkipped: false,
                barPercentage: 0.56,
                categoryPercentage: 0.82
              }
        ]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        animation: { duration: 420 },
        interaction: {
          mode: 'index',
          intersect: false
        },
        plugins: {
          legend: { display: false },
          tooltip: {
            backgroundColor: tooltipBg,
            titleColor: tooltipFg,
            bodyColor: tooltipFg,
            padding: 10,
            cornerRadius: 10,
            displayColors: false,
            callbacks: {
              label: (item) => {
                const v = Number(item.raw ?? 0);
                const sec = curSecondary[item.dataIndex];
                if (sec !== undefined && curSecUnit) {
                  return `${v} ${curUnit} • ${sec} ${curSecUnit}`;
                }
                return `${v} ${curUnit}`;
              }
            }
          }
        },
        scales: {
          x: {
            grid: { display: false },
            border: { display: false },
            ticks: {
              color: tickColor,
              font: { family: 'Poppins, sans-serif', size: 10.5, weight: 500 },
              maxRotation: 0,
              autoSkip: curType === 'line',
              maxTicksLimit: curType === 'line' ? 7 : 7
            }
          },
          y: {
            beginAtZero: true,
            suggestedMax: 5,
            border: { display: false, dash: [4, 4] },
            grid: { color: gridColor },
            ticks: {
              color: tickColor,
              font: { family: 'Poppins, sans-serif', size: 10, weight: 500 },
              maxTicksLimit: 4,
              precision: 0
            }
          }
        }
      }
    });

    return () => {
      chart.destroy();
    };
  });
</script>

<div class="relative w-full h-44">
  <canvas bind:this={canvasEl}></canvas>
</div>
