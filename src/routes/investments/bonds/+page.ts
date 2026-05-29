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
	account_id: number | null;
	first_year_rate: number;
	margin: number;
	capitalize: boolean;
	current_value: number;
	current_yield: number;
	created_at: string;
}

export interface AccountOption {
	id: number;
	name: string;
	category: string;
	owner_user_id: number | null;
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

export type LadderEventKind = 'redemption' | 'coupon';

export interface MaturityLadderEvent {
	month: string;
	type: TreasuryBond['type'];
	kind: LadderEventKind;
	bond_ids: number[];
	count: number;
	principal: number;
	interest_gross: number;
	tax: number;
	net_cashflow: number;
}

export interface NextMaturityWarning {
	date: string;
	type: TreasuryBond['type'];
	bond_ids: number[];
	count: number;
	principal: number;
	interest_gross: number;
	tax: number;
	net_cashflow: number;
	days_until: number;
}

export interface MaturityLadderResponse {
	events: MaturityLadderEvent[];
	next_maturity: NextMaturityWarning | null;
	tax_rate_pct: number;
}

export const load: PageLoad = async ({ fetch }) => {
	try {
		const apiUrl = resolveApiUrl();
		const [bondsRes, ownersRes, ladderRes, accountsRes] = await Promise.all([
			fetch(`${apiUrl}/api/bonds`),
			fetch(`${apiUrl}/api/users`),
			fetch(`${apiUrl}/api/bonds/maturity-ladder`),
			fetch(`${apiUrl}/api/accounts`)
		]);
		if (!bondsRes.ok) {
			throw error(bondsRes.status, 'Failed to load treasury bonds');
		}
		const bonds: BondsResponse = await bondsRes.json();
		const owners: OwnerOption[] = ownersRes.ok ? await ownersRes.json() : [];
		const ladder: MaturityLadderResponse = ladderRes.ok
			? await ladderRes.json()
			: { events: [], next_maturity: null, tax_rate_pct: 19 };
		// Filter to bond-category active accounts — those are the only valid
		// targets for a treasury bond row. Anything else (mortgage, stock,
		// real estate) would mis-roll into the wrong dashboard tile.
		let accounts: AccountOption[] = [];
		if (accountsRes.ok) {
			const payload = await accountsRes.json();
			accounts = [...(payload.assets ?? []), ...(payload.liabilities ?? [])]
				.filter(
					(a: { is_active: boolean; category: string }) => a.is_active && a.category === 'bond'
				)
				.map((a: { id: number; name: string; category: string; owner_user_id: number | null }) => ({
					id: a.id,
					name: a.name,
					category: a.category,
					owner_user_id: a.owner_user_id
				}));
		}
		return { ...bonds, owners, ladder, accounts };
	} catch (err) {
		if (err instanceof Error && 'status' in err) throw err;
		throw error(500, 'Failed to load treasury bonds');
	}
};
