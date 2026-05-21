import { error } from '@sveltejs/kit';
import { API_URL_NOT_CONFIGURED_MESSAGE, resolveApiUrl } from '$lib/utils/api';
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
	const apiUrl = resolveApiUrl();
	if (!apiUrl) {
		throw error(500, API_URL_NOT_CONFIGURED_MESSAGE);
	}
	const [goalsResponse, accountsResponse] = await Promise.all([
		fetch(`${apiUrl}/api/goals`),
		fetch(`${apiUrl}/api/accounts`)
	]);
	if (!goalsResponse.ok) {
		throw error(goalsResponse.status, 'Failed to load goals');
	}
	const goalsData: GoalsListResponse = await goalsResponse.json();
	const accounts: AccountOption[] = accountsResponse.ok
		? await accountsResponse
				.json()
				.then((d: { assets: AccountOption[]; liabilities: AccountOption[] }) => [
					...d.assets,
					...d.liabilities
				])
		: [];
	return { ...goalsData, accounts };
};
