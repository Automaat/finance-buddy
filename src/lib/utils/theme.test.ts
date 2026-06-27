import { describe, it, expect } from 'vitest';
import {
	chartPalette,
	chartAccent,
	chartAccentGradient,
	chartInk,
	chartInkMuted,
	chartAxisLine,
	chartSplitLine,
	chartContribution,
	chartValue,
	chartPositive
} from './theme';

describe('theme', () => {
	it('exposes 8 distinct palette colors', () => {
		expect(chartPalette).toHaveLength(8);
		expect(new Set(chartPalette).size).toBe(8);
	});

	it('every palette entry is a hex color', () => {
		for (const color of chartPalette) {
			expect(color).toMatch(/^#[0-9a-f]{6}$/);
		}
	});

	it('accent is the brand rose, independent of categorical palette', () => {
		expect(chartAccent).toBe('#e11d48');
		// Red/rose must not appear in the categorical palette (reserved for accent/negative only)
		expect([...chartPalette]).not.toContain(chartAccent);
	});

	it('gradient has two stops derived from the accent', () => {
		expect(chartAccentGradient).toHaveLength(2);
		for (const stop of chartAccentGradient) {
			expect(stop).toMatch(/^rgba\(225, 29, 72, /);
		}
	});

	it('exposes hex chrome + semantic tokens (no leftover Nord palette)', () => {
		const tokens = [
			chartInk,
			chartInkMuted,
			chartAxisLine,
			chartSplitLine,
			chartContribution,
			chartValue,
			chartPositive
		];
		for (const token of tokens) {
			expect(token).toMatch(/^#[0-9a-f]{6}$/);
		}
		// value is the primary blue (same hue as palette[0]), NOT the red accent
		expect(chartValue).toBe(chartPalette[0]);
		// contribution is an independent rose-400 tint, not a palette index
		expect(chartContribution).toBe('#fb7185');
	});
});
