import type { BarSeriesOption, EChartsOption } from 'echarts';
import type { CallbackDataParams, TopLevelFormatterParams } from 'echarts/types/dist/shared';

type AxisTooltipItem = CallbackDataParams & { axisValue: string };

// ECharts supports a callback for bar `label.position` at runtime, but the
// shipped type definition omits the function form. This narrows the label
// type to add it without resorting to `any`.
type RoiLabel = Omit<NonNullable<BarSeriesOption['label']>, 'position'> & {
	position: (params: { value: unknown }) => 'top' | 'bottom';
};

type RoiBarSeries = Omit<BarSeriesOption, 'label'> & { label: RoiLabel };

export interface AllocationCategory {
	category: string;
	current_percentage: number;
	target_percentage: number;
}

export interface AllocationWrapper {
	wrapper: string;
	value: number;
}

export interface TimeSeriesPoint {
	date: string;
	value?: number;
	total_value?: number;
	contributions?: number;
	cumulative_contributions?: number;
}

const formatPLN = (val: number): string =>
	val.toLocaleString('pl-PL', { minimumFractionDigits: 0, maximumFractionDigits: 0 });

export function buildAllocationChartOption(
	byCategory: AllocationCategory[],
	isMobile = false
): EChartsOption {
	const categories = byCategory.map((item) => item.category);
	const currentValues = byCategory.map((item) => parseFloat(item.current_percentage.toFixed(1)));
	const targetValues = byCategory.map((item) => parseFloat(item.target_percentage.toFixed(1)));

	return {
		backgroundColor: 'transparent',
		title: {
			text: 'Alokacja inwestycyjna: Obecna vs Docelowa',
			left: 'center',
			top: 10,
			textStyle: { color: '#2e3440', fontSize: isMobile ? 13 : 16, fontWeight: 'bold' }
		},
		tooltip: {
			trigger: 'axis',
			axisPointer: { type: 'shadow' },
			backgroundColor: 'rgba(255, 255, 255, 0.95)',
			borderColor: '#d8dee9',
			textStyle: { color: '#2e3440' },
			formatter: '{b}<br/>{a0}: {c0}%<br/>{a1}: {c1}%'
		},
		legend: {
			data: ['Obecna', 'Docelowa'],
			bottom: 10,
			textStyle: { color: '#2e3440', fontSize: isMobile ? 11 : 14 }
		},
		grid: {
			left: 60,
			right: 40,
			bottom: 60,
			top: 80,
			containLabel: false
		},
		xAxis: {
			type: 'category',
			data: categories,
			axisLabel: {
				color: '#2e3440',
				fontSize: isMobile ? 12 : 14,
				fontWeight: 'bold',
				formatter: function (value: string) {
					return value.charAt(0).toUpperCase() + value.slice(1);
				}
			},
			axisLine: { lineStyle: { color: '#d8dee9', width: 2 } },
			axisTick: { show: false }
		},
		yAxis: {
			type: 'value',
			name: 'Procent (%)',
			nameLocation: 'middle',
			nameGap: 50,
			nameTextStyle: { color: '#2e3440', fontSize: 14, fontWeight: 'bold' },
			max: 100,
			axisLabel: {
				color: '#2e3440',
				fontSize: 13,
				fontWeight: 'bold',
				formatter: '{value}%'
			},
			axisLine: { lineStyle: { color: '#d8dee9', width: 2 } },
			splitLine: { lineStyle: { color: '#e5e9f0', type: 'dashed' } }
		},
		series: [
			{
				name: 'Obecna',
				type: 'bar',
				data: currentValues,
				barWidth: '35%',
				itemStyle: { color: '#88c0d0', borderRadius: [4, 4, 0, 0] },
				label: {
					show: true,
					position: 'top',
					distance: 5,
					formatter: '{c}%',
					color: '#2e3440',
					fontSize: isMobile ? 10 : 14,
					fontWeight: 'bold'
				}
			},
			{
				name: 'Docelowa',
				type: 'bar',
				data: targetValues,
				barWidth: '35%',
				itemStyle: { color: '#81a1c1', borderRadius: [4, 4, 0, 0] },
				label: {
					show: true,
					position: 'top',
					distance: 5,
					formatter: '{c}%',
					color: '#2e3440',
					fontSize: isMobile ? 10 : 14,
					fontWeight: 'bold'
				}
			}
		]
	};
}

