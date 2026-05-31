import { describe, it, expect } from 'vitest';
import { baseChartOption, chartTooltip, axisLine, splitLine } from './chartBase';
import { chartInk, chartAxisLine, chartSplitLine, chartTooltipBg } from '$lib/utils/theme';

describe('chartBase', () => {
	it('baseChartOption sets a transparent background, title and tooltip', () => {
		const option = baseChartOption('Tytuł');
		expect(option.backgroundColor).toBe('transparent');
		expect((option.title as { text: string }).text).toBe('Tytuł');
		expect(option.tooltip).toBeTruthy();
	});

	it('baseChartOption uses the theme ink and a smaller title on mobile', () => {
		const desktop = baseChartOption('A').title as {
			textStyle: { color: string; fontSize: number };
		};
		const mobile = baseChartOption('A', true).title as { textStyle: { fontSize: number } };
		expect(desktop.textStyle.color).toBe(chartInk);
		expect(mobile.textStyle.fontSize).toBeLessThan(desktop.textStyle.fontSize);
	});

	it('chartTooltip defaults to axis trigger and the theme surface', () => {
		const tooltip = chartTooltip() as { trigger: string; backgroundColor: string };
		expect(tooltip.trigger).toBe('axis');
		expect(tooltip.backgroundColor).toBe(chartTooltipBg);
		expect((chartTooltip('item') as { trigger: string }).trigger).toBe('item');
	});

	it('axisLine and splitLine carry the theme colors', () => {
		expect(axisLine().lineStyle.color).toBe(chartAxisLine);
		expect(axisLine(5).lineStyle.width).toBe(5);
		expect(splitLine().lineStyle.color).toBe(chartSplitLine);
		expect(splitLine().lineStyle.type).toBe('dashed');
	});
});
