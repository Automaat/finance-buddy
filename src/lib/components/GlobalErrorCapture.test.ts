import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest';
import { render } from '@testing-library/svelte';
import GlobalErrorCapture from './GlobalErrorCapture.svelte';
import { toast } from '$lib/stores/toast.svelte';

describe('GlobalErrorCapture', () => {
	let toastSpy: ReturnType<typeof vi.spyOn>;
	let consoleSpy: ReturnType<typeof vi.spyOn>;

	beforeEach(() => {
		toastSpy = vi.spyOn(toast, 'error').mockImplementation(() => 0);
		consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});
	});

	afterEach(() => {
		toastSpy.mockRestore();
		consoleSpy.mockRestore();
	});

	it('toasts on window error events', () => {
		render(GlobalErrorCapture);
		window.dispatchEvent(new ErrorEvent('error', { message: 'oops', error: new Error('boom') }));
		expect(toastSpy).toHaveBeenCalledOnce();
		expect(toastSpy.mock.calls[0][0]).toContain('boom');
	});

	it('toasts on unhandled promise rejections', () => {
		render(GlobalErrorCapture);
		const promise = Promise.reject(new Error('rejected'));
		promise.catch(() => {}); // suppress vitest unhandled-rejection warning
		// PromiseRejectionEvent may not be implemented in every jsdom build —
		// fall back to a plain Event with the same shape.
		type RejectionLike = Event & { promise: Promise<unknown>; reason: unknown };
		let event: RejectionLike;
		if (typeof PromiseRejectionEvent === 'function') {
			event = new PromiseRejectionEvent('unhandledrejection', {
				promise,
				reason: new Error('rejected')
			}) as RejectionLike;
		} else {
			const ev = new Event('unhandledrejection') as RejectionLike;
			ev.promise = promise;
			ev.reason = new Error('rejected');
			event = ev;
		}
		window.dispatchEvent(event);
		expect(toastSpy).toHaveBeenCalledOnce();
		expect(toastSpy.mock.calls[0][0]).toContain('rejected');
	});

	it('dedupes repeated identical errors within 5s', () => {
		render(GlobalErrorCapture);
		const err = new Error('same');
		window.dispatchEvent(new ErrorEvent('error', { message: 'same', error: err }));
		window.dispatchEvent(new ErrorEvent('error', { message: 'same', error: err }));
		window.dispatchEvent(new ErrorEvent('error', { message: 'same', error: err }));
		expect(toastSpy).toHaveBeenCalledTimes(1);
	});
});
