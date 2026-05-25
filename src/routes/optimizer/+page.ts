import type { PageLoad } from './$types';
import { resolveApiUrl } from '$lib/api';
import { loadConfigDefaults } from '$lib/utils/configDefaults';
import type { OptionKey } from '$lib/utils/allocationOptimizer';

interface YearlyStat {
	account_wrapper: string;
	remaining: number | null;
}

interface DriftItem {
	category: string;
	drift_pp: number;
}

interface DriftScope {
	owner_user_id: number | null;
	items: DriftItem[];
}

interface DriftResponse {
	scopes: DriftScope[];
}

export const load: PageLoad = async ({ fetch }) => {
	const year = new Date().getFullYear();
	const apiUrl = resolveApiUrl();
	const [defaults, statsRes, driftRes] = await Promise.all([
		loadConfigDefaults(fetch),
		fetch(`${apiUrl}/api/retirement/stats?year=${year}`),
		fetch(`${apiUrl}/api/allocation/drift`)
	]);

	let ikeRemainingPLN = 0;
	let ikzeRemainingPLN = 0;
	if (statsRes.ok) {
		const stats = (await statsRes.json()) as YearlyStat[];
		for (const row of stats) {
			const r = row.remaining ?? 0;
			if (row.account_wrapper === 'IKE') ikeRemainingPLN += r;
			else if (row.account_wrapper === 'IKZE') ikzeRemainingPLN += r;
		}
	}

	// Drift maps the dashboard's per-category under/over-target signal onto
	// the optimizer's option keys. Household scope wins; fall back to summing
	// per-owner drifts if no household-level scope exists. ikze/ike/mortgage
	// have no allocation-drift equivalent and stay at zero.
	const allocationDrift: Record<OptionKey, number> = {
		ikze: 0,
		ike: 0,
		mortgage: 0,
		bonds: 0,
		brokerage: 0
	};
	if (driftRes.ok) {
		const drift = (await driftRes.json()) as DriftResponse;
		const household = drift.scopes.find((s) => s.owner_user_id === null);
		const sources = household ? [household] : drift.scopes;
		const sumDrift = (cat: string) =>
			sources.reduce((acc, scope) => {
				const item = scope.items.find((i) => i.category === cat);
				return acc + (item?.drift_pp ?? 0);
			}, 0);
		allocationDrift.bonds = sumDrift('bond');
		allocationDrift.brokerage = sumDrift('stock');
	}

	return { defaults, ikeRemainingPLN, ikzeRemainingPLN, allocationDrift };
};
