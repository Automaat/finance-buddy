import { browser } from '$app/environment';

const STORAGE_KEY = 'fb.nav.pinned';

export const DEFAULT_PINNED: ReadonlyArray<string> = ['/', '/snapshots', '/accounts', '/goals'];

export const MAX_PINNED = 4;

function load(): string[] {
	if (!browser) return [...DEFAULT_PINNED];
	try {
		const raw = window.localStorage.getItem(STORAGE_KEY);
		if (!raw) return [...DEFAULT_PINNED];
		const parsed = JSON.parse(raw);
		if (!Array.isArray(parsed)) return [...DEFAULT_PINNED];
		const filtered = parsed.filter((v): v is string => typeof v === 'string').slice(0, MAX_PINNED);
		return filtered.length > 0 ? filtered : [...DEFAULT_PINNED];
	} catch {
		return [...DEFAULT_PINNED];
	}
}

function save(value: ReadonlyArray<string>): void {
	if (!browser) return;
	try {
		window.localStorage.setItem(STORAGE_KEY, JSON.stringify(value));
	} catch {
		// ignore quota / private mode errors
	}
}

const state = $state<{ pinned: string[] }>({ pinned: load() });

export const navPrefs = {
	get pinned(): ReadonlyArray<string> {
		return state.pinned;
	},
	set(next: ReadonlyArray<string>): void {
		state.pinned = next.slice(0, MAX_PINNED);
		save(state.pinned);
	},
	reset(): void {
		state.pinned = [...DEFAULT_PINNED];
		save(state.pinned);
	}
};
