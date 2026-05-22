// Shared ECharts color tokens. Centralized so chart components don't carry
// their own hardcoded palettes. Rose/crimson scale matching the app theme.

/** Categorical palette for multi-series charts (e.g. allocation pie). */
export const chartPalette = [
	'#e11d48',
	'#f43f5e',
	'#fb7185',
	'#fda4af',
	'#881337',
	'#9f1239',
	'#be123c',
	'#be185d'
] as const;

/** Accent color for single-series charts (e.g. net worth line). */
export const chartAccent = '#e11d48';

/** Area-fill gradient stops for the accent color, top (opaque) to bottom (faint). */
export const chartAccentGradient = ['rgba(225, 29, 72, 0.5)', 'rgba(225, 29, 72, 0.1)'] as const;
