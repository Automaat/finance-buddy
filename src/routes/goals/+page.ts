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
	try {
		// Fetch in parallel; accounts are best-effort (the picker degrades to
		// empty rather than failing the page).
		const [goalsData, accounts] = await Promise.all([
			api.get<GoalsListResponse>('/api/goals', { fetch }),
			api
				.get<{ assets: AccountOption[]; liabilities: AccountOption[] }>('/api/accounts', { fetch })
				.then((d) => [...d.assets, ...d.liabilities])
				.catch(() => [] as AccountOption[])
		]);
		return { ...goalsData, accounts };
	} catch (err) {
		// A backend failure maps to its status; anything else (e.g. the
		// resolveApiUrl misconfig HttpError) rethrows with its actionable message.
		if (err instanceof ApiError) {
			throw error(err.status, 'Failed to load goals');
		}
		throw err;
	}
};
