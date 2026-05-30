import type { EChartsOption } from 'echarts';

// Desktop-tuned chart options clip their in-canvas titles and let pie callout
// labels collide on a phone-width canvas. applyMobileChartTweaks mutates the
// option in place (builders hand back a fresh object on every call) so that on
// a phone the title shrinks and donut/pie series drop their radial callouts in
// favour of the legend. A no-op when isMobile is false.
export function applyMobileChartTweaks(option: EChartsOption, isMobile: boolean): EChartsOption {
	if (!isMobile) return option;

	const titles = Array.isArray(option.title) ? option.title : option.title ? [option.title] : [];
	for (const title of titles) {
		const current = typeof title.textStyle?.fontSize === 'number' ? title.textStyle.fontSize : 16;
		title.textStyle = { ...title.textStyle, fontSize: Math.min(current, 13) };
	}

	const series = Array.isArray(option.series)
		? option.series
		: option.series
			? [option.series]
			: [];
	for (const item of series) {
		if (item.type === 'pie') {
			item.avoidLabelOverlap = true;
			item.label = { ...item.label, show: false };
			item.labelLine = { ...item.labelLine, show: false };
		}
	}

	return option;
}
