import { describe, it, expect, vi, beforeAll } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import ValuationFormModal, { type ValuationFormData } from './ValuationFormModal.svelte';

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

function makeData(overrides: Partial<ValuationFormData> = {}): ValuationFormData {
	return {
		company: 'Acme',
		date: '2026-05-01',
		currency: 'USD',
		fmv_per_share: 12.5,
		fmv_low: null,
		fmv_high: null,
		source: '409a',
		common_stock_discount_pct: null,
		notes: '',
		...overrides
	};
}

const baseProps = {
	open: true,
	editing: false,
	error: '',
	saving: false,
	onSave: vi.fn(),
	onCancel: vi.fn()
};

describe('ValuationFormModal', () => {
	it('renders the New title when not editing', () => {
		render(ValuationFormModal, { props: { ...baseProps, data: makeData() } });
		expect(screen.getByText('Nowa wycena')).toBeTruthy();
	});

	it('renders the Edit title when editing', () => {
		render(ValuationFormModal, {
			props: { ...baseProps, editing: true, data: makeData() }
		});
		expect(screen.getByText('Edytuj wycenę')).toBeTruthy();
	});

	it('renders error block when error prop is set', () => {
		render(ValuationFormModal, {
			props: { ...baseProps, error: 'Bzdura', data: makeData() }
		});
		expect(screen.getByText('Bzdura')).toBeTruthy();
	});

	it('shows saving label and disables confirm while saving', () => {
		render(ValuationFormModal, {
			props: { ...baseProps, saving: true, data: makeData() }
		});
		const confirmBtn = screen.getByRole('button', { name: 'Zapisywanie...' }) as HTMLButtonElement;
		expect(confirmBtn.disabled).toBe(true);
	});

	it('renders the four valuation sources', () => {
		render(ValuationFormModal, { props: { ...baseProps, data: makeData() } });
		expect(screen.getByText('409A')).toBeTruthy();
		expect(screen.getByText(/Runda uprzywilejowana/i)).toBeTruthy();
		expect(screen.getByText(/Oferta odkupu/i)).toBeTruthy();
		expect(screen.getByText(/Estymacja/i)).toBeTruthy();
	});

	it('invokes onSave when submitting via the form (preventDefault)', async () => {
		const onSave = vi.fn();
		const { container } = render(ValuationFormModal, {
			props: { ...baseProps, onSave, data: makeData() }
		});
		const form = container.querySelector('form')!;
		await fireEvent.submit(form);
		expect(onSave).toHaveBeenCalled();
	});

	it('invokes onSave when the modal confirm button is clicked', async () => {
		const onSave = vi.fn();
		render(ValuationFormModal, {
			props: { ...baseProps, onSave, data: makeData() }
		});
		await fireEvent.click(screen.getByRole('button', { name: 'Zapisz' }));
		expect(onSave).toHaveBeenCalled();
	});

	it('invokes onCancel when the cancel button is clicked', async () => {
		const onCancel = vi.fn();
		render(ValuationFormModal, {
			props: { ...baseProps, onCancel, data: makeData() }
		});
		await fireEvent.click(screen.getByRole('button', { name: /Anuluj|Cancel/ }));
		expect(onCancel).toHaveBeenCalled();
	});

	it('renders the common stock discount input with placeholder', () => {
		render(ValuationFormModal, { props: { ...baseProps, data: makeData() } });
		const placeholder = screen.getByPlaceholderText('np. 30');
		expect(placeholder).toBeTruthy();
	});
});
