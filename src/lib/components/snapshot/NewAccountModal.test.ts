import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import NewAccountModal from './NewAccountModal.svelte';

const personas = [
	{ id: 1, name: 'Marcin' },
	{ id: 2, name: 'Ewa' }
];

describe('NewAccountModal', () => {
	it('renders an accessible dialog with the title', () => {
		render(NewAccountModal, {
			props: {
				section: 'financial',
				onCreate: vi.fn(),
				onClose: vi.fn(),
				personas
			}
		});
		const dialog = screen.getByRole('dialog');
		expect(dialog.getAttribute('aria-modal')).toBe('true');
		expect(dialog.getAttribute('aria-labelledby')).toBe('new-account-modal-title');
		expect(screen.getByText('Dodaj nowe konto')).toBeTruthy();
	});

	it('shows financial category options for the financial section', () => {
		render(NewAccountModal, {
			props: { section: 'financial', onCreate: vi.fn(), onClose: vi.fn(), personas }
		});
		expect(screen.getByRole('option', { name: 'Konto bankowe' })).toBeTruthy();
		expect(screen.queryByRole('option', { name: 'Hipoteka' })).toBeNull();
	});

	it('shows the wrapper select only for the retirement section', () => {
		render(NewAccountModal, {
			props: { section: 'retirement', onCreate: vi.fn(), onClose: vi.fn(), personas }
		});
		expect(screen.getByLabelText('Wrapper *')).toBeTruthy();
	});

	it('hides the wrapper select for non-retirement sections', () => {
		render(NewAccountModal, {
			props: { section: 'liabilities', onCreate: vi.fn(), onClose: vi.fn(), personas }
		});
		expect(screen.queryByLabelText('Wrapper *')).toBeNull();
	});

	it('calls onCreate when the confirm button is clicked', async () => {
		const onCreate = vi.fn();
		render(NewAccountModal, {
			props: { section: 'financial', onCreate, onClose: vi.fn(), personas }
		});
		await fireEvent.click(screen.getByText('Utwórz konto'));
		expect(onCreate).toHaveBeenCalledOnce();
	});

	it('disables the confirm button while creating', () => {
		render(NewAccountModal, {
			props: { section: 'financial', creating: true, onCreate: vi.fn(), onClose: vi.fn(), personas }
		});
		const button = screen.getByText('Tworzenie...') as HTMLButtonElement;
		expect(button.disabled).toBe(true);
	});

	it('closes via the × button', async () => {
		const onClose = vi.fn();
		render(NewAccountModal, {
			props: { section: 'financial', onCreate: vi.fn(), onClose, personas }
		});
		await fireEvent.click(screen.getByTitle('Zamknij'));
		expect(onClose).toHaveBeenCalledOnce();
	});

	it('closes on Escape', async () => {
		const onClose = vi.fn();
		render(NewAccountModal, {
			props: { section: 'financial', onCreate: vi.fn(), onClose, personas }
		});
		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(onClose).toHaveBeenCalledOnce();
	});

	it('closes on backdrop click', async () => {
		const onClose = vi.fn();
		const { container } = render(NewAccountModal, {
			props: { section: 'financial', onCreate: vi.fn(), onClose, personas }
		});
		const backdrop = container.querySelector('[role="presentation"]');
		expect(backdrop).not.toBeNull();
		await fireEvent.click(backdrop as Element);
		expect(onClose).toHaveBeenCalledOnce();
	});

	it('stays open when the dialog body is clicked', async () => {
		const onClose = vi.fn();
		render(NewAccountModal, {
			props: { section: 'financial', onCreate: vi.fn(), onClose, personas }
		});
		await fireEvent.click(screen.getByRole('dialog'));
		expect(onClose).not.toHaveBeenCalled();
	});

	it('renders one owner option per persona', () => {
		render(NewAccountModal, {
			props: { section: 'financial', onCreate: vi.fn(), onClose: vi.fn(), personas }
		});
		expect(screen.getByRole('option', { name: 'Marcin' })).toBeTruthy();
		expect(screen.getByRole('option', { name: 'Ewa' })).toBeTruthy();
	});
});
