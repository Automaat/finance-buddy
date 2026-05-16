import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import NewAssetModal from './NewAssetModal.svelte';

describe('NewAssetModal', () => {
	it('renders an accessible dialog with the title', () => {
		render(NewAssetModal, { props: { onCreate: vi.fn(), onClose: vi.fn() } });
		const dialog = screen.getByRole('dialog');
		expect(dialog.getAttribute('aria-modal')).toBe('true');
		expect(dialog.getAttribute('aria-labelledby')).toBe('new-asset-modal-title');
		expect(screen.getByText('Dodaj nowy majątek')).toBeTruthy();
	});

	it('renders the name and value inputs', () => {
		render(NewAssetModal, { props: { onCreate: vi.fn(), onClose: vi.fn() } });
		expect(screen.getByLabelText('Nazwa *')).toBeTruthy();
		expect(screen.getByLabelText('Wartość początkowa')).toBeTruthy();
	});

	it('calls onCreate when the confirm button is clicked', async () => {
		const onCreate = vi.fn();
		render(NewAssetModal, { props: { onCreate, onClose: vi.fn() } });
		await fireEvent.click(screen.getByText('Utwórz majątek'));
		expect(onCreate).toHaveBeenCalledOnce();
	});

	it('disables the confirm button while creating', () => {
		render(NewAssetModal, { props: { creating: true, onCreate: vi.fn(), onClose: vi.fn() } });
		const button = screen.getByText('Tworzenie...') as HTMLButtonElement;
		expect(button.disabled).toBe(true);
	});

	it('closes via the × button', async () => {
		const onClose = vi.fn();
		render(NewAssetModal, { props: { onCreate: vi.fn(), onClose } });
		await fireEvent.click(screen.getByTitle('Zamknij'));
		expect(onClose).toHaveBeenCalledOnce();
	});

	it('closes on Escape', async () => {
		const onClose = vi.fn();
		render(NewAssetModal, { props: { onCreate: vi.fn(), onClose } });
		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(onClose).toHaveBeenCalledOnce();
	});

	it('closes on backdrop click', async () => {
		const onClose = vi.fn();
		const { container } = render(NewAssetModal, { props: { onCreate: vi.fn(), onClose } });
		const backdrop = container.querySelector('[role="presentation"]');
		expect(backdrop).not.toBeNull();
		await fireEvent.click(backdrop as Element);
		expect(onClose).toHaveBeenCalledOnce();
	});

	it('stays open when the dialog body is clicked', async () => {
		const onClose = vi.fn();
		render(NewAssetModal, { props: { onCreate: vi.fn(), onClose } });
		await fireEvent.click(screen.getByRole('dialog'));
		expect(onClose).not.toHaveBeenCalled();
	});
});
