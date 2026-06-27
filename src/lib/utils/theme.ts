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

// --- Chart "chrome" tokens ---------------------------------------------------
// The non-data furniture every chart shares: text, axes, grid lines, tooltip
// surface. Pulled into one place so builders stop hardcoding a disconnected
// Nord palette and instead read tokens aligned with the rose app theme.

/** Primary text/ink for titles, axis names, labels. */
export const chartInk = '#3f3f46';

/** Muted ink for secondary axis labels and de-emphasized text. */
export const chartInkMuted = '#71717a';

/** Axis line color. */
export const chartAxisLine = '#d4d4d8';

/** Dashed split (grid) line color. */
export const chartSplitLine = '#e4e4e7';

/** Tooltip surface + border. */
export const chartTooltipBg = 'rgba(255, 255, 255, 0.95)';
export const chartTooltipBorder = '#d4d4d8';

// --- Semantic series colors --------------------------------------------------
// Stable meanings used across the investment charts so the same concept reads
// the same color everywhere.

/** Contributions / deposits (area fill under the value line). */
export const chartContribution = '#fb7185';

/** Portfolio value / primary line. */
export const chartValue = '#e11d48';

/** Positive return / gains. */
export const chartPositive = '#16a34a';

/** Negative values / losses — unambiguously red, reserved for bad outcomes only. */
export const chartNegative = '#dc2626';

/** Area-fill gradient stops for chartPositive, top (opaque) to bottom (faint). */
export const chartPositiveGradient = ['rgba(22, 163, 74, 0.5)', 'rgba(22, 163, 74, 0.1)'] as const;
