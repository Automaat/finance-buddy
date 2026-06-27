// Shared ECharts color tokens. Centralized so chart components don't carry
// their own hardcoded palettes.

/** Categorical palette for multi-series charts (e.g. allocation pie, retirement by wrapper).
 *  Multi-hue so each category reads as visually distinct. Red/rose are excluded —
 *  those are reserved for chartNegative / chartAccent. */
export const chartPalette = [
	'#3b82f6', // blue-500
	'#10b981', // emerald-500
	'#f59e0b', // amber-500
	'#a855f7', // purple-500
	'#06b6d4', // cyan-500
	'#84cc16', // lime-500
	'#ec4899', // pink-500
	'#f97316' // orange-500
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

/** Contributions / deposits (area fill under the value line). Amber — not red/rose. */
export const chartContribution = '#f59e0b';

/** Portfolio value / primary line. Same hue as chartPalette[0] — blue, not red. */
export const chartValue = '#3b82f6';

/** Area-fill gradient stops for chartValue, top (opaque) to bottom (faint). */
export const chartValueGradient = ['rgba(59, 130, 246, 0.5)', 'rgba(59, 130, 246, 0.1)'] as const;

/** Positive return / gains. */
export const chartPositive = '#16a34a';

/** Negative values / losses — unambiguously red, reserved for bad outcomes only. */
export const chartNegative = '#dc2626';

/** Area-fill gradient stops for chartPositive, top (opaque) to bottom (faint). */
export const chartPositiveGradient = ['rgba(22, 163, 74, 0.5)', 'rgba(22, 163, 74, 0.1)'] as const;
