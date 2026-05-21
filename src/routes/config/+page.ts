import { error } from '@sveltejs/kit';
import { API_URL_NOT_CONFIGURED_MESSAGE, resolveApiUrl } from '$lib/utils/api';
import type { PageLoad } from './$types';
import type { AppConfig } from '$lib/types/config';

export const load: PageLoad = async ({ fetch }) => {
	const apiUrl = resolveApiUrl();
	if (!apiUrl) {
		throw error(500, API_URL_NOT_CONFIGURED_MESSAGE);
	}

	try {
		const response = await fetch(`${apiUrl}/api/config`);

		// Handle 404 - config not initialized, return defaults
		if (response.status === 404) {
			return {
				config: null,
				isFirstTime: true,
				retirementAccountValue: 0
			};
		}

		if (!response.ok) {
			throw error(response.status, 'Failed to load configuration');
		}

		const data: AppConfig = await response.json();

		// Fetch retirement account value from dashboard
		let retirementAccountValue = 0;
		try {
			const dashboardResponse = await fetch(`${apiUrl}/api/dashboard`);
			if (dashboardResponse.ok) {
				const dashboardData = await dashboardResponse.json();
				retirementAccountValue = dashboardData.retirement_account_value || 0;
			}
		} catch {
			// If dashboard fetch fails, default to 0
			retirementAccountValue = 0;
		}

		return {
			config: data,
			isFirstTime: false,
			retirementAccountValue
		};
	} catch (err) {
		if (err && typeof err === 'object' && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to load configuration');
	}
};
