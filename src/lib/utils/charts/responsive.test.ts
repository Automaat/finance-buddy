import { describe, it, expect } from 'vitest';
import type { EChartsOption } from 'echarts';
import { applyMobileChartTweaks } from './responsive';

type TitleShape = { textStyle: { fontSize: number } };
type PieShape = {
	label: { show: boolean };
	labelLine: { show: boolean };
	avoidLabelOverlap: boolean;
};

describe('applyMobileChartTweaks', () => {
	it('returns the option untouched when not mobile', () => {
		const option: EChartsOption = { title: { text: 'X', textStyle: { fontSize: 16 } } };
		expect(applyMobileChartTweaks(option, false)).toBe(option);
		expect((option.title as TitleShape).textStyle.fontSize).toBe(16);
	});

	it('shrinks an oversized in-canvas title on mobile', () => {
		const option: EChartsOption = { title: { text: 'Długi tytuł', textStyle: { fontSize: 18 } } };
		applyMobileChartTweaks(option, true);
		expect((option.title as TitleShape).textStyle.fontSize).toBe(13);
	});

	it('caps a title with no explicit font size', () => {
		const option: EChartsOption = { title: { text: 'X' } };
		applyMobileChartTweaks(option, true);
		expect((option.title as TitleShape).textStyle.fontSize).toBe(13);
	});

	it('leaves an already-small title font as-is', () => {
		const option: EChartsOption = { title: { text: 'X', textStyle: { fontSize: 11 } } };
		applyMobileChartTweaks(option, true);
		expect((option.title as TitleShape).textStyle.fontSize).toBe(11);
	});

	it('handles an array of titles', () => {
		const option: EChartsOption = { title: [{ text: 'A', textStyle: { fontSize: 20 } }] };
		applyMobileChartTweaks(option, true);
		expect((option.title as TitleShape[])[0].textStyle.fontSize).toBe(13);
	});

	it('drops pie callout labels in favour of the legend on mobile', () => {
		const option: EChartsOption = {
			series: [{ type: 'pie', label: { show: true }, labelLine: { show: true }, data: [] }]
		};
		applyMobileChartTweaks(option, true);
		const series = (option.series as PieShape[])[0];
		expect(series.label.show).toBe(false);
		expect(series.labelLine.show).toBe(false);
		expect(series.avoidLabelOverlap).toBe(true);
	});

	it('handles a single (non-array) pie series', () => {
		const option: EChartsOption = { series: { type: 'pie', label: { show: true }, data: [] } };
		applyMobileChartTweaks(option, true);
		expect((option.series as PieShape).label.show).toBe(false);
	});

	it('leaves non-pie series labels untouched', () => {
		const option: EChartsOption = { series: [{ type: 'bar', label: { show: true }, data: [] }] };
		applyMobileChartTweaks(option, true);
		expect((option.series as { label: { show: boolean } }[])[0].label.show).toBe(true);
	});
});
