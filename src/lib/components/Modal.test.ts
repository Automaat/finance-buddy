import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import Modal from './Modal.svelte';

describe('Modal', () => {
	it('does not render when closed', () => {
		render(Modal, { props: { open: false, title: 'Tytuł' } });
		expect(screen.queryByRole('dialog')).toBeNull();
	});

	it('renders an accessible dialog with the title when open', () => {
		render(Modal, { props: { open: true, title: 'Tytuł' } });
		const dialog = screen.getByRole('dialog');
		expect(dialog.getAttribute('aria-modal')).toBe('true');
		expect(dialog.getAttribute('aria-labelledby')).toBe('modal-title');
		expect(screen.getByText('Tytuł')).toBeTruthy();
	});

	it('closes via the × button', async () => {
		const onCancel = vi.fn();
		render(Modal, { props: { open: true, title: 'Tytuł', onCancel } });
		await fireEvent.click(screen.getByLabelText('Zamknij'));
		expect(onCancel).toHaveBeenCalledOnce();
	});

	it('closes on Escape', async () => {
		const onCancel = vi.fn();
		render(Modal, { props: { open: true, title: 'Tytuł', onCancel } });
		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(onCancel).toHaveBeenCalledOnce();
	});

	it('closes on backdrop click', async () => {
		const onCancel = vi.fn();
		const { container } = render(Modal, { props: { open: true, title: 'Tytuł', onCancel } });
		const backdrop = container.querySelector('[role="presentation"]');
		expect(backdrop).not.toBeNull();
		await fireEvent.click(backdrop as Element);
		expect(onCancel).toHaveBeenCalledOnce();
	});

	it('stays open when the dialog body is clicked', async () => {
		const onCancel = vi.fn();
		render(Modal, { props: { open: true, title: 'Tytuł', onCancel } });
		await fireEvent.click(screen.getByRole('dialog'));
		expect(onCancel).not.toHaveBeenCalled();
	});

	it('confirms via the confirm button', async () => {
		const onConfirm = vi.fn();
		render(Modal, { props: { open: true, title: 'Tytuł', onConfirm } });
		await fireEvent.click(screen.getByText('Zapisz'));
		expect(onConfirm).toHaveBeenCalledOnce();
	});
});
