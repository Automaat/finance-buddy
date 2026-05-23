// Preset chips for the dashboard date-range picker. Order matters — chips
// render in this sequence.
export const RANGE_PRESETS = ['1m', '3m', '6m', '1y', '3y', '5y', 'all'] as const;

export type RangePreset = (typeof RANGE_PRESETS)[number];
export type RangeValue = RangePreset | 'custom';

export const PRESET_LABEL: Record<RangePreset, string> = {
	'1m': '1M',
	'3m': '3M',
	'6m': '6M',
	'1y': '1R',
	'3y': '3L',
	'5y': '5L',
	all: 'Wszystko'
};

export const RANGE_LABEL: Record<RangeValue, string> = {
	...PRESET_LABEL,
	custom: 'Własny'
};

export const DEFAULT_RANGE: RangePreset = 'all';

export function isRangePreset(value: string | null | undefined): value is RangePreset {
	return !!value && (RANGE_PRESETS as readonly string[]).includes(value);
}

export function isRangeValue(value: string | null | undefined): value is RangeValue {
	return isRangePreset(value) || value === 'custom';
}

// computePresetBounds returns the [from, to] dates that a preset chip resolves
// to. `to` is always today; `from` is today minus the preset's window.
// `all` returns nulls so the backend sees no bounds.
export function computePresetBounds(
	preset: RangePreset,
	now: Date = new Date()
): { from: string | null; to: string | null } {
	if (preset === 'all') return { from: null, to: null };
	const to = toISODate(now);
	const from = new Date(now);
	switch (preset) {
		case '1m':
			from.setUTCMonth(from.getUTCMonth() - 1);
			break;
		case '3m':
			from.setUTCMonth(from.getUTCMonth() - 3);
			break;
		case '6m':
			from.setUTCMonth(from.getUTCMonth() - 6);
			break;
		case '1y':
			from.setUTCFullYear(from.getUTCFullYear() - 1);
			break;
		case '3y':
			from.setUTCFullYear(from.getUTCFullYear() - 3);
			break;
		case '5y':
			from.setUTCFullYear(from.getUTCFullYear() - 5);
			break;
	}
	return { from: toISODate(from), to };
}

// resolveRangeParams returns the date_from/date_to a load() function should
// forward to the backend, given the current URL search params. A `custom`
// range honors user-supplied date_from/date_to; a preset overrides them.
export function resolveRangeParams(
	searchParams: URLSearchParams,
	now: Date = new Date()
): { range: RangeValue; dateFrom: string | null; dateTo: string | null } {
	const raw = searchParams.get('range');
	const explicitFrom = searchParams.get('date_from');
	const explicitTo = searchParams.get('date_to');

	if (raw === 'custom') {
		return { range: 'custom', dateFrom: explicitFrom, dateTo: explicitTo };
	}
	if (isRangePreset(raw)) {
		const { from, to } = computePresetBounds(raw, now);
		return { range: raw, dateFrom: from, dateTo: to };
	}
	// No `range=` — fall back to either explicit bounds (then it's custom) or
	// the default preset (currently `all`, i.e. no filter).
	if (explicitFrom || explicitTo) {
		return { range: 'custom', dateFrom: explicitFrom, dateTo: explicitTo };
	}
	return { range: DEFAULT_RANGE, dateFrom: null, dateTo: null };
}

function toISODate(d: Date): string {
	// Format in UTC so the result doesn't shift across CI/dev timezones.
	const y = d.getUTCFullYear();
	const m = String(d.getUTCMonth() + 1).padStart(2, '0');
	const day = String(d.getUTCDate()).padStart(2, '0');
	return `${y}-${m}-${day}`;
}
