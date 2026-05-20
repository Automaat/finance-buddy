import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import Toast from './Toast.svelte';
import { toast } from '$lib/stores/toast.svelte';
import { flushSync, tick } from 'svelte';

describe('Toast', () => {
	beforeEach(() => {
		vi.useFakeTimers();
		for (const item of [...toast.items]) toast.dismiss(item.id);
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it('renders nothing when there are no toasts', () => {
		render(Toast);
		expect(screen.queryByRole('alert')).toBeNull();
		expect(screen.queryByRole('status')).toBeNull();
	});

	it('renders an error toast with role="alert"', async () => {
		render(Toast);
		toast.error('Coś poszło nie tak');
		flushSync();
		await tick();
		const alert = await screen.findByRole('alert');
		expect(alert.textContent).toContain('Coś poszło nie tak');
	});

	it('renders a success toast with role="status"', async () => {
		render(Toast);
		toast.success('Zapisano');
		flushSync();
		await tick();
		const status = await screen.findByRole('status');
		expect(status.textContent).toContain('Zapisano');
	});

	it('renders an info toast with role="status"', async () => {
		render(Toast);
		toast.info('FYI');
		flushSync();
		await tick();
		const status = await screen.findByRole('status');
		expect(status.textContent).toContain('FYI');
	});

	it('auto-dismisses after 4 seconds', async () => {
		render(Toast);
		toast.error('Will fade');
		flushSync();
		await tick();
		expect(screen.queryByRole('alert')).not.toBeNull();
		vi.advanceTimersByTime(4000);
		flushSync();
		await tick();
		expect(screen.queryByRole('alert')).toBeNull();
	});

	it('dismisses on close-button click', async () => {
		render(Toast);
		toast.error('Manual close');
		flushSync();
		await tick();
		await fireEvent.click(screen.getByLabelText('Zamknij powiadomienie'));
		flushSync();
		await tick();
		expect(screen.queryByRole('alert')).toBeNull();
	});

	it('stacks multiple toasts', async () => {
		render(Toast);
		toast.error('First');
		toast.success('Second');
		toast.info('Third');
		flushSync();
		await tick();
		expect(screen.getAllByText(/First|Second|Third/).length).toBe(3);
	});
});
