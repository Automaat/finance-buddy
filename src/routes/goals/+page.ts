import { error } from '@sveltejs/kit';
import { api, ApiError } from '$lib/apiClient';
import type { PageLoad } from './$types';

export interface Goal {
	id: number;
	name: string;
	target_amount: number;
	target_date: string;
	current_amount: number;
	monthly_contribution: number;
	is_completed: boolean;
	account_id: number | null;
	account_name: string | null;
	category: string | null;
	created_at: string;
	progress_percent: number;
	remaining_amount: number;
	projected_hit_date: string | null;
}

export interface GoalsListResponse {
	goals: Goal[];
	total_count: number;
	completed_count: number;
}

export interface AccountOption {
	id: number;
	name: string;
}

export const load: PageLoad = async ({ fetch }) => {
	let goalsData: GoalsListResponse;
	try {
		goalsData = await api.get<GoalsListResponse>('/api/goals', { fetch });
	} catch (err) {
		throw error(err instanceof ApiError ? err.status : 500, 'Failed to load goals');
	}
	// Accounts are best-effort: the picker degrades to empty rather than
	// failing the whole page.
	const accounts = await api
		.get<{ assets: AccountOption[]; liabilities: AccountOption[] }>('/api/accounts', { fetch })
		.then((d) => [...d.assets, ...d.liabilities])
		.catch(() => [] as AccountOption[]);
	return { ...goalsData, accounts };
};
