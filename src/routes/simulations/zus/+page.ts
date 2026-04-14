import { error } from '@sveltejs/kit';
import { browser } from '$app/environment';
import { env } from '$env/dynamic/public';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	try {
		const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
		if (!apiUrl) throw error(500, 'API URL not configured');

		const [prefillRes, personasRes] = await Promise.all([
			fetch(`${apiUrl}/api/zus/prefill`),
			fetch(`${apiUrl}/api/personas`)
		]);
		if (!prefillRes.ok) throw error(prefillRes.status, 'Failed to load ZUS prefill data');

		const prefill = await prefillRes.json();
		const personas = personasRes.ok ? await personasRes.json() : [];
		return { prefill, personas };
	} catch (err) {
		if (err instanceof Error && 'status' in err) throw err;
		throw error(500, 'Failed to load ZUS calculator data');
	}
};
