import { error } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ fetch }) => {
	const apiUrl = resolveApiUrl();

	const [targetsRes, ownersRes] = await Promise.all([
		fetch(`${apiUrl}/api/allocation/targets`),
		fetch(`${apiUrl}/api/users`)
	]);
	if (!targetsRes.ok) {
		throw error(targetsRes.status, 'Nie udało się pobrać celów alokacji');
	}
	const targets = await targetsRes.json();
	const owners = ownersRes.ok ? await ownersRes.json() : [];
	return { targets, owners };
};
