import { describe, expect, it, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import BottomSheet from './BottomSheet.svelte';

describe('BottomSheet', () => {
	it('does not render when closed', () => {
		render(BottomSheet, { props: { open: false, onClose: () => {} } });
		expect(screen.queryByRole('dialog')).toBeNull();
	});

	it('renders an accessible dialog with the title when open', () => {
		render(BottomSheet, { props: { open: true, title: 'Więcej', onClose: () => {} } });
		const dialog = screen.getByRole('dialog');
		expect(dialog.getAttribute('aria-modal')).toBe('true');
		expect(dialog.getAttribute('aria-label')).toBe('Więcej');
	});

	it('closes on Escape', async () => {
		const onClose = vi.fn();
		render(BottomSheet, { props: { open: true, onClose } });
		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(onClose).toHaveBeenCalledOnce();
	});

	it('closes on backdrop click', async () => {
		const onClose = vi.fn();
		const { container } = render(BottomSheet, { props: { open: true, onClose } });
		const backdrop = container.querySelector('[role="presentation"]');
		expect(backdrop).not.toBeNull();
		await fireEvent.click(backdrop as Element);
		expect(onClose).toHaveBeenCalledOnce();
	});

	it('does not close when the sheet body is clicked', async () => {
		const onClose = vi.fn();
		render(BottomSheet, { props: { open: true, onClose } });
		await fireEvent.click(screen.getByRole('dialog'));
		expect(onClose).not.toHaveBeenCalled();
	});

	it('closes after dragging the handle down past the threshold', async () => {
		const onClose = vi.fn();
		const { container } = render(BottomSheet, { props: { open: true, onClose } });
		const sheet = screen.getByRole('dialog');
		const handle = container.querySelector('[data-sheet-drag]');
		expect(handle).not.toBeNull();
		// jsdom lacks setPointerCapture — stub it.
		(sheet as Element & { setPointerCapture?: (id: number) => void }).setPointerCapture = () => {};
		await fireEvent.pointerDown(handle as Element, { clientY: 0, pointerId: 1 });
		await fireEvent.pointerMove(sheet, { clientY: 150, pointerId: 1 });
		await fireEvent.pointerUp(sheet, { clientY: 150, pointerId: 1 });
		expect(onClose).toHaveBeenCalledOnce();
	});

	it('does not close when drag distance is below the threshold', async () => {
		const onClose = vi.fn();
		const { container } = render(BottomSheet, { props: { open: true, onClose } });
		const sheet = screen.getByRole('dialog');
		const handle = container.querySelector('[data-sheet-drag]');
		(sheet as Element & { setPointerCapture?: (id: number) => void }).setPointerCapture = () => {};
		await fireEvent.pointerDown(handle as Element, { clientY: 0, pointerId: 1 });
		await fireEvent.pointerMove(sheet, { clientY: 40, pointerId: 1 });
		await fireEvent.pointerUp(sheet, { clientY: 40, pointerId: 1 });
		expect(onClose).not.toHaveBeenCalled();
	});

	it('ignores pointer drag that does not start on the handle', async () => {
		const onClose = vi.fn();
		render(BottomSheet, { props: { open: true, onClose } });
		const sheet = screen.getByRole('dialog');
		(sheet as Element & { setPointerCapture?: (id: number) => void }).setPointerCapture = () => {};
		await fireEvent.pointerDown(sheet, { clientY: 0, pointerId: 1 });
		await fireEvent.pointerMove(sheet, { clientY: 300, pointerId: 1 });
		await fireEvent.pointerUp(sheet, { clientY: 300, pointerId: 1 });
		expect(onClose).not.toHaveBeenCalled();
	});
});
