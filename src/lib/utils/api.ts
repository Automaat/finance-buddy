import { browser, dev } from '$app/environment';
import { env } from '$env/dynamic/public';

export const DEFAULT_DEV_API_URL = 'http://localhost:8000';
export const API_URL_NOT_CONFIGURED_MESSAGE = 'API URL is not configured';

export function resolveApiUrl(): string | undefined {
	const apiUrl = browser
		? env.PUBLIC_API_URL_BROWSER || env.PUBLIC_API_URL
		: env.PUBLIC_API_URL || env.PUBLIC_API_URL_BROWSER;

	return apiUrl || (dev ? DEFAULT_DEV_API_URL : undefined);
}

export function getApiUrlOrThrow(): string {
	const apiUrl = resolveApiUrl();

	if (apiUrl) {
		return apiUrl;
	}

	throw new Error(API_URL_NOT_CONFIGURED_MESSAGE);
}
