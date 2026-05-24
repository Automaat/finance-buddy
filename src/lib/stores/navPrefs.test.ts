import { describe, expect, it, beforeEach } from 'vitest';
import { navPrefs, DEFAULT_PINNED, MAX_PINNED } from './navPrefs.svelte';

describe('navPrefs', () => {
	beforeEach(() => {
		window.localStorage.clear();
		navPrefs.reset();
	});

	it('defaults to DEFAULT_PINNED', () => {
		expect(navPrefs.pinned).toEqual([...DEFAULT_PINNED]);
	});

	it('persists set() to localStorage', () => {
		navPrefs.set(['/', '/accounts']);
		expect(navPrefs.pinned).toEqual(['/', '/accounts']);
		expect(JSON.parse(window.localStorage.getItem('fb.nav.pinned') ?? '[]')).toEqual([
			'/',
			'/accounts'
		]);
	});

	it('truncates to MAX_PINNED', () => {
		navPrefs.set(['/', '/a', '/b', '/c', '/d', '/e']);
		expect(navPrefs.pinned.length).toBe(MAX_PINNED);
	});

	it('reset() restores defaults', () => {
		navPrefs.set(['/accounts']);
		navPrefs.reset();
		expect(navPrefs.pinned).toEqual([...DEFAULT_PINNED]);
	});
});
