import { error } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import type { PageLoad } from './$types';
import type { OwnerOption } from '$lib/types/owners';

export const load: PageLoad = async ({ fetch }) => {
	try {
		const apiUrl = resolveApiUrl();

		const [prefillRes, ownersRes] = await Promise.all([
			fetch(`${apiUrl}/api/zus/prefill`),
			fetch(`${apiUrl}/api/users`)
		]);
		if (!prefillRes.ok) throw error(prefillRes.status, 'Failed to load ZUS prefill data');

		const prefill = await prefillRes.json();
		const owners: OwnerOption[] = ownersRes.ok ? await ownersRes.json() : [];
		return { prefill, owners };
	} catch (err) {
		if (err instanceof Error && 'status' in err) throw err;
		throw error(500, 'Failed to load ZUS calculator data');
	}
};
