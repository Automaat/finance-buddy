import { browser } from '$app/environment';
import { env } from '$env/dynamic/public';
import type { PageLoad } from './$types';
import type { SalaryRecord } from '$lib/types/salaries';

export const load: PageLoad = async ({ fetch }) => {
	try {
		const apiUrl = browser ? env.PUBLIC_API_URL_BROWSER : env.PUBLIC_API_URL;
		if (!apiUrl) return { latestSalaries: [] };

		const res = await fetch(`${apiUrl}/api/salaries`);
		if (!res.ok) return { latestSalaries: [] };

		const data = await res.json();
		const records: SalaryRecord[] = data.salary_records ?? [];

		// Pick the most recent record per owner
		const byOwner = new Map<string, SalaryRecord>();
		for (const r of records) {
			const existing = byOwner.get(r.owner);
			if (!existing || r.date > existing.date) {
				byOwner.set(r.owner, r);
			}
		}

		return { latestSalaries: [...byOwner.values()] };
	} catch {
		return { latestSalaries: [] };
	}
};
