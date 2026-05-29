import { error } from '@sveltejs/kit';
import { resolveApiUrl } from '$lib/api';
import type { CurrencyExposureReport } from '$lib/types/exposure';
import type { PageLoad } from './$types';

// Target PLN share + tolerance are URL-driven so the drift band is shareable
// and survives reloads. A valid target_pln_pct turns on the drift band; an
// absent/invalid one fetches the plain breakdown.
export const load: PageLoad = async ({ fetch, url }) => {
	const apiUrl = resolveApiUrl();

	const targetParamRaw = url.searchParams.get('target_pln_pct');
	const targetNum = targetParamRaw === null ? null : Number(targetParamRaw);
	const hasTarget =
		targetNum !== null && Number.isFinite(targetNum) && targetNum >= 0 && targetNum <= 100;

	const toleranceRaw = url.searchParams.get('tolerance');
	const toleranceNum = toleranceRaw === null ? 5 : Number(toleranceRaw);
	const tolerance = Number.isFinite(toleranceNum) && toleranceNum >= 0 ? toleranceNum : 5;

	const params = new URLSearchParams();
	if (hasTarget) {
		params.set('target_pln_pct', String(targetNum));
		params.set('tolerance', String(tolerance));
	}
	const qs = params.toString();
	const res = await fetch(`${apiUrl}/api/exposure/currency${qs ? `?${qs}` : ''}`);
	if (!res.ok) {
		throw error(res.status, 'Nie udało się pobrać ekspozycji walutowej');
	}
	const report: CurrencyExposureReport = await res.json();

	return {
		report,
		targetPLNPct: hasTarget ? targetNum : null,
		tolerance
	};
};
