import { describe, it, expect, vi, beforeAll } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import EquityFormModal, { type VestingPreset } from './EquityFormModal.svelte';
import type { EquityFormData } from './EquityFormModal.svelte';
import type { EquityTaxTreatment } from '$lib/types/salaries';

beforeAll(() => {
	Object.defineProperty(window, 'matchMedia', {
		writable: true,
		value: vi.fn().mockImplementation((query: string) => ({
			matches: false,
			media: query,
			onchange: null,
			addListener: vi.fn(),
			removeListener: vi.fn(),
			addEventListener: vi.fn(),
			removeEventListener: vi.fn(),
			dispatchEvent: vi.fn()
		}))
	});
});

function makeData(overrides: Partial<EquityFormData> = {}): EquityFormData {
	return {
		grant_date: '2026-01-01',
		type: 'rsu',
		company: 'Acme',
		owner_user_id: 1,
		total_shares: 1000,
		strike_price: null,
		currency: 'USD',
		vest_start_date: '2026-01-01',
		vest_cliff_months: 12,
		vest_total_months: 48,
		vest_frequency: 'monthly',
		preset: '4yr_1yrcliff',
		vest_custom_schedule: null,
		requires_liquidity_event: false,
		liquidity_event_date: null,
		tax_treatment: 'capital_gains_19',
		notes: '',
		...overrides
	};
}

const presets: Record<string, VestingPreset> = {
	'4yr_1yrcliff': {
		label: '4-letni z 1-letnim cliffem',
		cliff: 12,
		total: 48,
		frequency: 'monthly',
		custom: null
	},
	custom: {
		label: 'Niestandardowy',
		cliff: 0,
		total: 0,
		frequency: 'monthly',
		custom: []
	}
};

const taxTreatmentLabels: Record<EquityTaxTreatment, string> = {
	capital_gains_19: 'Kapitałowy (19%)',
	employment_income: 'Wynagrodzenie'
};

const baseProps = {
	open: true,
	editing: false,
	error: '',
	saving: false,
	today: '2026-05-20',
	owners: [
		{ id: 1, name: 'Marcin' },
		{ id: 2, name: 'Ewa' }
	],
	vestingPresets: presets,
	taxTreatmentLabels,
	onApplyPreset: vi.fn(),
	onSave: vi.fn(),
	onCancel: vi.fn()
};

