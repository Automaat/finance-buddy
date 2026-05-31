import type { EChartsOption } from 'echarts';
import {
	chartInk,
	chartAxisLine,
	chartSplitLine,
	chartTooltipBg,
	chartTooltipBorder
} from '$lib/utils/theme';

// Shared ECharts "chrome" the metryki builders all repeat: a transparent
// background, a centered bold title, and a token-colored tooltip surface.
// Builders spread this and override only what differs (series, axes, legend),
// so the furniture lives in one place instead of being copy-pasted per chart.

/** Tooltip surface styled with the theme tokens. `trigger` defaults to 'axis'. */
export function chartTooltip(
	trigger: 'axis' | 'item' = 'axis'
): NonNullable<EChartsOption['tooltip']> {
	return {
		trigger,
		backgroundColor: chartTooltipBg,
		borderColor: chartTooltipBorder,
		textStyle: { color: chartInk }
	};
}

/** A solid axis line in the theme color. */
export function axisLine(width = 2) {
	return { lineStyle: { color: chartAxisLine, width } };
}

/** A dashed split (grid) line in the theme color. */
export function splitLine() {
	return { lineStyle: { color: chartSplitLine, type: 'dashed' as const } };
}

/**
 * Base option shared by every metryki chart: transparent background plus a
 * token-styled centered title and tooltip. Spread it first, then override.
 */
export function baseChartOption(title: string, isMobile = false): EChartsOption {
	return {
		backgroundColor: 'transparent',
		title: {
			text: title,
			left: 'center',
			top: 10,
			textStyle: { color: chartInk, fontSize: isMobile ? 13 : 16, fontWeight: 'bold' }
		},
		tooltip: chartTooltip()
	};
}
