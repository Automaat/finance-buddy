import { describe, it, expect } from 'vitest';
import { CATEGORY_LABELS, categoryLabel } from './categories';

describe('categoryLabel', () => {
	it('maps known categories to Polish labels', () => {
		expect(categoryLabel('bank')).toBe('Konto bankowe');
		expect(categoryLabel('saving_account')).toBe('Konto oszczędnościowe');
		expect(categoryLabel('stock')).toBe('Akcje');
		expect(categoryLabel('real_estate')).toBe('Nieruchomość');
	});

	it('capitalizes the raw key for unknown categories', () => {
		expect(categoryLabel('crypto')).toBe('Crypto');
	});

	it('exposes the canonical map', () => {
		expect(CATEGORY_LABELS.etf).toBe('ETF');
		expect(Object.keys(CATEGORY_LABELS)).toContain('installment');
	});
});
