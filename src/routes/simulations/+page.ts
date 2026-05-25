import { error } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import type { PageLoad } from './$types';
import type { OwnerOption } from '$lib/types/owners';
import { loadConfigDefaults } from '$lib/utils/configDefaults';

export const load: PageLoad = async ({ fetch }) => {
	try {
		const apiUrl = resolveApiUrl();

		const [prefillRes, ownersRes, defaults] = await Promise.all([
			fetch(`${apiUrl}/api/simulations/prefill`),
			fetch(`${apiUrl}/api/users`),
			loadConfigDefaults(fetch)
		]);
		if (!prefillRes.ok) throw error(prefillRes.status, 'Failed to load prefill data');

		const prefill = await prefillRes.json();
		const owners: OwnerOption[] = ownersRes.ok ? await ownersRes.json() : [];
		return { ...prefill, owners, defaults };
	} catch (err) {
		if (err instanceof Error && 'status' in err) throw err;
		throw error(500, 'Failed to load simulation data');
	}
};
