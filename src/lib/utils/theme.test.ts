import { describe, it, expect } from 'vitest';
import { chartPalette, chartAccent, chartAccentGradient } from './theme';

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

	it('accent is the first palette color', () => {
		expect(chartAccent).toBe(chartPalette[0]);
	});

	it('gradient has two stops derived from the accent', () => {
		expect(chartAccentGradient).toHaveLength(2);
		for (const stop of chartAccentGradient) {
			expect(stop).toMatch(/^rgba\(225, 29, 72, /);
		}
	});
});
