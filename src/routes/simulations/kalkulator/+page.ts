import { browser } from '$app/environment';
import { env } from '$env/dynamic/public';
import type { PageLoad } from './$types';
import type { SalaryRecord } from '$lib/types/salaries';
import type { OwnerOption } from '$lib/types/owners';

export const load: PageLoad = async ({ fetch }) => {
	try {
		const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
		if (!apiUrl) return { latestSalaries: [], owners: [] };

		const [res, ownersRes] = await Promise.all([
			fetch(`${apiUrl}/api/salaries`),
			fetch(`${apiUrl}/api/users`)
		]);
		if (!res.ok) return { latestSalaries: [], owners: [] };

		const data = await res.json();
		const records: SalaryRecord[] = data.salary_records ?? [];
		const owners: OwnerOption[] = ownersRes.ok ? await ownersRes.json() : [];

		// Pick the most recent record per owner
		const byOwner = new Map<number | null, SalaryRecord>();
		for (const r of records) {
			const existing = byOwner.get(r.owner_user_id);
			if (!existing || r.date > existing.date) {
				byOwner.set(r.owner_user_id, r);
			}
		}

		const latestSalaries = [...byOwner.values()].sort((a, b) => {
			if (a.date === b.date) {
				if (a.owner_user_id === b.owner_user_id) return 0;
				return (a.owner_user_id ?? -1) < (b.owner_user_id ?? -1) ? -1 : 1;
			}
			// Newest dates first
			return a.date < b.date ? 1 : -1;
		});

		return { latestSalaries, owners };
	} catch {
		return { latestSalaries: [], owners: [] };
	}
};
