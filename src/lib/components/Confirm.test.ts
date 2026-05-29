import { describe, it, expect, afterEach } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import Confirm from './Confirm.svelte';
import { confirm, confirmDialog } from '$lib/stores/confirm.svelte';

describe('Confirm', () => {
	afterEach(() => {
		if (confirmDialog.current && !confirmDialog.current.pending) confirmDialog.cancel();
	});

	it('renders nothing when there is no pending request', () => {
		render(Confirm);
		expect(screen.queryByText('Usuń pozycję')).toBeNull();
	});

	it('renders the title + message of an open request', () => {
		render(Confirm);
		const p = confirm({ title: 'Usuń pozycję', message: 'Na pewno?' });
		return Promise.resolve().then(() => {
			expect(screen.getByText('Usuń pozycję')).toBeTruthy();
			expect(screen.getByText('Na pewno?')).toBeTruthy();
			confirmDialog.cancel();
			return p;
		});
	});

	it('resolves true when the request is confirmed', async () => {
		render(Confirm);
		const p = confirm({ title: 't', message: 'm' });
		confirmDialog.confirm();
		await expect(p).resolves.toBe(true);
	});
});
