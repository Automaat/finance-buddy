import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import AllocationDriftWidget from './AllocationDriftWidget.svelte';

const baseScope = {
	owner_user_id: null as number | null,
	total_value: 100000,
	target_sum_pct: 100,
	has_complete_target: true,
	items: [
		{
			category: 'stock',
			owner_user_id: null,
			current_value: 60000,
			current_percentage: 60,
			target_percentage: 60,
			drift_pp: 0,
			severity: 'ok' as const,
			rebalance_amount: 0
		}
	]
};

describe('AllocationDriftWidget', () => {
	it('renders nothing when no scopes are present', () => {
		const { container } = render(AllocationDriftWidget, {
			props: { drift: { scopes: [] }, owners: [] }
		});
		expect(container.textContent ?? '').not.toContain('Dryft alokacji');
	});

	it('renders scope label and category for household scope', () => {
		render(AllocationDriftWidget, {
			props: { drift: { scopes: [baseScope] }, owners: [] }
		});
		expect(screen.getByText('Dryft alokacji')).toBeTruthy();
		expect(screen.getByText('Wspólne')).toBeTruthy();
		expect(screen.getByText('Akcje')).toBeTruthy();
	});

	it('renders ok badge when severity is ok', () => {
		render(AllocationDriftWidget, {
			props: { drift: { scopes: [baseScope] }, owners: [] }
		});
		expect(screen.getByText('OK')).toBeTruthy();
	});

	it('renders warning badge with drift pp when severity is warning', () => {
		const scope = {
			...baseScope,
			items: [
				{
					...baseScope.items[0],
					current_percentage: 80,
					target_percentage: 60,
					drift_pp: 20,
					severity: 'warning' as const,
					rebalance_amount: -20000
				}
			]
		};
		render(AllocationDriftWidget, {
			props: { drift: { scopes: [scope] }, owners: [] }
		});
		expect(screen.getByText(/Dryft \+20\.0 pp/)).toBeTruthy();
		expect(screen.getByText(/SPRZEDAJ/)).toBeTruthy();
	});

	it('shows rebalance hint with DOKUP when under target', () => {
		const scope = {
			...baseScope,
			items: [
				{
					...baseScope.items[0],
					current_percentage: 40,
					target_percentage: 60,
					drift_pp: -20,
					severity: 'warning' as const,
					rebalance_amount: 20000
				}
			]
		};
		render(AllocationDriftWidget, {
			props: { drift: { scopes: [scope] }, owners: [] }
		});
		expect(screen.getByText(/DOKUP/)).toBeTruthy();
	});

	it('shows incomplete-target warning when sum != 100', () => {
		const scope = { ...baseScope, target_sum_pct: 90, has_complete_target: false };
		render(AllocationDriftWidget, {
			props: { drift: { scopes: [scope] }, owners: [] }
		});
		expect(screen.getByText(/nie sumują się do 100/)).toBeTruthy();
	});

	it('renders missing_target badge for untargeted holdings', () => {
		const scope = {
			...baseScope,
			items: [
				{
					...baseScope.items[0],
					category: 'crypto',
					current_percentage: 5,
					target_percentage: 0,
					drift_pp: 5,
					severity: 'missing_target' as const,
					rebalance_amount: -5000
				}
			]
		};
		render(AllocationDriftWidget, {
			props: { drift: { scopes: [scope] }, owners: [] }
		});
		expect(screen.getByText('Brak celu')).toBeTruthy();
	});

	it('uses owner name when scope has owner_user_id', () => {
		const scope = { ...baseScope, owner_user_id: 1 };
		render(AllocationDriftWidget, {
			props: {
				drift: { scopes: [scope] },
				owners: [{ id: 1, name: 'Marcin' }]
			}
		});
		expect(screen.getByText('Marcin')).toBeTruthy();
	});
});