export function buildWrapperChartOption(
	byWrapper: AllocationWrapper[],
	isMobile = false
): EChartsOption {
	const wrapperData = byWrapper.map((item) => ({
		name: item.wrapper,
		value: item.value
	}));

	return {
		backgroundColor: 'transparent',
		title: {
			text: 'Podział według kont (IKE/IKZE/PPK)',
			left: 'center',
			top: 10,
			textStyle: { color: '#2e3440', fontSize: isMobile ? 13 : 16, fontWeight: 'bold' }
		},
		tooltip: {
			trigger: 'item',
			backgroundColor: 'rgba(255, 255, 255, 0.95)',
			borderColor: '#d8dee9',
			textStyle: { color: '#2e3440' },
			formatter: function (params) {
				const single = Array.isArray(params) ? params[0] : params;
				const value = (single.value as number).toLocaleString('pl-PL', {
					minimumFractionDigits: 0,
					maximumFractionDigits: 0
				});
				return `${single.name}: ${value} PLN (${(single.percent ?? 0).toFixed(1)}%)`;
			}
		},
		legend: isMobile
			? {
					type: 'scroll',
					bottom: 0,
					left: 'center',
					textStyle: { color: '#2e3440', fontSize: 11, fontWeight: 'bold' }
				}
			: {
					orient: 'vertical',
					left: 20,
					top: 'middle',
					textStyle: { color: '#2e3440', fontSize: 14, fontWeight: 'bold' },
					itemGap: 15
				},
		series: [
			{
				type: 'pie',
				radius: ['40%', '65%'],
				center: isMobile ? ['50%', '42%'] : ['50%', '50%'],
				avoidLabelOverlap: true,
				minShowLabelAngle: 1,
				data: wrapperData,
				color: ['#88c0d0', '#81a1c1', '#5e81ac', '#b48ead'],
				emphasis: {
					itemStyle: {
						shadowBlur: 10,
						shadowOffsetX: 0,
						shadowColor: 'rgba(0, 0, 0, 0.5)'
					},
					label: {
						show: true,
						fontSize: 18,
						fontWeight: 'bold'
					}
				},
				label: {
					show: !isMobile,
					position: 'outside',
					alignTo: 'edge',
					margin: 20,
					edgeDistance: '15%',
					color: '#000000',
					fontSize: 15,
					fontWeight: 'bold',
					formatter: function (params) {
						return `${params.name}\n${(params.percent ?? 0).toFixed(1)}%`;
					},
					overflow: 'none'
				},
				labelLine: {
					show: !isMobile,
					length: 25,
					length2: 20,
					smooth: 0.2,
					lineStyle: {
						color: '#2e3440',
						width: 2
					}
				},
				labelLayout: {
					hideOverlap: false,
					moveOverlap: 'shiftY'
				}
			}
		]
	};
}

export function buildInvestmentTrendChartOption(series: TimeSeriesPoint[]): EChartsOption {
	const dates = series.map((item) => item.date);
	const values = series.map((item) => item.value ?? 0);
	const contributions = series.map((item) => item.contributions ?? 0);

	return {
		backgroundColor: 'transparent',
		title: {
			text: 'Inwestycje w czasie',
			left: 'center',
			top: 10,
			textStyle: { color: '#2e3440', fontSize: 16, fontWeight: 'bold' }
		},
		tooltip: {
			trigger: 'axis',
			backgroundColor: 'rgba(255, 255, 255, 0.95)',
			borderColor: '#d8dee9',
			textStyle: { color: '#2e3440' },
			formatter: function (params: TopLevelFormatterParams) {
				const items = (Array.isArray(params) ? params : [params]) as AxisTooltipItem[];
				const date = items[0].axisValue;
				const contributed = items[0].value as number;
				const value = items[1].value as number;
				const returns = value - contributed;

				return `${date}<br/>
					<span style="color:#5e81ac">●</span> Wartość portfela: <b>${formatPLN(value)} PLN</b><br/>
					<span style="color:#88c0d0">■</span> Wpłaty: ${formatPLN(contributed)} PLN<br/>
					<span style="color:#a3be8c">■</span> Zyski: ${formatPLN(returns)} PLN`;
			}
		},
		legend: {
			data: ['Wpłaty', 'Wartość portfela'],
			bottom: 10,
			textStyle: { color: '#2e3440', fontSize: 14 }
		},
		grid: {
			left: 80,
			right: 40,
			bottom: 80,
			top: 80,
			containLabel: false
		},
		xAxis: {
			type: 'category',
			data: dates,
			axisLabel: {
				color: '#2e3440',
				fontSize: 12,
				rotate: 45
			},
			axisLine: { lineStyle: { color: '#d8dee9', width: 2 } },
			boundaryGap: false
		},
		yAxis: {
			type: 'value',
			name: 'Wartość (PLN)',
			nameLocation: 'middle',
			nameGap: 60,
			nameTextStyle: { color: '#2e3440', fontSize: 14, fontWeight: 'bold' },
			axisLabel: {
				color: '#2e3440',
				fontSize: 12,
				formatter: function (value: number) {
					return (value / 1000).toFixed(0) + 'k';
				}
			},
			axisLine: { lineStyle: { color: '#d8dee9', width: 2 } },
			splitLine: { lineStyle: { color: '#e5e9f0', type: 'dashed' } }
		},
		series: [
			{
				name: 'Wpłaty',
				type: 'line',
				data: contributions,
				smooth: true,
				lineStyle: { width: 0 },
				showSymbol: false,
				areaStyle: {
					color: '#88c0d0',
					opacity: 0.8
				},
				emphasis: {
					focus: 'series'
				}
			},
			{
				name: 'Wartość portfela',
				type: 'line',
				data: values,
				smooth: true,
				lineStyle: { width: 3, color: '#5e81ac' },
				showSymbol: false,
				emphasis: {
					focus: 'series'
				}
			}
		]
	};
}

