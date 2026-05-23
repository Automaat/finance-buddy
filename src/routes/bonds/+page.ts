import { error } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import type { PageLoad } from './$types';

export interface TreasuryBond {
	id: number;
	type: 'EDO' | 'COI' | 'ROR' | 'TOZ' | 'DOS';
	series: string;
	face_value: number;
	purchase_date: string;
	maturity_date: string;
	owner_user_id: number | null;
	first_year_rate: number;
	margin: number;
	capitalize: boolean;
	current_value: number;
	current_yield: number;
	created_at: string;
}

export interface BondsResponse {
	bonds: TreasuryBond[];
	total_value: number;
	total_count: number;
}

export interface OwnerOption {
	id: number;
	name: string;
}

export const load: PageLoad = async ({ fetch }) => {
	try {
		const apiUrl = resolveApiUrl();
		const [bondsRes, ownersRes] = await Promise.all([
			fetch(`${apiUrl}/api/bonds`),
			fetch(`${apiUrl}/api/users`)
		]);
		if (!bondsRes.ok) {
			throw error(bondsRes.status, 'Failed to load treasury bonds');
		}
		const bonds: BondsResponse = await bondsRes.json();
		const owners: OwnerOption[] = ownersRes.ok ? await ownersRes.json() : [];
		return { ...bonds, owners };
	} catch (err) {
		if (err instanceof Error && 'status' in err) throw err;
		throw error(500, 'Failed to load treasury bonds');
	}
};
