import { error } from '@sveltejs/kit';
import { API_URL_NOT_CONFIGURED_MESSAGE, resolveApiUrl } from '$lib/utils/api';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	const apiUrl = resolveApiUrl();
	if (!apiUrl) throw error(500, API_URL_NOT_CONFIGURED_MESSAGE);

	try {
		const [prefillRes, personasRes] = await Promise.all([
			fetch(`${apiUrl}/api/simulations/prefill`),
			fetch(`${apiUrl}/api/personas`)
		]);
		if (!prefillRes.ok) throw error(prefillRes.status, 'Failed to load prefill data');

		const prefill = await prefillRes.json();
		const personas = personasRes.ok ? await personasRes.json() : [];
		return { ...prefill, personas };
	} catch (err) {
		if (err && typeof err === 'object' && 'status' in err) throw err;
		throw error(500, 'Failed to load simulation data');
	}
};