describe('EquityFormModal', () => {
	it('renders the New title when not editing', () => {
		render(EquityFormModal, { props: { ...baseProps, data: makeData() } });
		expect(screen.getByText('Nowy grant')).toBeTruthy();
	});

	it('renders the Edit title when editing', () => {
		render(EquityFormModal, {
			props: { ...baseProps, editing: true, data: makeData() }
		});
		expect(screen.getByText('Edytuj grant')).toBeTruthy();
	});

	it('renders error block when error prop is set', () => {
		render(EquityFormModal, {
			props: { ...baseProps, error: 'Coś poszło nie tak', data: makeData() }
		});
		expect(screen.getByText('Coś poszło nie tak')).toBeTruthy();
	});

	it('shows saving label and disables confirm while saving', () => {
		render(EquityFormModal, {
			props: { ...baseProps, saving: true, data: makeData() }
		});
		const confirmBtn = screen.getByRole('button', { name: 'Zapisywanie...' }) as HTMLButtonElement;
		expect(confirmBtn.disabled).toBe(true);
	});

	it('hides strike price input for rsu and shows it for options', async () => {
		const { rerender } = render(EquityFormModal, {
			props: { ...baseProps, data: makeData({ type: 'rsu' }) }
		});
		expect(screen.queryByText(/Strike price/i)).toBeNull();
		await rerender({ ...baseProps, data: makeData({ type: 'option', strike_price: 1.5 }) });
		expect(screen.getByText(/Strike price/i)).toBeTruthy();
	});

	it('calls onApplyPreset when preset select changes', async () => {
		const onApplyPreset = vi.fn();
		render(EquityFormModal, {
			props: { ...baseProps, onApplyPreset, data: makeData() }
		});
		const schemat = screen.getByText('Schemat').parentElement!.querySelector('select')!;
		await fireEvent.change(schemat, { target: { value: 'custom' } });
		expect(onApplyPreset).toHaveBeenCalledWith('custom');
	});

	it('renders custom schedule editor when preset is custom', () => {
		render(EquityFormModal, {
			props: {
				...baseProps,
				data: makeData({
					preset: 'custom',
					vest_custom_schedule: [{ month: 12, pct: 25 }]
				})
			}
		});
		expect(screen.getByText(/Niestandardowy harmonogram/)).toBeTruthy();
		expect(screen.getByRole('button', { name: /Dodaj zdarzenie/i })).toBeTruthy();
		expect(screen.getByRole('button', { name: /Usuń wiersz/i })).toBeTruthy();
	});

	it('adds a custom schedule row when "Dodaj zdarzenie" is clicked', async () => {
		const data = makeData({ preset: 'custom', vest_custom_schedule: [] });
		render(EquityFormModal, { props: { ...baseProps, data } });
		await fireEvent.click(screen.getByRole('button', { name: /Dodaj zdarzenie/i }));
		expect(data.vest_custom_schedule).toEqual([{ month: 0, pct: 0 }]);
	});

	it('removes a custom schedule row when trash icon is clicked', async () => {
		const data = makeData({
			preset: 'custom',
			vest_custom_schedule: [
				{ month: 6, pct: 10 },
				{ month: 12, pct: 20 }
			]
		});
		render(EquityFormModal, { props: { ...baseProps, data } });
		const removeBtns = screen.getAllByRole('button', { name: /Usuń wiersz/i });
		await fireEvent.click(removeBtns[0]);
		expect(data.vest_custom_schedule).toEqual([{ month: 12, pct: 20 }]);
	});

	it('shows liquidity event date input only when the checkbox is checked', async () => {
		const { rerender } = render(EquityFormModal, {
			props: { ...baseProps, data: makeData({ requires_liquidity_event: false }) }
		});
		expect(screen.queryByText(/Data liquidity event/i)).toBeNull();
		await rerender({ ...baseProps, data: makeData({ requires_liquidity_event: true }) });
		expect(screen.getByText(/Data liquidity event/i)).toBeTruthy();
	});

	it('lists every owner in the owner select', () => {
		render(EquityFormModal, { props: { ...baseProps, data: makeData() } });
		expect(screen.getByText('Marcin')).toBeTruthy();
		expect(screen.getByText('Ewa')).toBeTruthy();
	});

	it('lists every tax treatment label', () => {
		render(EquityFormModal, { props: { ...baseProps, data: makeData() } });
		expect(screen.getByText('Kapitałowy (19%)')).toBeTruthy();
		expect(screen.getByText('Wynagrodzenie')).toBeTruthy();
	});

	it('invokes onSave when submitting via the form (preventDefault)', async () => {
		const onSave = vi.fn();
		const { container } = render(EquityFormModal, {
			props: { ...baseProps, onSave, data: makeData() }
		});
		const form = container.querySelector('form')!;
		await fireEvent.submit(form);
		expect(onSave).toHaveBeenCalled();
	});

	it('invokes onSave when the modal confirm button is clicked', async () => {
		const onSave = vi.fn();
		render(EquityFormModal, {
			props: { ...baseProps, onSave, data: makeData() }
		});
		await fireEvent.click(screen.getByRole('button', { name: 'Zapisz' }));
		expect(onSave).toHaveBeenCalled();
	});

	it('invokes onCancel when the modal cancel button is clicked', async () => {
		const onCancel = vi.fn();
		render(EquityFormModal, {
			props: { ...baseProps, onCancel, data: makeData() }
		});
		await fireEvent.click(screen.getByRole('button', { name: /Anuluj|Cancel/ }));
		expect(onCancel).toHaveBeenCalled();
	});
});