export function buildWrapperTrendChartOption(
	title: string,
	series: TimeSeriesPoint[]
): EChartsOption {
	const dates = series.map((item) => item.date);
	const contributions = series.map((item) => item.contributions ?? 0);
	const values = series.map((item) => item.value ?? 0);

	return {
		backgroundColor: 'transparent',
		title: {
			text: title,
			left: 'center',
			top: 10,
			textStyle: { color: '#2e3440', fontSize: 16, fontWeight: 'bold' }
		},
		tooltip: {
			trigger: 'axis',
			backgroundColor: 'rgba(255, 255, 255, 0.95)',
			borderColor: '#d8dee9',
			textStyle: { color: '#2e3440' },
			formatter: function (params: TopLevelFormatterParams) {
				const items = (Array.isArray(params) ? params : [params]) as AxisTooltipItem[];
				const date = items[0].axisValue;
				const contributed = items[0].value as number;
				const value = items[1].value as number;
				const returns = value - contributed;

				return `${date}<br/>
					<span style="color:#5e81ac">●</span> Wartość: <b>${formatPLN(value)} PLN</b><br/>
					<span style="color:#88c0d0">■</span> Wpłaty: ${formatPLN(contributed)} PLN<br/>
					<span style="color:#a3be8c">■</span> Zyski: ${formatPLN(returns)} PLN`;
			}
		},
		legend: {
			data: ['Wpłaty', 'Wartość portfela'],
			bottom: 10,
			textStyle: { color: '#2e3440', fontSize: 14 }
		},
		grid: {
			left: 80,
			right: 40,
			bottom: 80,
			top: 80,
			containLabel: false
		},
		xAxis: {
			type: 'category',
			data: dates,
			axisLabel: {
				color: '#2e3440',
				fontSize: 11,
				rotate: 45
			},
			axisLine: { lineStyle: { color: '#d8dee9', width: 2 } },
			boundaryGap: false
		},
		yAxis: {
			type: 'value',
			name: 'Wartość (PLN)',
			nameLocation: 'middle',
			nameGap: 60,
			nameTextStyle: { color: '#2e3440', fontSize: 14, fontWeight: 'bold' },
			axisLabel: {
				color: '#2e3440',
				fontSize: 11,
				formatter: function (value: number) {
					return (value / 1000).toFixed(0) + 'k';
				}
			},
			axisLine: { lineStyle: { color: '#d8dee9', width: 2 } },
			splitLine: { lineStyle: { color: '#e5e9f0', type: 'dashed' } }
		},
		series: [
			{
				name: 'Wpłaty',
				type: 'line',
				data: contributions,
				smooth: true,
				lineStyle: { width: 0 },
				showSymbol: false,
				areaStyle: {
					color: '#88c0d0',
					opacity: 0.8
				},
				emphasis: {
					focus: 'series'
				}
			},
			{
				name: 'Wartość portfela',
				type: 'line',
				data: values,
				smooth: true,
				lineStyle: { width: 3, color: '#5e81ac' },
				showSymbol: false,
				emphasis: {
					focus: 'series'
				}
			}
		]
	};
}

