import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { loadConfigDefaults } from './configDefaults';

vi.mock('$env/dynamic/public', () => ({
	env: {
		PUBLIC_API_URL_BROWSER: 'http://localhost:8000',
		PUBLIC_API_URL: 'http://localhost:8000'
	}
}));

vi.mock('$app/environment', () => ({ browser: false }));

function makeResponse(body: unknown, ok = true): Response {
	return {
		ok,
		json: async () => body
	} as Response;
}

describe('loadConfigDefaults', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		// Pin "now" so age-from-birth is deterministic.
		vi.setSystemTime(new Date('2026-05-25'));
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it('maps app_config fields onto simulator defaults', async () => {
		const fetchFn = vi.fn(async () =>
			makeResponse({
				expected_return_rate: '0.07',
				withdrawal_rate: '0.04',
				birth_date: '1990-05-25',
				retirement_age: 65,
				monthly_expenses: '8000',
				monthly_mortgage_payment: '2500'
			})
		) as unknown as typeof fetch;

		const defaults = await loadConfigDefaults(fetchFn);

		expect(defaults.annualReturnPct).toBeCloseTo(7);
		expect(defaults.withdrawalRatePct).toBeCloseTo(4);
		expect(defaults.currentAge).toBe(36);
		expect(defaults.retirementAge).toBe(65);
		expect(defaults.monthlyExpensesPLN).toBe(8000);
		expect(defaults.annualExpensesPLN).toBe(96000);
		expect(defaults.monthlyMortgagePLN).toBe(2500);
	});

	it('subtracts a year when birthday has not passed yet this year', async () => {
		vi.setSystemTime(new Date('2026-05-24'));
		const fetchFn = vi.fn(async () =>
			makeResponse({ birth_date: '1990-05-25', retirement_age: 67 })
		) as unknown as typeof fetch;

		const defaults = await loadConfigDefaults(fetchFn);
		expect(defaults.currentAge).toBe(35);
		expect(defaults.retirementAge).toBe(67);
	});

	it('falls back to hardcoded defaults when /api/config is unreachable', async () => {
		const fetchFn = vi.fn(async () => makeResponse({}, false)) as unknown as typeof fetch;

		const defaults = await loadConfigDefaults(fetchFn);
		expect(defaults.annualReturnPct).toBe(7);
		expect(defaults.withdrawalRatePct).toBe(4);
		expect(defaults.currentAge).toBeNull();
		expect(defaults.retirementAge).toBeNull();
		expect(defaults.monthlyExpensesPLN).toBeNull();
		expect(defaults.monthlyMortgagePLN).toBeNull();
	});

	it('falls back when the fetch itself throws', async () => {
		const fetchFn = vi.fn(async () => {
			throw new Error('network down');
		}) as unknown as typeof fetch;

		const defaults = await loadConfigDefaults(fetchFn);
		expect(defaults.annualReturnPct).toBe(7);
		expect(defaults.currentAge).toBeNull();
	});

	it('returns null age when birth_date is unparseable', async () => {
		const fetchFn = vi.fn(async () =>
			makeResponse({ birth_date: 'not-a-date', retirement_age: 60 })
		) as unknown as typeof fetch;

		const defaults = await loadConfigDefaults(fetchFn);
		expect(defaults.currentAge).toBeNull();
		expect(defaults.retirementAge).toBe(60);
	});
});
