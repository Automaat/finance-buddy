import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
	API_URL_NOT_CONFIGURED_MESSAGE,
	DEFAULT_DEV_API_URL,
	getApiUrlOrThrow,
	resolveApiUrl
} from './api';

const state = vi.hoisted(() => ({
	browser: false,
	dev: false,
	env: {
		PUBLIC_API_URL: undefined as string | undefined,
		PUBLIC_API_URL_BROWSER: undefined as string | undefined
	}
}));

vi.mock('$app/environment', () => ({
	get browser() {
		return state.browser;
	},
	get dev() {
		return state.dev;
	}
}));

vi.mock('$env/dynamic/public', () => ({
	get env() {
		return state.env;
	}
}));

describe('api utils', () => {
	beforeEach(() => {
		state.browser = false;
		state.dev = false;
		state.env.PUBLIC_API_URL = undefined;
		state.env.PUBLIC_API_URL_BROWSER = undefined;
	});

	it('prefers the server API URL during SSR', () => {
		state.env.PUBLIC_API_URL = 'http://backend:8000';
		state.env.PUBLIC_API_URL_BROWSER = 'https://app.example.com/api';

		expect(resolveApiUrl()).toBe('http://backend:8000');
	});

	it('prefers the browser API URL in the browser', () => {
		state.browser = true;
		state.env.PUBLIC_API_URL = 'http://backend:8000';
		state.env.PUBLIC_API_URL_BROWSER = 'https://app.example.com/api';

		expect(resolveApiUrl()).toBe('https://app.example.com/api');
	});

	it('falls back to the other public URL when only one is configured', () => {
		state.browser = true;
		state.env.PUBLIC_API_URL = 'http://backend:8000';

		expect(resolveApiUrl()).toBe('http://backend:8000');

		state.browser = false;
		state.env.PUBLIC_API_URL = undefined;
		state.env.PUBLIC_API_URL_BROWSER = 'https://app.example.com/api';

		expect(resolveApiUrl()).toBe('https://app.example.com/api');
	});

	it('uses the local backend in development when no API URL is configured', () => {
		state.dev = true;

		expect(resolveApiUrl()).toBe(DEFAULT_DEV_API_URL);
		expect(getApiUrlOrThrow()).toBe(DEFAULT_DEV_API_URL);
	});

	it('keeps production misconfiguration loud', () => {
		expect(resolveApiUrl()).toBeUndefined();
		expect(() => getApiUrlOrThrow()).toThrow(API_URL_NOT_CONFIGURED_MESSAGE);
	});
});