// Computes year-over-year ROI using the modified Dietz method: the
// denominator weights mid-period contributions to approximate average
// invested capital. The first year is skipped when it begins with a
// zero-value placeholder entry.
export function computeYearlyRoi(series: TimeSeriesPoint[]): Map<number, number> {
	type Pt = { date: string; value: number; contribs: number };
	const byYear = new Map<number, { first: Pt; last: Pt }>();
	for (const point of series) {
		const year = new Date(point.date).getFullYear();
		const val = point.value ?? point.total_value ?? 0;
		const contrib = point.contributions ?? point.cumulative_contributions ?? 0;
		const pt: Pt = { date: point.date, value: val, contribs: contrib };
		const existing = byYear.get(year);
		if (!existing) {
			byYear.set(year, { first: pt, last: pt });
		} else {
			if (point.date < existing.first.date) existing.first = pt;
			if (point.date > existing.last.date) existing.last = pt;
		}
	}
	const years = [...byYear.keys()].sort((a, b) => a - b);
	const result = new Map<number, number>();
	for (let i = 0; i < years.length; i++) {
		const { first, last: end } = byYear.get(years[i])!;
		let start: Pt;
		if (i === 0) {
			if (first.value === 0) continue;
			start = first;
		} else {
			start = byYear.get(years[i - 1])!.last;
		}
		const yearContribs = end.contribs - start.contribs;
		const denom = start.value + yearContribs / 2;
		const roi = denom > 0 ? ((end.value - start.value - yearContribs) / denom) * 100 : 0;
		result.set(years[i], parseFloat(roi.toFixed(2)));
	}
	return result;
}

export function buildYearlyRoiChartOption(
	stockSeries: TimeSeriesPoint[],
	bondSeries: TimeSeriesPoint[],
	ppkSeries: TimeSeriesPoint[]
): EChartsOption {
	const stockRoi = computeYearlyRoi(stockSeries);
	const bondRoi = computeYearlyRoi(bondSeries);
	const ppkRoi = computeYearlyRoi(ppkSeries);
	const allYears = [...new Set([...stockRoi.keys(), ...bondRoi.keys(), ...ppkRoi.keys()])].sort(
		(a, b) => a - b
	);

	const labelPosition = (params: { value: unknown }): 'top' | 'bottom' =>
		Number(params.value) >= 0 ? 'top' : 'bottom';

	const roiSeries: RoiBarSeries[] = [
		{
			name: 'Akcje',
			type: 'bar',
			barWidth: '25%',
			data: allYears.map((y) => stockRoi.get(y) ?? null),
			itemStyle: { color: '#a3be8c' },
			label: {
				show: true,
				position: labelPosition,
				formatter: '{c}%',
				color: '#2e3440',
				fontSize: 11
			},
			markLine: {
				silent: true,
				symbol: 'none',
				data: [{ yAxis: 0 }],
				lineStyle: { color: '#4c566a', type: 'solid', width: 1 }
			}
		},
		{
			name: 'Obligacje',
			type: 'bar',
			barWidth: '25%',
			data: allYears.map((y) => bondRoi.get(y) ?? null),
			itemStyle: { color: '#d08770' },
			label: {
				show: true,
				position: labelPosition,
				formatter: '{c}%',
				color: '#2e3440',
				fontSize: 11
			}
		},
		{
			name: 'PPK',
			type: 'bar',
			barWidth: '25%',
			data: allYears.map((y) => ppkRoi.get(y) ?? null),
			itemStyle: { color: '#88c0d0' },
			label: {
				show: true,
				position: labelPosition,
				formatter: '{c}%',
				color: '#2e3440',
				fontSize: 11
			}
		}
	];

	return {
		backgroundColor: 'transparent',
		title: {
			text: 'Roczny ROI: Akcje, Obligacje, PPK',
			left: 'center',
			top: 10,
			textStyle: { color: '#2e3440', fontSize: 16, fontWeight: 'bold' }
		},
		tooltip: {
			trigger: 'axis',
			axisPointer: { type: 'shadow' },
			backgroundColor: 'rgba(255, 255, 255, 0.95)',
			borderColor: '#d8dee9',
			textStyle: { color: '#2e3440' },
			formatter: function (params: TopLevelFormatterParams) {
				const items = (Array.isArray(params) ? params : [params]) as CallbackDataParams[];
				return items
					.map((p) => `${p.seriesName}: ${p.value != null ? p.value + '%' : 'brak danych'}`)
					.join('<br/>');
			}
		},
		legend: {
			top: 40,
			textStyle: { color: '#2e3440' }
		},
		grid: { top: 80, left: 60, right: 30, bottom: 60 },
		xAxis: {
			type: 'category',
			data: allYears.map(String),
			axisLabel: { color: '#4c566a' }
		},
		yAxis: {
			type: 'value',
			axisLabel: { formatter: '{value}%', color: '#4c566a' },
			splitLine: { lineStyle: { color: '#e5e9f0' } }
		},
		series: roiSeries as unknown as BarSeriesOption[]
	};
}
