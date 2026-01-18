import { error } from '@sveltejs/kit';
import { env } from '$env/dynamic/public';
import { browser } from '$app/environment';
import type { PageLoad } from './$types';
import type { AppConfig } from '$lib/types/config';

export const load: PageLoad = async ({ fetch }) => {
	try {
		const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
		if (!apiUrl) {
			throw error(500, 'API URL is not configured');
		}

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
		if (err instanceof Error && 'status' in err) {
			throw err;
		}
		throw error(500, 'Failed to load configuration');
	}
};
