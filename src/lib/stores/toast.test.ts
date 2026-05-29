import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { toast } from './toast.svelte';

describe('toast store', () => {
	beforeEach(() => {
		// Drain any leftover toasts between tests.
		for (const t of [...toast.items]) toast.dismiss(t.id);
	});
	afterEach(() => {
		vi.useRealTimers();
	});

	it('pushes error/success/info toasts with kind + message', () => {
		const e = toast.error('boom');
		const s = toast.success('yay');
		const i = toast.info('fyi');
		const kinds = toast.items.map((t) => `${t.kind}:${t.message}`);
		expect(kinds).toEqual(['error:boom', 'success:yay', 'info:fyi']);
		expect(new Set([e, s, i]).size).toBe(3); // unique ids
	});

	it('dismiss removes a toast by id', () => {
		const id = toast.error('x');
		expect(toast.items).toHaveLength(1);
		toast.dismiss(id);
		expect(toast.items).toHaveLength(0);
	});

	it('auto-dismisses after the duration', () => {
		vi.useFakeTimers();
		toast.success('temp', 1000);
		expect(toast.items).toHaveLength(1);
		vi.advanceTimersByTime(1000);
		expect(toast.items).toHaveLength(0);
	});

	it('does not auto-dismiss when duration is 0', () => {
		vi.useFakeTimers();
		toast.info('sticky', 0);
		vi.advanceTimersByTime(100000);
		expect(toast.items).toHaveLength(1);
	});
});
