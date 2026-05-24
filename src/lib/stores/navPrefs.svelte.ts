import { browser } from '$app/environment';
import { NAV_HREFS } from '$lib/nav/routes';

const STORAGE_KEY = 'fb.nav.pinned';

export const DEFAULT_PINNED: ReadonlyArray<string> = ['/', '/snapshots', '/accounts', '/goals'];

export const MAX_PINNED = 4;

function sanitize(value: unknown): string[] {
	if (!Array.isArray(value)) return [...DEFAULT_PINNED];
	const seen = new Set<string>();
	const out: string[] = [];
	for (const v of value) {
		if (typeof v !== 'string') continue;
		if (!NAV_HREFS.has(v)) continue;
		if (seen.has(v)) continue;
		seen.add(v);
		out.push(v);
		if (out.length === MAX_PINNED) break;
	}
	return out.length > 0 ? out : [...DEFAULT_PINNED];
}

function load(): string[] {
	if (!browser) return [...DEFAULT_PINNED];
	try {
		const raw = window.localStorage.getItem(STORAGE_KEY);
		if (!raw) return [...DEFAULT_PINNED];
		return sanitize(JSON.parse(raw));
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
		state.pinned = sanitize(next);
		save(state.pinned);
	},
	reset(): void {
		state.pinned = [...DEFAULT_PINNED];
		save(state.pinned);
	}
};
