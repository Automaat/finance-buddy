const plnFormatter = new Intl.NumberFormat('pl-PL', {
	style: 'currency',
	currency: 'PLN',
	maximumFractionDigits: 0
});

const percentFormatter = new Intl.NumberFormat('pl-PL', {
	style: 'percent',
	minimumFractionDigits: 1,
	maximumFractionDigits: 1
});

const dateFormatter = new Intl.DateTimeFormat('pl-PL', {
	year: 'numeric',
	month: '2-digit',
	day: '2-digit'
});

export function formatPLN(value: number | null | undefined): string {
	if (value == null || Number.isNaN(value)) return '—';
	return plnFormatter.format(value);
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

export interface Change {
	absolute: number;
	percent: number;
	direction: 'up' | 'down' | 'flat';
}

export function calculateChange(current: number, previous: number): Change {
	const absolute = current - previous;
	const percent = previous === 0 ? 0 : (absolute / Math.abs(previous)) * 100;
	const direction: Change['direction'] = absolute > 0 ? 'up' : absolute < 0 ? 'down' : 'flat';
	return { absolute, percent, direction };
}
