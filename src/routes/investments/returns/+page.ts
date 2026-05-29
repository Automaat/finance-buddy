import { error } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import { INVESTMENT_CATEGORIES } from '$lib/constants';
import type { PageLoad } from './$types';

export interface ReturnsAccountOption {
	id: number;
	name: string;
	category: string;
	account_wrapper: string | null;
	is_active: boolean;
}

// Investment accounts feed the per-account scope selector. Categories and
// wrappers are fixed sets, so only the account list needs fetching. An active
// account counts as investment-capable if its category is an investment one OR
// it carries a retirement wrapper (IKE/IKZE/PPK accounts may be categorized as
// ppk/other yet still hold investments) — matching how the returns endpoint
// scopes per-account.
export const load: PageLoad = async ({ fetch }) => {
	const apiUrl = resolveApiUrl();
	const res = await fetch(`${apiUrl}/api/accounts`);
	if (!res.ok) {
		throw error(res.status, 'Nie udało się pobrać kont');
	}
	const data: { assets: ReturnsAccountOption[] } = await res.json();
	const accounts = (data.assets ?? [])
		.filter(
			(a) => a.is_active && (INVESTMENT_CATEGORIES.has(a.category) || a.account_wrapper != null)
		)
		.sort((a, b) => a.name.localeCompare(b.name, 'pl'));
	return { accounts };
};
