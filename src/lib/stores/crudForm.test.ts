import { describe, it, expect } from 'vitest';
import { CrudForm } from './crudForm.svelte';

interface Item {
	id: number;
	name: string;
}

describe('CrudForm', () => {
	it('starts closed and not editing', () => {
		const form = new CrudForm<Item>();
		expect(form.open).toBe(false);
		expect(form.editing).toBeNull();
		expect(form.isEditing).toBe(false);
		expect(form.saving).toBe(false);
		expect(form.error).toBe('');
	});

	it('openCreate opens without an editing target', () => {
		const form = new CrudForm<Item>();
		form.error = 'stale';
		form.openCreate();
		expect(form.open).toBe(true);
		expect(form.editing).toBeNull();
		expect(form.isEditing).toBe(false);
		expect(form.error).toBe('');
	});

	it('openEdit opens with the item', () => {
		const form = new CrudForm<Item>();
		const item = { id: 1, name: 'x' };
		form.openEdit(item);
		expect(form.open).toBe(true);
		expect(form.editing).toEqual(item);
		expect(form.isEditing).toBe(true);
	});

	it('close resets open/editing/error', () => {
		const form = new CrudForm<Item>();
		form.openEdit({ id: 1, name: 'x' });
		form.error = 'boom';
		form.close();
		expect(form.open).toBe(false);
		expect(form.editing).toBeNull();
		expect(form.error).toBe('');
	});

	it('submit closes and returns true on success', async () => {
		const form = new CrudForm<Item>();
		form.openCreate();
		let ran = false;
		const ok = await form.submit(async () => {
			ran = true;
		});
		expect(ran).toBe(true);
		expect(ok).toBe(true);
		expect(form.open).toBe(false);
		expect(form.saving).toBe(false);
		expect(form.error).toBe('');
	});

	it('submit keeps the form open and records the error on failure', async () => {
		const form = new CrudForm<Item>();
		form.openCreate();
		const ok = await form.submit(async () => {
			throw new Error('nope');
		});
		expect(ok).toBe(false);
		expect(form.open).toBe(true);
		expect(form.saving).toBe(false);
		expect(form.error).toBe('nope');
	});

	it('submit clears a prior error before re-running', async () => {
		const form = new CrudForm<Item>();
		form.openCreate();
		await form.submit(async () => {
			throw new Error('first');
		});
		expect(form.error).toBe('first');
		await form.submit(async () => {});
		expect(form.error).toBe('');
	});
});
