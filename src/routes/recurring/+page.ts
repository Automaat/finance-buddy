import { error } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import type { PageLoad } from './$types';

export interface RecurringRow {
	id: number;
	account_id: number;
	amount: string;
	owner_user_id: number | null;
	transaction_type: string | null;
	category: string | null;
	description: string;
	frequency: string;
	day_of_month: number | null;
	start_date: string;
	end_date: string | null;
	active: boolean;
	skipped_dates: string[];
	last_run_date: string | null;
	next_occurrence: string | null;
	created_at: string;
	updated_at: string;
}

export interface AccountOption {
	id: number;
	name: string;
}

export const load: PageLoad = async ({ fetch }) => {
	const apiUrl = resolveApiUrl();
	const [recurringRes, accountsRes] = await Promise.all([
		fetch(`${apiUrl}/api/recurring`),
		fetch(`${apiUrl}/api/accounts`)
	]);
	if (!recurringRes.ok) {
		throw error(recurringRes.status, 'Failed to load recurring transactions');
	}
	if (!accountsRes.ok) {
		throw error(accountsRes.status, 'Failed to load accounts');
	}
	const recurringData = (await recurringRes.json()) as { recurring: RecurringRow[] };
	const accountsData = (await accountsRes.json()) as { accounts: AccountOption[] };
	return {
		recurring: recurringData.recurring,
		accounts: accountsData.accounts.map((a) => ({ id: a.id, name: a.name }))
	};
};
