import { error } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import type { OwnerOption } from '$lib/types/owners';
import type { PageLoad } from './$types';

// Per-owner PPK summary as served by GET /api/retirement/ppk-stats.
export interface PPKStat {
	owner_user_id: number | null;
	total_value: number;
	employee_contributed: number;
	employer_contributed: number;
	government_contributed: number;
	total_contributed: number;
	returns: number;
	roi_percentage: number;
}

// A single PPK-wrapped account, surfaced from GET /api/accounts.
export interface PPKAccount {
	id: number;
	name: string;
	owner_user_id: number | null;
	is_active: boolean;
	current_value: number;
}

export const load: PageLoad = async ({ fetch }) => {
	try {
		const apiUrl = resolveApiUrl();
		const [statsRes, ownersRes, accountsRes] = await Promise.all([
			fetch(`${apiUrl}/api/retirement/ppk-stats`),
			fetch(`${apiUrl}/api/users`),
			fetch(`${apiUrl}/api/accounts`)
		]);
		if (!statsRes.ok) {
			throw error(statsRes.status, 'Nie udało się pobrać danych PPK');
		}
		const stats: PPKStat[] = await statsRes.json();
		const owners: OwnerOption[] = ownersRes.ok ? await ownersRes.json() : [];

		let accounts: PPKAccount[] = [];
		if (accountsRes.ok) {
			const payload = await accountsRes.json();
			accounts = (payload.assets ?? [])
				.filter((a: { account_wrapper: string | null }) => a.account_wrapper === 'PPK')
				.map(
					(a: {
						id: number;
						name: string;
						owner_user_id: number | null;
						is_active: boolean;
						current_value: number;
					}) => ({
						id: a.id,
						name: a.name,
						owner_user_id: a.owner_user_id,
						is_active: a.is_active,
						current_value: a.current_value
					})
				);
		}

		return { stats, owners, accounts };
	} catch (err) {
		if (err instanceof Error && 'status' in err) throw err;
		throw error(500, 'Nie udało się pobrać danych PPK');
	}
};
