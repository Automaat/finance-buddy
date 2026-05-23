import { describe, it, expect, beforeEach } from 'vitest';
import { confirm, confirmDialog } from './confirm.svelte';

describe('confirm store', () => {
	beforeEach(() => {
		// Drain any prior request.
		if (confirmDialog.current && !confirmDialog.current.pending) {
			confirmDialog.cancel();
		}
	});

	it('exposes the open request', () => {
		const p = confirm({ title: 'Delete', message: 'Sure?' });
		expect(confirmDialog.current).not.toBeNull();
		expect(confirmDialog.current?.title).toBe('Delete');
		expect(confirmDialog.current?.message).toBe('Sure?');
		expect(confirmDialog.current?.pending).toBe(false);
		confirmDialog.cancel();
		return p;
	});

	it('resolves true when confirm() called (no handler)', async () => {
		const p = confirm({ title: 't', message: 'm' });
		confirmDialog.confirm();
		await expect(p).resolves.toBe(true);
		expect(confirmDialog.current).toBeNull();
	});

	it('resolves false when cancel() called', async () => {
		const p = confirm({ title: 't', message: 'm' });
		confirmDialog.cancel();
		await expect(p).resolves.toBe(false);
		expect(confirmDialog.current).toBeNull();
	});

	it('opening a second request cancels the first', async () => {
		const first = confirm({ title: 'a', message: 'a' });
		const second = confirm({ title: 'b', message: 'b' });
		await expect(first).resolves.toBe(false);
		expect(confirmDialog.current?.title).toBe('b');
		confirmDialog.confirm();
		await expect(second).resolves.toBe(true);
	});

	it('passes options through unchanged', () => {
		const p = confirm({
			title: 'X',
			message: 'Y',
			confirmText: 'Yes',
			cancelText: 'No',
			danger: true
		});
		const req = confirmDialog.current;
		expect(req?.confirmText).toBe('Yes');
		expect(req?.cancelText).toBe('No');
		expect(req?.danger).toBe(true);
		confirmDialog.cancel();
		return p;
	});

	it('with onConfirm: flips pending, awaits, then resolves true', async () => {
		let resolveHandler: () => void = () => {};
		let handlerStarted!: () => void;
		const started = new Promise<void>((res) => {
			handlerStarted = res;
		});
		const p = confirm({
			title: 't',
			message: 'm',
			onConfirm: () =>
				new Promise<void>((resolveOnConfirm) => {
					resolveHandler = resolveOnConfirm;
					handlerStarted();
				})
		});
		confirmDialog.confirm();
		await started;
		expect(confirmDialog.current?.pending).toBe(true);
		// Cancel during pending is a no-op.
		confirmDialog.cancel();
		expect(confirmDialog.current).not.toBeNull();
		// Opening a second confirm while one is pending must not clobber.
		const ignored = confirm({ title: 'x', message: 'x' });
		await expect(ignored).resolves.toBe(false);
		expect(confirmDialog.current?.title).toBe('t');
		resolveHandler();
		await expect(p).resolves.toBe(true);
		expect(confirmDialog.current).toBeNull();
	});

	it('with onConfirm that throws: resolves false and closes', async () => {
		const p = confirm({
			title: 't',
			message: 'm',
			onConfirm: () => {
				throw new Error('boom');
			}
		});
		confirmDialog.confirm();
		await expect(p).resolves.toBe(false);
		expect(confirmDialog.current).toBeNull();
	});
});
