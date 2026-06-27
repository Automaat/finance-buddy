const percentFormatter = new Intl.NumberFormat('pl-PL', {
	style: 'percent',
	minimumFractionDigits: 1,
	maximumFractionDigits: 1
});

const numberFormatterCache = new Map<number, Intl.NumberFormat>();

const dateFormatter = new Intl.DateTimeFormat('pl-PL', {
	year: 'numeric',
	month: '2-digit',
	day: '2-digit'
});

export interface Change {
	absolute: number;
	percent: number;
	direction: 'up' | 'down' | 'flat';
}

// pl-PL CLDR sets minimumGroupingDigits=2, so Intl only groups at 10 000+.
// Disable built-in grouping and insert U+202F manually from 1 000 upward.
export function formatNumber(value: number | null | undefined, decimals = 2): string {
	if (value == null || Number.isNaN(value)) return '—';
	const isNeg = value < 0;
	let nf = numberFormatterCache.get(decimals);
	if (!nf) {
		nf = new Intl.NumberFormat('pl-PL', {
			minimumFractionDigits: decimals,
			maximumFractionDigits: decimals,
			useGrouping: false
		});
		numberFormatterCache.set(decimals, nf);
	}
	const formatted = nf.format(Math.abs(value));
	const [intPart, fracPart] = formatted.split(',');
	const grouped = intPart.replace(/\B(?=(\d{3})+(?!\d))/g, ' ');
	const result = fracPart !== undefined ? `${grouped},${fracPart}` : grouped;
	return isNeg ? `−${result}` : result;
}

export function formatPLN(value: number | null | undefined): string {
	if (value == null || Number.isNaN(value)) return '—';
	return `${formatNumber(value, 0)} zł`;
}

export function formatPercent(value: number | null | undefined): string {
	if (value == null || Number.isNaN(value)) return '—';
	return percentFormatter.format(value / 100);
}

export function formatDate(value: string | Date | null | undefined): string {
	if (!value) return '—';
	const date = value instanceof Date ? value : new Date(value);
	if (Number.isNaN(date.getTime())) return '—';
	return dateFormatter.format(date);
}

export function formatSignedPLN(value: number | null | undefined): string {
	if (value == null || Number.isNaN(value)) return '—';
	if (value === 0) return `${formatNumber(0, 0)} zł`;
	const sign = value > 0 ? '+' : '−';
	return `${sign}${formatNumber(Math.abs(value), 0)} zł`;
}

export function formatSignedPercent(value: number | null | undefined): string {
	if (value == null || Number.isNaN(value)) return '—';
	if (value === 0) return percentFormatter.format(0);
	const sign = value > 0 ? '+' : '−';
	return `${sign}${percentFormatter.format(Math.abs(value) / 100)}`;
}

export function calculateChange(current: number, previous: number): Change {
	const absolute = current - previous;
	const percent = previous === 0 ? 0 : (absolute / Math.abs(previous)) * 100;
	const direction: Change['direction'] = absolute > 0 ? 'up' : absolute < 0 ? 'down' : 'flat';
	return { absolute, percent, direction };
}
