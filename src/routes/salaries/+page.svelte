<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import * as echarts from 'echarts';
	import type { EChartsOption } from 'echarts';
	import Modal from '$lib/components/Modal.svelte';
	import { formatPLN } from '$lib/utils/format';
	import { buildCpiLookup, inflationAdjust, parseIsoDate } from '$lib/utils/inflation';
	import { grossToNet, type PlContractType } from '$lib/utils/pl_tax';
	import {
		Plus,
		Banknote,
		TrendingUp,
		Search,
		BarChart3,
		Pencil,
		Trash2,
		Scale,
		Gift,
		Award,
		Building2,
		Wallet
	} from 'lucide-svelte';
	import { env } from '$env/dynamic/public';
	import { goto, invalidateAll } from '$app/navigation';
	import { toast } from '$lib/stores/toast.svelte';
	import type {
		BonusEvent,
		BonusType,
		CompanyValuation,
		CustomVestingEvent,
		EquityGrant,
		EquityGrantType,
		EquityTaxTreatment,
		SalaryRecord,
		ValuationSource,
		VestingFrequency
	} from '$lib/types/salaries';
	import type { CpiSeries } from '$lib/types/cpi';
	import { type OwnerOption, ownerName } from '$lib/types/owners';
	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	const apiUrl = env.PUBLIC_API_URL_BROWSER || 'http://localhost:8000';
	const owners = $derived(data.owners as OwnerOption[]);
	const defaultOwnerId = $derived<number | null>(owners.length > 0 ? owners[0].id : null);
	const cpiSeries = $derived(data.cpiSeries as CpiSeries);
	const inflationContext = $derived(data.salaries.inflation_context ?? {});
	const inflationEntries = $derived(Object.values(inflationContext));

	let showNominal = $state(true);
	let showReal = $state(false);
	let showInflationTracked = $state(false);

	const monthNamesPL = [
		'styczeń',
		'luty',
		'marzec',
		'kwiecień',
		'maj',
		'czerwiec',
		'lipiec',
		'sierpień',
		'wrzesień',
		'październik',
		'listopad',
		'grudzień'
	];

	function isNonNegative(value: number | null): boolean {
		return (value ?? 0) >= 0;
	}

	function formatPctSigned(value: number | null): string {
		if (value == null || Number.isNaN(value)) return '—';
		const sign = value >= 0 ? '+' : '';
		return `${sign}${value.toFixed(1)}%`;
	}

	function formatPlnSigned(value: number | null): string {
		if (value == null || Number.isNaN(value)) return '—';
		const sign = value >= 0 ? '+' : '−';
		return `${sign}${formatPLN(Math.abs(value))}`;
	}

	let chartContainer: HTMLDivElement;
	let chart: echarts.ECharts | undefined;

	let filterOwnerUserId = $state<number | null>(
		untrack(() => (data.filters.owner_user_id ? Number(data.filters.owner_user_id) : null))
	);
	let filterDateFrom = $state(untrack(() => data.filters.date_from || ''));
	let filterDateTo = $state(untrack(() => data.filters.date_to || ''));
	let filterCompany = $state(untrack(() => data.filters.company || ''));

	let showNewSalaryModal = $state(false);
	let editingSalary: SalaryRecord | null = $state(null);
	let salaryFormData = $state({
		date: new Date().toISOString().split('T')[0],
		gross_amount: 0,
		contract_type: 'UOP',
		company: '',
		owner_user_id: untrack(() => defaultOwnerId)
	});
	let salaryError = $state('');
	let savingSalary = $state(false);

	const currentYear = new Date().getFullYear();

	const latestContractByOwner = $derived.by(() => {
		const map = new Map<number | null, string>();
		for (const r of data.salaries.salary_records) {
			if (!map.has(r.owner_user_id)) map.set(r.owner_user_id, r.contract_type);
		}
		return map;
	});

	function netMonthlyForOwner(ownerUserId: number | null, grossMonthly: number): number | null {
		const ct = latestContractByOwner.get(ownerUserId);
		if (!ct) return null;
		const allowed: PlContractType[] = ['UOP', 'B2B', 'UZ', 'UoD'];
		if (!allowed.includes(ct as PlContractType)) return null;
		const breakdown = grossToNet(grossMonthly, ct as PlContractType, currentYear);
		return breakdown.netAnnual / 12;
	}

	type CurrentSalaryRow = {
		name: string;
		salary: number | null;
		net: number | null;
	};

	const currentSalaryRows = $derived.by<CurrentSalaryRow[]>(() =>
		Object.entries(data.salaries.current_salaries).map(([key, salary]) => {
			const ownerUserId = Number(key);
			return {
				name: ownerName(owners, ownerUserId),
				salary,
				net: salary !== null ? netMonthlyForOwner(ownerUserId, salary) : null
			};
		})
	);

	const allYears = $derived.by(() => {
		const years = new Set<number>();
		years.add(currentYear);
		for (const r of data.salaries.salary_records) years.add(new Date(r.date).getFullYear());
		for (const b of data.bonuses?.bonus_events ?? []) years.add(new Date(b.date).getFullYear());
		return [...years].sort((a, b) => b - a);
	});

	let totalCompYear = $state(new Date().getFullYear());
	let totalCompOwner = $state<number | null>(untrack(() => defaultOwnerId));
	let includeEquityInTotal = $state(false);

	type OwnerCompSummary = {
		ownerUserId: number | null;
		baseAnnualGross: number;
		baseAnnualNet: number;
		bonusesPln: number;
		bonusesNetPln: number;
		equityPaperPln: number;
		equityPaperLowPln: number;
		equityPaperHighPln: number;
		equityNetPln: number;
		hasEquityWithoutFx: boolean;
	};

	// Capital gains rate for equity sold under art. 24 ust. 11 PIT — the default
	// tax treatment for foreign-parent ESOPs. Applied to paper value for the
	// "net" estimate only; real net depends on actual sale and tax_treatment.
	const EQUITY_CAPITAL_GAINS_RATE = 0.19;

	function pad2(n: number): string {
		return n.toString().padStart(2, '0');
	}

	function isoDateLocal(d: Date): string {
		// toISOString() converts to UTC and shifts the day for TZ > UTC (e.g. PL
		// in winter: 2026-12-31 00:00 local → 2026-12-30T23:00Z). Build YYYY-MM-DD
		// from the local components to keep comparisons consistent with date-only
		// values returned by the API.
		return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`;
	}

	const compSummary = $derived.by<OwnerCompSummary | null>(() => {
		if (totalCompOwner === null) return null;
		const ownerUserId = totalCompOwner;
		const yearEndIso = isoDateLocal(new Date(totalCompYear, 11, 31));

		const latestSalary = data.salaries.salary_records.find(
			(r) => r.owner_user_id === ownerUserId && r.date <= yearEndIso
		);
		const baseMonthly = latestSalary?.gross_amount ?? 0;
		const baseAnnualGross = baseMonthly * 12;

		const ct = latestSalary?.contract_type;
		const allowed: PlContractType[] = ['UOP', 'B2B', 'UZ', 'UoD'];
		const useTaxCalc = ct && allowed.includes(ct as PlContractType);

		const baseAnnualNet = useTaxCalc
			? grossToNet(baseMonthly, ct as PlContractType, totalCompYear).netAnnual
			: 0;

		const bonusesPln = (data.bonuses?.bonus_events ?? [])
			.filter(
				(b) => b.owner_user_id === ownerUserId && new Date(b.date).getFullYear() === totalCompYear
			)
			.reduce((s, b) => s + (b.amount_pln ?? (b.currency === 'PLN' ? b.amount : 0)), 0);

		// Net for bonuses: treat as additional gross in the same year and take the
		// marginal delta from gross_to_net. This applies progressive PIT + ZUS
		// correctly when bonuses push the year over the 120k threshold.
		const bonusesNetPln = useTaxCalc
			? grossToNet((baseAnnualGross + bonusesPln) / 12, ct as PlContractType, totalCompYear)
					.netAnnual - baseAnnualNet
			: 0;

		let equityPaperPln = 0;
		let equityPaperLowPln = 0;
		let equityPaperHighPln = 0;
		let hasEquityWithoutFx = false;
		for (const g of data.equity?.equity_grants ?? []) {
			if (g.owner_user_id !== ownerUserId) continue;
			if (g.paper_value_base_pln === null && g.paper_value_base !== null) {
				hasEquityWithoutFx = true;
				continue;
			}
			if (g.paper_value_base_pln !== null) {
				equityPaperPln += g.paper_value_base_pln;
				equityPaperLowPln += g.paper_value_low_pln ?? g.paper_value_base_pln;
				equityPaperHighPln += g.paper_value_high_pln ?? g.paper_value_base_pln;
			}
		}
		// Equity "net" assumes capital-gains treatment on realization; grants on
		// employment_income would be ~12/32% + ZUS instead. Rough estimate only.
		const equityNetPln = equityPaperPln * (1 - EQUITY_CAPITAL_GAINS_RATE);

		return {
			ownerUserId,
			baseAnnualGross,
			baseAnnualNet,
			bonusesPln,
			bonusesNetPln,
			equityPaperPln,
			equityPaperLowPln,
			equityPaperHighPln,
			equityNetPln,
			hasEquityWithoutFx
		};
	});

	// Computed in script — prettier-plugin-svelte 4.0.0 chokes on chained `+`
	// BinaryExpressions inside {@const} blocks.
	const totalCompGross = $derived(
		(compSummary?.baseAnnualGross ?? 0) +
			(compSummary?.bonusesPln ?? 0) +
			(includeEquityInTotal ? (compSummary?.equityPaperPln ?? 0) : 0)
	);
	const totalCompNet = $derived(
		(compSummary?.baseAnnualNet ?? 0) +
			(compSummary?.bonusesNetPln ?? 0) +
			(includeEquityInTotal ? (compSummary?.equityNetPln ?? 0) : 0)
	);

	const bonusEvents = $derived(data.bonuses?.bonus_events ?? []);
	const bonusGroupedByCompany = $derived(
		bonusEvents.reduce<Map<string, BonusEvent[]>>((acc, b) => {
			const key = b.company || 'Nieokreślona firma';
			if (!acc.has(key)) acc.set(key, []);
			acc.get(key)!.push(b);
			return acc;
		}, new Map())
	);

	const bonusTypeLabels: Record<BonusType, string> = {
		annual: 'Roczny',
		signon: 'Powitalny',
		spot: 'Uznaniowy',
		retention: 'Retencyjny'
	};

	let showBonusModal = $state(false);
	let editingBonus: BonusEvent | null = $state(null);
	let bonusFormData = $state({
		date: new Date().toISOString().split('T')[0],
		amount: 0,
		currency: 'PLN',
		type: 'annual' as BonusType,
		company: '',
		owner_user_id: untrack(() => defaultOwnerId),
		contract_type: 'UOP',
		notes: ''
	});
	let bonusError = $state('');
	let savingBonus = $state(false);

	function formatBonusAmount(amount: number, currency: string): string {
		if (currency === 'PLN') return formatPLN(amount);
		return `${amount.toLocaleString('pl-PL', { maximumFractionDigits: 2 })} ${currency}`;
	}

	function openNewBonusModal() {
		editingBonus = null;
		bonusFormData = {
			date: new Date().toISOString().split('T')[0],
			amount: 0,
			currency: 'PLN',
			type: 'annual',
			company: '',
			owner_user_id: defaultOwnerId,
			contract_type: 'UOP',
			notes: ''
		};
		bonusError = '';
		showBonusModal = true;
	}

	function openEditBonusModal(bonus: BonusEvent) {
		editingBonus = bonus;
		bonusFormData = {
			date: bonus.date,
			amount: bonus.amount,
			currency: bonus.currency,
			type: bonus.type,
			company: bonus.company,
			owner_user_id: bonus.owner_user_id,
			contract_type: bonus.contract_type,
			notes: bonus.notes ?? ''
		};
		bonusError = '';
		showBonusModal = true;
	}

	function closeBonusModal() {
		showBonusModal = false;
		editingBonus = null;
		bonusError = '';
	}

	async function saveBonus() {
		if (!bonusFormData.date) {
			bonusError = 'Data jest wymagana';
			return;
		}
		const todayNow = new Date().toISOString().split('T')[0];
		if (bonusFormData.date > todayNow) {
			bonusError = 'Data nie może być z przyszłości';
			return;
		}
		if (!bonusFormData.amount || bonusFormData.amount <= 0) {
			bonusError = 'Kwota musi być większa niż 0';
			return;
		}
		if (!bonusFormData.company || !bonusFormData.company.trim()) {
			bonusError = 'Firma nie może być pusta';
			return;
		}

		bonusError = '';
		savingBonus = true;

		try {
			const method = editingBonus ? 'PATCH' : 'POST';
			const url = editingBonus
				? `${apiUrl}/api/bonuses/${editingBonus.id}`
				: `${apiUrl}/api/bonuses`;

			const payload = {
				...bonusFormData,
				notes: bonusFormData.notes.trim() || null
			};

			const response = await fetch(url, {
				method,
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload)
			});

			if (!response.ok) {
				const errorData = await response.json();
				const detail = errorData.detail;
				const fallback = 'Failed to save bonus';
				let message: string;
				if (Array.isArray(detail)) {
					const joined = detail
						.map((d: { msg?: string }) => (typeof d?.msg === 'string' ? d.msg : ''))
						.filter(Boolean)
						.join('; ');
					message = joined || fallback;
				} else if (typeof detail === 'string' && detail) {
					message = detail;
				} else {
					message = fallback;
				}
				throw new Error(message);
			}

			await invalidateAll();
			closeBonusModal();
		} catch (err) {
			if (err instanceof Error) {
				bonusError = err.message;
			}
		} finally {
			savingBonus = false;
		}
	}

	async function deleteBonus(id: number) {
		if (!confirm('Czy na pewno chcesz usunąć ten bonus?')) {
			return;
		}

		try {
			const response = await fetch(`${apiUrl}/api/bonuses/${id}`, { method: 'DELETE' });
			if (!response.ok) {
				throw new Error('Failed to delete bonus');
			}
			await invalidateAll();
		} catch (err) {
			console.error('Failed to delete bonus:', err);
			toast.error('Nie udało się usunąć bonusu');
		}
	}

	const equityGrants = $derived(data.equity?.equity_grants ?? []);

	type EquityGroup = {
		company: string;
		grants: EquityGrant[];
		grantLabel: string;
		totalShares: number;
		vestedShares: number;
		paperBase: number;
		paperBasePln: number;
		currency: string;
		hasPaperValue: boolean;
		hasPaperValuePln: boolean;
	};

	// Per-company aggregates pre-computed here — keeping these out of {@const}
	// in the template avoids a prettier-plugin-svelte 4.0.0 crash on
	// BinaryExpression inside @const initializers (chained `+`, reduce bodies).
	const equityGroups = $derived.by<EquityGroup[]>(() => {
		const byCompany = new Map<string, EquityGrant[]>();
		for (const g of equityGrants) {
			const key = g.company || 'Nieokreślona firma';
			if (!byCompany.has(key)) byCompany.set(key, []);
			byCompany.get(key)!.push(g);
		}
		return [...byCompany.entries()].map(([company, grants]) => {
			let totalShares = 0;
			let vestedShares = 0;
			let paperBase = 0;
			let paperBasePln = 0;
			let currency: string | null = null;
			for (const g of grants) {
				totalShares += g.total_shares;
				vestedShares += g.vested_shares_today;
				paperBase += g.paper_value_base ?? 0;
				paperBasePln += g.paper_value_base_pln ?? 0;
				if (currency === null && g.paper_value_currency) {
					currency = g.paper_value_currency;
				}
			}
			const safeCurrency = currency ?? '';
			const hasPaperValue = safeCurrency !== '' && paperBase > 0;
			const hasPaperValuePln = hasPaperValue && safeCurrency !== 'PLN' && paperBasePln > 0;
			const grantLabel = grants.length === 1 ? 'grant' : 'grantów';
			return {
				company,
				grants,
				grantLabel,
				totalShares,
				vestedShares,
				paperBase,
				paperBasePln,
				currency: safeCurrency,
				hasPaperValue,
				hasPaperValuePln
			};
		});
	});

	const equityTypeLabels: Record<EquityGrantType, string> = {
		option: 'Opcje',
		rsu: 'RSU'
	};

	const vestingFrequencyLabels: Record<VestingFrequency, string> = {
		monthly: 'miesięczny',
		quarterly: 'kwartalny',
		yearly: 'roczny'
	};

	const taxTreatmentLabels: Record<EquityTaxTreatment, string> = {
		capital_gains_19: 'Kapitałowy 19% (art. 24 ust. 11)',
		employment_income: 'Przychód ze stosunku pracy (12/32%)'
	};

	type VestingPresetKey = 'standard_4_1_monthly' | '4_1_quarterly' | '3_0_monthly' | 'custom';

	const vestingPresets: Record<
		VestingPresetKey,
		{
			label: string;
			cliff: number;
			total: number;
			frequency: VestingFrequency;
			custom: CustomVestingEvent[] | null;
		}
	> = {
		standard_4_1_monthly: {
			label: '4 lata / 1 rok cliff / miesięczny',
			cliff: 12,
			total: 48,
			frequency: 'monthly',
			custom: null
		},
		'4_1_quarterly': {
			label: '4 lata / 1 rok cliff / kwartalny',
			cliff: 12,
			total: 48,
			frequency: 'quarterly',
			custom: null
		},
		'3_0_monthly': {
			label: '3 lata / bez cliffu / miesięczny',
			cliff: 0,
			total: 36,
			frequency: 'monthly',
			custom: null
		},
		custom: {
			label: 'Niestandardowy',
			cliff: 12,
			total: 48,
			frequency: 'yearly',
			custom: [
				{ month: 12, pct: 10 },
				{ month: 24, pct: 20 },
				{ month: 36, pct: 30 },
				{ month: 48, pct: 40 }
			]
		}
	};

	let showEquityModal = $state(false);
	let editingGrant: EquityGrant | null = $state(null);
	let equityFormData = $state({
		grant_date: new Date().toISOString().split('T')[0],
		type: 'rsu' as EquityGrantType,
		company: '',
		owner_user_id: untrack(() => defaultOwnerId),
		total_shares: 0,
		strike_price: null as number | null,
		currency: 'USD',
		vest_start_date: new Date().toISOString().split('T')[0],
		vest_cliff_months: 12,
		vest_total_months: 48,
		vest_frequency: 'monthly' as VestingFrequency,
		preset: 'standard_4_1_monthly' as VestingPresetKey,
		vest_custom_schedule: null as CustomVestingEvent[] | null,
		requires_liquidity_event: false,
		liquidity_event_date: null as string | null,
		tax_treatment: 'capital_gains_19' as EquityTaxTreatment,
		notes: ''
	});
	let equityError = $state('');
	let savingEquity = $state(false);

	function applyPreset(key: VestingPresetKey) {
		const preset = vestingPresets[key];
		equityFormData.vest_cliff_months = preset.cliff;
		equityFormData.vest_total_months = preset.total;
		equityFormData.vest_frequency = preset.frequency;
		equityFormData.vest_custom_schedule = preset.custom ? [...preset.custom] : null;
	}

	function formatShares(n: number): string {
		return n.toLocaleString('pl-PL', { maximumFractionDigits: 0 });
	}

	function formatCurrency(amount: number, currency: string): string {
		if (currency === 'PLN') return formatPLN(amount);
		return `${amount.toLocaleString('pl-PL', { maximumFractionDigits: 2 })} ${currency}`;
	}

	function openNewEquityModal() {
		editingGrant = null;
		equityFormData = {
			grant_date: new Date().toISOString().split('T')[0],
			type: 'rsu',
			company: '',
			owner_user_id: defaultOwnerId,
			total_shares: 0,
			strike_price: null,
			currency: 'USD',
			vest_start_date: new Date().toISOString().split('T')[0],
			vest_cliff_months: 12,
			vest_total_months: 48,
			vest_frequency: 'monthly',
			preset: 'standard_4_1_monthly',
			vest_custom_schedule: null,
			requires_liquidity_event: false,
			liquidity_event_date: null,
			tax_treatment: 'capital_gains_19',
			notes: ''
		};
		equityError = '';
		showEquityModal = true;
	}

	function detectPreset(grant: EquityGrant): VestingPresetKey {
		if (grant.vest_custom_schedule) return 'custom';
		if (
			grant.vest_cliff_months === 12 &&
			grant.vest_total_months === 48 &&
			grant.vest_frequency === 'monthly'
		)
			return 'standard_4_1_monthly';
		if (
			grant.vest_cliff_months === 12 &&
			grant.vest_total_months === 48 &&
			grant.vest_frequency === 'quarterly'
		)
			return '4_1_quarterly';
		if (
			grant.vest_cliff_months === 0 &&
			grant.vest_total_months === 36 &&
			grant.vest_frequency === 'monthly'
		)
			return '3_0_monthly';
		return 'custom';
	}

	function openEditEquityModal(grant: EquityGrant) {
		editingGrant = grant;
		equityFormData = {
			grant_date: grant.grant_date,
			type: grant.type,
			company: grant.company,
			owner_user_id: grant.owner_user_id,
			total_shares: grant.total_shares,
			strike_price: grant.strike_price,
			currency: grant.currency,
			vest_start_date: grant.vest_start_date,
			vest_cliff_months: grant.vest_cliff_months,
			vest_total_months: grant.vest_total_months,
			vest_frequency: grant.vest_frequency,
			preset: detectPreset(grant),
			vest_custom_schedule: grant.vest_custom_schedule ? [...grant.vest_custom_schedule] : null,
			requires_liquidity_event: grant.requires_liquidity_event,
			liquidity_event_date: grant.liquidity_event_date,
			tax_treatment: grant.tax_treatment,
			notes: grant.notes ?? ''
		};
		equityError = '';
		showEquityModal = true;
	}

	function closeEquityModal() {
		showEquityModal = false;
		editingGrant = null;
		equityError = '';
	}

	async function saveEquityGrant() {
		if (!equityFormData.company.trim()) {
			equityError = 'Firma nie może być pusta';
			return;
		}
		if (equityFormData.total_shares <= 0) {
			equityError = 'Liczba akcji musi być większa niż 0';
			return;
		}
		if (equityFormData.type === 'option' && (equityFormData.strike_price ?? 0) <= 0) {
			equityError = 'Opcje wymagają ceny wykonania (strike price)';
			return;
		}
		if (equityFormData.vest_cliff_months > equityFormData.vest_total_months) {
			equityError = 'Cliff nie może przekraczać całkowitego okresu vestingu';
			return;
		}

		equityError = '';
		savingEquity = true;

		try {
			const method = editingGrant ? 'PATCH' : 'POST';
			const url = editingGrant
				? `${apiUrl}/api/equity-grants/${editingGrant.id}`
				: `${apiUrl}/api/equity-grants`;

			const payload = {
				grant_date: equityFormData.grant_date,
				type: equityFormData.type,
				company: equityFormData.company,
				owner_user_id: equityFormData.owner_user_id,
				total_shares: equityFormData.total_shares,
				strike_price: equityFormData.type === 'option' ? equityFormData.strike_price : null,
				currency: equityFormData.currency,
				vest_start_date: equityFormData.vest_start_date,
				vest_cliff_months: equityFormData.vest_cliff_months,
				vest_total_months: equityFormData.vest_total_months,
				vest_frequency: equityFormData.vest_frequency,
				vest_custom_schedule:
					equityFormData.preset === 'custom' ? equityFormData.vest_custom_schedule : null,
				requires_liquidity_event: equityFormData.requires_liquidity_event,
				liquidity_event_date: equityFormData.liquidity_event_date || null,
				tax_treatment: equityFormData.tax_treatment,
				notes: equityFormData.notes.trim() || null
			};

			const response = await fetch(url, {
				method,
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload)
			});

			if (!response.ok) {
				const errorData = await response.json();
				const detail = errorData.detail;
				const fallback = 'Failed to save grant';
				let message: string;
				if (Array.isArray(detail)) {
					const joined = detail
						.map((d: { msg?: string }) => (typeof d?.msg === 'string' ? d.msg : ''))
						.filter(Boolean)
						.join('; ');
					message = joined || fallback;
				} else if (typeof detail === 'string' && detail) {
					message = detail;
				} else {
					message = fallback;
				}
				throw new Error(message);
			}

			await invalidateAll();
			closeEquityModal();
		} catch (err) {
			if (err instanceof Error) {
				equityError = err.message;
			}
		} finally {
			savingEquity = false;
		}
	}

	async function deleteEquityGrant(id: number) {
		if (!confirm('Czy na pewno chcesz usunąć ten grant?')) return;
		try {
			const response = await fetch(`${apiUrl}/api/equity-grants/${id}`, { method: 'DELETE' });
			if (!response.ok) throw new Error('Failed to delete grant');
			await invalidateAll();
		} catch (err) {
			console.error('Failed to delete grant:', err);
			toast.error('Nie udało się usunąć grantu');
		}
	}

	const valuations = $derived(data.valuations?.company_valuations ?? []);

	const valuationSourceLabels: Record<ValuationSource, string> = {
		'409a': '409A',
		preferred_round: 'Runda preferred',
		tender: 'Tender / wykup',
		estimate: 'Estymacja'
	};

	let showValuationModal = $state(false);
	let editingValuation: CompanyValuation | null = $state(null);
	let valuationFormData = $state({
		company: '',
		date: new Date().toISOString().split('T')[0],
		currency: 'USD',
		fmv_per_share: 0,
		fmv_low: null as number | null,
		fmv_high: null as number | null,
		source: '409a' as ValuationSource,
		common_stock_discount_pct: null as number | null,
		notes: ''
	});
	let valuationError = $state('');
	let savingValuation = $state(false);

	function formatRange(grant: EquityGrant): string {
		if (grant.paper_value_base === null) {
			if (grant.valuation_date) return 'brak FX';
			return '—';
		}
		const currency = grant.paper_value_currency ?? grant.currency;
		const base = formatCurrency(grant.paper_value_base, currency);
		if (
			grant.paper_value_low !== null &&
			grant.paper_value_high !== null &&
			grant.paper_value_low !== grant.paper_value_base
		) {
			return `${base} (${formatCurrency(grant.paper_value_low, currency)}–${formatCurrency(grant.paper_value_high, currency)})`;
		}
		return base;
	}

	function openNewValuationModal() {
		editingValuation = null;
		valuationFormData = {
			company: '',
			date: new Date().toISOString().split('T')[0],
			currency: 'USD',
			fmv_per_share: 0,
			fmv_low: null,
			fmv_high: null,
			source: '409a',
			common_stock_discount_pct: null,
			notes: ''
		};
		valuationError = '';
		showValuationModal = true;
	}

	function openEditValuationModal(valuation: CompanyValuation) {
		editingValuation = valuation;
		valuationFormData = {
			company: valuation.company,
			date: valuation.date,
			currency: valuation.currency,
			fmv_per_share: valuation.fmv_per_share,
			fmv_low: valuation.fmv_low,
			fmv_high: valuation.fmv_high,
			source: valuation.source,
			common_stock_discount_pct: valuation.common_stock_discount_pct,
			notes: valuation.notes ?? ''
		};
		valuationError = '';
		showValuationModal = true;
	}

	function closeValuationModal() {
		showValuationModal = false;
		editingValuation = null;
		valuationError = '';
	}

	async function saveValuation() {
		if (!valuationFormData.company.trim()) {
			valuationError = 'Firma nie może być pusta';
			return;
		}
		if (valuationFormData.fmv_per_share < 0) {
			valuationError = 'FMV musi być nieujemne';
			return;
		}
		if (
			valuationFormData.fmv_low !== null &&
			valuationFormData.fmv_low > valuationFormData.fmv_per_share
		) {
			valuationError = 'fmv_low nie może być większe niż fmv_per_share';
			return;
		}
		if (
			valuationFormData.fmv_high !== null &&
			valuationFormData.fmv_high < valuationFormData.fmv_per_share
		) {
			valuationError = 'fmv_high nie może być mniejsze niż fmv_per_share';
			return;
		}

		valuationError = '';
		savingValuation = true;

		try {
			const method = editingValuation ? 'PATCH' : 'POST';
			const url = editingValuation
				? `${apiUrl}/api/company-valuations/${editingValuation.id}`
				: `${apiUrl}/api/company-valuations`;

			const payload = {
				...valuationFormData,
				notes: valuationFormData.notes.trim() || null
			};

			const response = await fetch(url, {
				method,
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload)
			});

			if (!response.ok) {
				const errorData = await response.json();
				const detail = errorData.detail;
				const fallback = 'Failed to save valuation';
				let message: string;
				if (Array.isArray(detail)) {
					const joined = detail
						.map((d: { msg?: string }) => (typeof d?.msg === 'string' ? d.msg : ''))
						.filter(Boolean)
						.join('; ');
					message = joined || fallback;
				} else if (typeof detail === 'string' && detail) {
					message = detail;
				} else {
					message = fallback;
				}
				throw new Error(message);
			}

			await invalidateAll();
			closeValuationModal();
		} catch (err) {
			if (err instanceof Error) valuationError = err.message;
		} finally {
			savingValuation = false;
		}
	}

	async function deleteValuation(id: number) {
		if (!confirm('Czy na pewno chcesz usunąć tę wycenę?')) return;
		try {
			const response = await fetch(`${apiUrl}/api/company-valuations/${id}`, {
				method: 'DELETE'
			});
			if (!response.ok) throw new Error('Failed to delete valuation');
			await invalidateAll();
		} catch (err) {
			console.error('Failed to delete valuation:', err);
			toast.error('Nie udało się usunąć wyceny');
		}
	}

	function getPreviousCompany(ownerUserId: number | null, date: string | null): string | null {
		if (!date) return null;
		return (
			data.salaries.salary_records.find((r) => r.owner_user_id === ownerUserId && r.date === date)
				?.company ?? null
		);
	}

	function applyFilters() {
		const params = new URLSearchParams();
		if (filterOwnerUserId !== null) params.set('owner_user_id', String(filterOwnerUserId));
		if (filterDateFrom) params.set('date_from', filterDateFrom);
		if (filterDateTo) params.set('date_to', filterDateTo);
		if (filterCompany) params.set('company', filterCompany);

		goto(`/salaries?${params.toString()}`);
	}

	function clearFilters() {
		filterOwnerUserId = null;
		filterDateFrom = '';
		filterDateTo = '';
		filterCompany = '';
		goto('/salaries');
	}

	function openNewSalaryModal() {
		editingSalary = null;
		salaryFormData = {
			date: new Date().toISOString().split('T')[0],
			gross_amount: 0,
			contract_type: 'UOP',
			company: '',
			owner_user_id: defaultOwnerId
		};
		salaryError = '';
		showNewSalaryModal = true;
	}

	function openEditSalaryModal(record: SalaryRecord) {
		editingSalary = record;
		salaryFormData = {
			date: record.date,
			gross_amount: record.gross_amount,
			contract_type: record.contract_type,
			company: record.company,
			owner_user_id: record.owner_user_id
		};
		salaryError = '';
		showNewSalaryModal = true;
	}

	function closeSalaryModal() {
		showNewSalaryModal = false;
		editingSalary = null;
		salaryError = '';
	}

	const today = $derived(new Date().toISOString().split('T')[0]);

	async function saveSalary() {
		if (!salaryFormData.date) {
			salaryError = 'Data jest wymagana';
			return;
		}

		const todayNow = new Date().toISOString().split('T')[0];
		if (salaryFormData.date > todayNow) {
			salaryError = 'Data nie może być z przyszłości';
			return;
		}

		if (!salaryFormData.gross_amount || salaryFormData.gross_amount <= 0) {
			salaryError = 'Pensja brutto musi być większa niż 0';
			return;
		}

		if (!salaryFormData.company || !salaryFormData.company.trim()) {
			salaryError = 'Firma nie może być pusta';
			return;
		}

		salaryError = '';
		savingSalary = true;

		try {
			const method = editingSalary ? 'PATCH' : 'POST';
			const url = editingSalary
				? `${apiUrl}/api/salaries/${editingSalary.id}`
				: `${apiUrl}/api/salaries`;

			const response = await fetch(url, {
				method,
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(salaryFormData)
			});

			if (!response.ok) {
				const errorData = await response.json();
				const detail = errorData.detail;
				const fallback = 'Failed to save salary record';
				let message: string;
				if (Array.isArray(detail)) {
					const joined = detail
						.map((d: { msg?: string }) => (typeof d?.msg === 'string' ? d.msg : ''))
						.filter(Boolean)
						.join('; ');
					message = joined || fallback;
				} else if (typeof detail === 'string' && detail) {
					message = detail;
				} else {
					message = fallback;
				}
				throw new Error(message);
			}

			await invalidateAll();
			closeSalaryModal();
		} catch (err) {
			if (err instanceof Error) {
				salaryError = err.message;
			}
		} finally {
			savingSalary = false;
		}
	}

	async function deleteSalary(id: number) {
		if (!confirm('Czy na pewno chcesz usunąć ten rekord wynagrodzenia?')) {
			return;
		}

		try {
			const response = await fetch(`${apiUrl}/api/salaries/${id}`, {
				method: 'DELETE'
			});

			if (!response.ok) {
				throw new Error('Failed to delete salary record');
			}

			await invalidateAll();
		} catch (err) {
			console.error('Failed to delete salary record:', err);
			toast.error('Nie udało się usunąć rekordu wynagrodzenia');
		}
	}

	type LineSeries = {
		name: string;
		data: Array<[string, number]>;
		type: 'line';
		smooth: boolean;
		lineStyle: {
			color: string;
			width: number;
			type?: 'dashed' | 'solid' | 'dotted';
			opacity?: number;
		};
		itemStyle?: { color: string };
	};

	function buildSeries(): LineSeries[] {
		const companyMap = new Map<string, Array<[string, number]>>();

		data.salaries.salary_records.forEach((r) => {
			const companyName = (r.company ?? '').trim() || 'Nieokreślona firma';
			if (!companyMap.has(companyName)) companyMap.set(companyName, []);
			companyMap.get(companyName)!.push([r.date, r.gross_amount]);
		});

		companyMap.forEach((rows) =>
			rows.sort((a, b) => new Date(a[0]).getTime() - new Date(b[0]).getTime())
		);

		const colors = ['#5E81AC', '#88C0D0', '#A3BE8C', '#EBCB8B', '#D08770', '#B48EAD', '#BF616A'];
		// Date-only `today` matches the backend (which is also date-only).
		const now = new Date();
		const todayDate = new Date(now.getFullYear(), now.getMonth(), now.getDate());
		const cpiLookup = buildCpiLookup(cpiSeries);
		const hasCpi = cpiLookup !== null;

		const series: LineSeries[] = [];
		let colorIndex = 0;

		companyMap.forEach((salaryData, company) => {
			const color = colors[colorIndex % colors.length];
			colorIndex++;

			if (showNominal) {
				series.push({
					name: company,
					data: salaryData,
					type: 'line',
					smooth: true,
					lineStyle: { color, width: 2 },
					itemStyle: { color }
				});
			}

			if (hasCpi && showReal) {
				const realData: Array<[string, number]> = [];
				for (const [dateStr, nominal] of salaryData) {
					const adjusted = inflationAdjust(nominal, parseIsoDate(dateStr), todayDate, cpiLookup);
					if (adjusted != null) realData.push([dateStr, adjusted]);
				}
				if (realData.length > 0) {
					series.push({
						name: `${company} (realna wartość)`,
						data: realData,
						type: 'line',
						smooth: true,
						lineStyle: { color, width: 2, type: 'dashed', opacity: 0.7 },
						itemStyle: { color }
					});
				}
			}

			if (hasCpi && showInflationTracked && salaryData.length > 0) {
				const [firstDateStr, firstAmount] = salaryData[0];
				const firstDate = parseIsoDate(firstDateStr);
				const trackedData: Array<[string, number]> = [];
				for (const [dateStr] of salaryData) {
					const projected = inflationAdjust(
						firstAmount,
						firstDate,
						parseIsoDate(dateStr),
						cpiLookup
					);
					if (projected != null) trackedData.push([dateStr, projected]);
				}
				if (trackedData.length > 0) {
					series.push({
						name: `${company} (indeksowana inflacją)`,
						data: trackedData,
						type: 'line',
						smooth: true,
						lineStyle: { color, width: 2, type: 'dotted', opacity: 0.8 },
						itemStyle: { color }
					});
				}
			}
		});

		return series;
	}

	function applyChart() {
		if (!chart) return;
		const series = buildSeries();
		const option: EChartsOption = {
			title: { text: 'Progresja wynagrodzenia', left: 'center', top: 8 },
			tooltip: {
				trigger: 'axis',
				formatter: (params: unknown) => {
					if (!params || !Array.isArray(params) || params.length === 0) return '';
					const rows = params as Array<{ value: [string, number]; seriesName: string }>;
					let result = `${new Date(rows[0].value[0]).toLocaleDateString('pl-PL')}<br/>`;
					rows.forEach((p) => {
						result += `${p.seriesName}: ${formatPLN(p.value[1])}<br/>`;
					});
					return result;
				}
			},
			legend: {
				top: 44,
				left: 'center',
				type: 'scroll',
				selectedMode: false,
				data: series.map((s) => s.name)
			},
			xAxis: { type: 'time' },
			yAxis: {
				type: 'value',
				axisLabel: { formatter: (value: number) => formatPLN(value) }
			},
			series,
			grid: { left: '80px', right: '40px', top: 90, bottom: 40 }
		};
		chart.setOption(option, { notMerge: true });
	}

	$effect(() => {
		// Touch reactive dependencies so chart redraws on data + toggle changes.
		void [data.salaries.salary_records, cpiSeries, showNominal, showReal, showInflationTracked];

		if (!chartContainer) return;
		if (!chart) chart = echarts.init(chartContainer);
		applyChart();
	});

	onMount(() => () => {
		chart?.dispose();
		chart = undefined;
	});
</script>

<svelte:head>
	<title>Wynagrodzenia | Finansowa Forteca</title>
</svelte:head>

<div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4 mb-6">
	<div class="space-y-1">
		<h1 class="h2">Historia wynagrodzeń</h1>
		<p class="text-surface-700-300 text-sm">Śledź zmiany wynagrodzenia w czasie</p>
	</div>
	<button
		type="button"
		class="btn preset-filled-primary-500 w-full sm:w-auto gap-2"
		onclick={openNewSalaryModal}
	>
		<Plus size={16} />
		Nowe Wynagrodzenie
	</button>
</div>

<div class="space-y-4">
	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3">
			<div>
				<h3 class="h3 flex items-center gap-2">
					<Wallet size={20} /> Total compensation
				</h3>
				<p class="text-xs text-surface-700-300">
					Roczna pensja podstawowa + bonusy + equity (opcjonalnie). Wszystko w PLN.
				</p>
			</div>
			<div class="flex flex-wrap gap-2">
				<label class="label">
					<span class="text-xs">Rok</span>
					<select class="select" bind:value={totalCompYear}>
						{#each allYears as y (y)}
							<option value={y}>{y}</option>
						{/each}
					</select>
				</label>
				<label class="label">
					<span class="text-xs">Właściciel</span>
					<select class="select" bind:value={totalCompOwner}>
						{#each owners as owner (owner.id)}
							<option value={owner.id}>{owner.name}</option>
						{/each}
					</select>
				</label>
			</div>
		</header>

		{#if compSummary}
			<div class="grid grid-cols-2 md:grid-cols-4 gap-4">
				<div class="card preset-tonal-surface p-3">
					<div class="text-xs text-surface-700-300">Pensja podstawowa (gross)</div>
					<div class="text-lg font-semibold">{formatPLN(compSummary.baseAnnualGross)}</div>
					<div class="text-xs text-surface-700-300">
						netto: {formatPLN(compSummary.baseAnnualNet)}
					</div>
				</div>
				<div class="card preset-tonal-surface p-3">
					<div class="text-xs text-surface-700-300">Bonusy w {totalCompYear}</div>
					<div class="text-lg font-semibold">{formatPLN(compSummary.bonusesPln)}</div>
					{#if compSummary.bonusesPln > 0}
						<div class="text-xs text-surface-700-300">
							netto (szac.): {formatPLN(compSummary.bonusesNetPln)}
						</div>
					{/if}
				</div>
				<div class="card preset-tonal-surface p-3">
					<div class="text-xs text-surface-700-300">Equity (paper, dziś)</div>
					<div class="text-lg font-semibold">{formatPLN(compSummary.equityPaperPln)}</div>
					{#if compSummary.equityPaperLowPln !== compSummary.equityPaperHighPln}
						<div class="text-xs text-surface-700-300">
							{formatPLN(compSummary.equityPaperLowPln)}–{formatPLN(compSummary.equityPaperHighPln)}
						</div>
					{/if}
					{#if compSummary.equityPaperPln > 0}
						<div class="text-xs text-surface-700-300">
							po podatku 19% (szac.): {formatPLN(compSummary.equityNetPln)}
						</div>
					{/if}
					{#if compSummary.hasEquityWithoutFx}
						<div class="text-xs text-warning-500">część grantów bez FX</div>
					{/if}
				</div>
				<div class="card preset-tonal-surface p-3">
					<div class="text-xs text-surface-700-300">
						Total {includeEquityInTotal ? '(z equity)' : '(bez equity)'}
					</div>
					<div class="text-xl font-bold text-primary-600-400">{formatPLN(totalCompGross)}</div>
					<div class="text-xs text-surface-700-300">
						po podatku (szac.): {formatPLN(totalCompNet)}
					</div>
				</div>
			</div>
			<div class="text-xs text-surface-700-300">
				Pensja + bonusy filtrowane po roku. Equity zawsze jako wartość dzisiejsza (bieżący vested ×
				najnowsza wycena), niezależnie od wybranego roku.
			</div>
			<label class="flex items-center gap-2 text-sm cursor-pointer">
				<input type="checkbox" class="checkbox" bind:checked={includeEquityInTotal} />
				<span>Wlicz equity paper value do total (uwaga: nie zrealizowane do sprzedaży)</span>
			</label>
		{:else}
			<p class="text-sm text-surface-700-300">Wybierz właściciela aby zobaczyć podsumowanie.</p>
		{/if}
	</div>

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header>
			<h3 class="h3 flex items-center gap-2"><Banknote size={20} /> Aktualne wynagrodzenia</h3>
		</header>
		<div class="flex flex-wrap gap-6">
			{#each currentSalaryRows as row (row.name)}
				<div class="flex flex-col gap-1">
					<span class="text-sm text-surface-700-300">{row.name}</span>
					<strong class="text-lg">
						{row.salary !== null ? formatPLN(row.salary) : 'Brak danych'}
						{#if row.salary !== null}
							<span class="text-xs font-normal text-surface-700-300">brutto</span>
						{/if}
					</strong>
					{#if row.net !== null}
						<span class="text-sm text-success-500 font-semibold">
							≈ {formatPLN(row.net)} <span class="text-xs font-normal">netto/mc</span>
						</span>
					{/if}
				</div>
			{/each}
		</div>
	</div>

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header>
			<h3 class="h3 flex items-center gap-2">
				<Scale size={20} /> Wpływ inflacji (od ostatniej podwyżki)
			</h3>
			<p class="text-xs text-surface-700-300">
				Źródło danych CPI: GUS (Wskaźnik cen towarów i usług konsumpcyjnych — ogółem)
			</p>
		</header>
		{#if inflationEntries.length === 0}
			<p class="text-sm text-surface-700-300">
				Za mało danych — dodaj kolejną zmianę pensji lub poczekaj na świeże dane CPI, aby zobaczyć
				realny wpływ inflacji.
			</p>
		{:else}
			<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
				{#each inflationEntries as ctx (ctx.owner_user_id)}
					<div class="card preset-tonal-surface p-4 space-y-2">
						<div class="flex items-baseline justify-between flex-wrap gap-2">
							<strong class="text-lg">{ownerName(owners, ctx.owner_user_id)}</strong>
							<span class="text-xs text-surface-700-300">
								od {new Date(ctx.last_change_date).toLocaleDateString('pl-PL')}
								{#if getPreviousCompany(ctx.owner_user_id, ctx.previous_change_date)}
									· {getPreviousCompany(ctx.owner_user_id, ctx.previous_change_date)}
								{/if}
							</span>
						</div>
						<dl class="grid grid-cols-[auto,1fr] gap-x-4 gap-y-1 text-sm">
							<dt class="text-surface-700-300">Poprzednia pensja:</dt>
							<dd class="text-right font-semibold">{formatPLN(ctx.previous_salary)}</dd>

							<dt class="text-surface-700-300">W dzisiejszych PLN:</dt>
							<dd class="text-right font-semibold">
								{formatPLN(ctx.previous_salary_in_today_pln)}
							</dd>

							<dt class="text-surface-700-300">Obecna pensja:</dt>
							<dd class="text-right font-semibold">{formatPLN(ctx.current_salary)}</dd>

							<dt class="font-semibold pt-1">Realna podwyżka:</dt>
							<dd
								class="text-right font-bold pt-1"
								class:text-success-500={isNonNegative(ctx.real_change_pln)}
								class:text-error-500={!isNonNegative(ctx.real_change_pln)}
							>
								{formatPlnSigned(ctx.real_change_pln)}
								<span class="text-xs font-normal">
									({formatPctSigned(ctx.real_change_pct)})
								</span>
							</dd>
						</dl>
						<p class="text-xs text-surface-700-300">
							CPI na koniec: {ctx.cpi_as_of_year}
						</p>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header>
			<h3 class="h3 flex items-center gap-2"><TrendingUp size={20} /> Progresja wynagrodzenia</h3>
			<p class="text-xs text-surface-700-300">
				Linia ciągła: pensja nominalna. Linia przerywana: nominalna przeliczona na dzisiejsze PLN wg
				CPI GUS. Linia kropkowana: hipotetyczna pensja, gdyby od pierwszej zmiany rosła tylko o
				inflację.
			</p>
		</header>
		<div class="flex flex-wrap gap-4 text-sm">
			<label class="flex items-center gap-2 cursor-pointer">
				<input type="checkbox" class="checkbox" bind:checked={showNominal} />
				<span>Pensja nominalna</span>
			</label>
			<label class="flex items-center gap-2 cursor-pointer">
				<input type="checkbox" class="checkbox" bind:checked={showReal} />
				<span>Realna wartość (dzisiejsze PLN)</span>
			</label>
			<label class="flex items-center gap-2 cursor-pointer">
				<input type="checkbox" class="checkbox" bind:checked={showInflationTracked} />
				<span>Indeksowana inflacją</span>
			</label>
		</div>
		<div bind:this={chartContainer} style="width: 100%; height: 400px;"></div>
	</div>

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header>
			<h3 class="h3 flex items-center gap-2"><Search size={20} /> Filtry</h3>
		</header>
		<form
			class="space-y-4"
			onsubmit={(event) => {
				event.preventDefault();
				applyFilters();
			}}
		>
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
				<label class="label">
					<span class="font-semibold text-sm">Właściciel</span>
					<select class="select" bind:value={filterOwnerUserId}>
						<option value={null}>Wszystkie</option>
						{#each owners as owner (owner.id)}
							<option value={owner.id}>{owner.name}</option>
						{/each}
					</select>
				</label>

				<label class="label">
					<span class="font-semibold text-sm">Firma</span>
					<select class="select" bind:value={filterCompany}>
						<option value="">Wszystkie</option>
						{#each data.salaries.available_companies as company}
							<option value={company}>{company}</option>
						{/each}
					</select>
				</label>

				<label class="label">
					<span class="font-semibold text-sm">Data od</span>
					<input type="date" class="input" bind:value={filterDateFrom} />
				</label>

				<label class="label">
					<span class="font-semibold text-sm">Data do</span>
					<input type="date" class="input" bind:value={filterDateTo} />
				</label>
			</div>

			<div class="flex flex-col sm:flex-row gap-2">
				<button type="submit" class="btn preset-filled-primary-500">Filtruj</button>
				<button type="button" class="btn preset-tonal-surface" onclick={clearFilters}
					>Wyczyść filtry</button
				>
			</div>
		</form>
	</div>

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header>
			<h3 class="h3 flex items-center gap-2"><BarChart3 size={20} /> Historia zmian</h3>
		</header>
		{#if data.salaries.salary_records.length === 0}
			<div class="text-center py-12 text-surface-700-300">
				<p>Brak rekordów wynagrodzeń</p>
			</div>
		{:else}
			<div class="table-wrap">
				<table class="table table-hover">
					<thead>
						<tr>
							<th>Data zmiany</th>
							<th>Właściciel</th>
							<th>Firma</th>
							<th>Pensja brutto</th>
							<th>Rodzaj umowy</th>
							<th class="text-right">Akcje</th>
						</tr>
					</thead>
					<tbody>
						{#each data.salaries.salary_records as record}
							<tr>
								<td>{new Date(record.date).toLocaleDateString('pl-PL')}</td>
								<td>{ownerName(owners, record.owner_user_id)}</td>
								<td>{record.company}</td>
								<td class="font-semibold text-primary-600-400">{formatPLN(record.gross_amount)}</td>
								<td>{record.contract_type}</td>
								<td class="text-right whitespace-nowrap">
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Edytuj"
										onclick={() => openEditSalaryModal(record)}
									>
										<Pencil size={16} />
									</button>
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Usuń"
										onclick={() => deleteSalary(record.id)}
									>
										<Trash2 size={16} />
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3">
			<div>
				<h3 class="h3 flex items-center gap-2"><Gift size={20} /> Premie i bonusy</h3>
				<p class="text-xs text-surface-700-300">
					Roczne, powitalne, uznaniowe i retencyjne. Pokazywane w oryginalnej walucie.
				</p>
			</div>
			<button type="button" class="btn preset-filled-primary-500 gap-2" onclick={openNewBonusModal}>
				<Plus size={16} />
				Nowy bonus
			</button>
		</header>

		{#if bonusEvents.length === 0}
			<div class="text-center py-8 text-surface-700-300">
				<p>Brak zarejestrowanych bonusów</p>
			</div>
		{:else}
			<div class="space-y-4">
				{#each [...bonusGroupedByCompany.entries()] as [company, bonuses] (company)}
					<div class="card preset-tonal-surface p-3 space-y-2">
						<header class="flex items-baseline justify-between flex-wrap gap-2">
							<strong class="text-base">{company}</strong>
							<span class="text-xs text-surface-700-300">
								{bonuses.length}
								{bonuses.length === 1 ? 'bonus' : 'bonusów'}
							</span>
						</header>
						<div class="table-wrap">
							<table class="table table-hover">
								<thead>
									<tr>
										<th>Data</th>
										<th>Typ</th>
										<th>Właściciel</th>
										<th>Kwota</th>
										<th>Notatki</th>
										<th class="text-right">Akcje</th>
									</tr>
								</thead>
								<tbody>
									{#each bonuses as bonus (bonus.id)}
										<tr>
											<td>{new Date(bonus.date).toLocaleDateString('pl-PL')}</td>
											<td>{bonusTypeLabels[bonus.type]}</td>
											<td>{ownerName(owners, bonus.owner_user_id)}</td>
											<td class="font-semibold text-primary-600-400">
												{formatBonusAmount(bonus.amount, bonus.currency)}
												{#if bonus.currency !== 'PLN' && bonus.amount_pln !== null}
													<br /><span class="text-xs font-normal text-surface-700-300">
														≈ {formatPLN(bonus.amount_pln)}
														{#if bonus.fx_rate}@ {bonus.fx_rate.toFixed(4)}{/if}
													</span>
												{/if}
											</td>
											<td class="text-sm text-surface-700-300">{bonus.notes ?? ''}</td>
											<td class="text-right whitespace-nowrap">
												<button
													type="button"
													class="btn-icon btn-icon-sm"
													aria-label="Edytuj"
													onclick={() => openEditBonusModal(bonus)}
												>
													<Pencil size={16} />
												</button>
												<button
													type="button"
													class="btn-icon btn-icon-sm"
													aria-label="Usuń"
													onclick={() => deleteBonus(bonus.id)}
												>
													<Trash2 size={16} />
												</button>
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3">
			<div>
				<h3 class="h3 flex items-center gap-2"><Award size={20} /> Equity (opcje + RSU)</h3>
				<p class="text-xs text-surface-700-300">
					Grupy po firmie. Vested = ile akcji już Ci się odblokowało dziś. Dla RSU z double-trigger
					pokazane jest 0 dopóki nie wystąpi liquidity event.
				</p>
			</div>
			<button
				type="button"
				class="btn preset-filled-primary-500 gap-2"
				onclick={openNewEquityModal}
			>
				<Plus size={16} />
				Nowy grant
			</button>
		</header>

		{#if equityGrants.length === 0}
			<div class="text-center py-8 text-surface-700-300">
				<p>Brak zarejestrowanych grantów</p>
			</div>
		{:else}
			<div class="space-y-4">
				{#each equityGroups as group (group.company)}
					<div class="card preset-tonal-surface p-3 space-y-2">
						<header class="flex items-baseline justify-between flex-wrap gap-2">
							<strong class="text-base">{group.company}</strong>
							<span class="text-xs text-surface-700-300">
								{group.grants.length}
								{group.grantLabel} ·
								{formatShares(group.vestedShares)} / {formatShares(group.totalShares)} vested
								{#if group.hasPaperValue}
									· paper {formatCurrency(group.paperBase, group.currency)}
									{#if group.hasPaperValuePln}
										(≈ {formatPLN(group.paperBasePln)})
									{/if}
								{/if}
							</span>
						</header>
						<div class="table-wrap">
							<table class="table table-hover">
								<thead>
									<tr>
										<th>Data grantu</th>
										<th>Typ</th>
										<th>Właściciel</th>
										<th>Akcje (vested / total)</th>
										<th>Strike</th>
										<th>Paper value</th>
										<th>Vesting</th>
										<th>Status</th>
										<th class="text-right">Akcje</th>
									</tr>
								</thead>
								<tbody>
									{#each group.grants as grant (grant.id)}
										<tr>
											<td>{new Date(grant.grant_date).toLocaleDateString('pl-PL')}</td>
											<td>{equityTypeLabels[grant.type]}</td>
											<td>{ownerName(owners, grant.owner_user_id)}</td>
											<td class="font-semibold">
												{formatShares(grant.vested_shares_today)} /
												{formatShares(grant.total_shares)}
												<span class="text-xs text-surface-700-300">
													({grant.vesting_progress_pct.toFixed(1)}%)
												</span>
											</td>
											<td>
												{#if grant.strike_price !== null}
													{formatCurrency(grant.strike_price, grant.currency)}
												{:else}
													—
												{/if}
											</td>
											<td class="text-xs">
												{#if grant.paper_value_base !== null}
													<span class="font-semibold">{formatRange(grant)}</span>
													{#if grant.paper_value_base_pln !== null && grant.paper_value_currency !== 'PLN'}
														<br /><span class="text-surface-700-300">
															≈ {formatPLN(grant.paper_value_base_pln)}
															{#if grant.fx_rate}@ {grant.fx_rate.toFixed(4)}{/if}
														</span>
													{/if}
													{#if grant.valuation_date}
														<br /><span class="text-surface-700-300"
															>wg {new Date(grant.valuation_date).toLocaleDateString('pl-PL')}</span
														>
													{/if}
												{:else if grant.valuation_date}
													<span class="text-warning-500">{formatRange(grant)}</span>
												{:else}
													<span class="text-surface-700-300">brak wyceny</span>
												{/if}
											</td>
											<td class="text-xs">
												{grant.vest_cliff_months}m cliff · {grant.vest_total_months}m · {vestingFrequencyLabels[
													grant.vest_frequency
												]}
												{#if grant.vest_custom_schedule}
													<br /><span class="text-surface-700-300">niestandardowy harmonogram</span>
												{/if}
											</td>
											<td class="text-xs">
												{#if grant.requires_liquidity_event && !grant.liquidity_event_date}
													<span class="text-warning-500">double-trigger: oczekuje</span>
												{:else if grant.requires_liquidity_event}
													<span class="text-success-500">trigger uruchomiony</span>
												{:else}
													<span class="text-surface-700-300">single-trigger</span>
												{/if}
											</td>
											<td class="text-right whitespace-nowrap">
												<button
													type="button"
													class="btn-icon btn-icon-sm"
													aria-label="Edytuj"
													onclick={() => openEditEquityModal(grant)}
												>
													<Pencil size={16} />
												</button>
												<button
													type="button"
													class="btn-icon btn-icon-sm"
													aria-label="Usuń"
													onclick={() => deleteEquityGrant(grant.id)}
												>
													<Trash2 size={16} />
												</button>
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-3">
			<div>
				<h3 class="h3 flex items-center gap-2"><Building2 size={20} /> Wycena spółek</h3>
				<p class="text-xs text-surface-700-300">
					FMV per share dla spółek prywatnych (i publicznych). Range low/high pokazuje niepewność.
				</p>
			</div>
			<button
				type="button"
				class="btn preset-filled-primary-500 gap-2"
				onclick={openNewValuationModal}
			>
				<Plus size={16} />
				Nowa wycena
			</button>
		</header>

		{#if valuations.length === 0}
			<div class="text-center py-8 text-surface-700-300">
				<p>Brak wycen — dodaj wycenę, aby zobaczyć paper value grantów</p>
			</div>
		{:else}
			<div class="table-wrap">
				<table class="table table-hover">
					<thead>
						<tr>
							<th>Firma</th>
							<th>Data</th>
							<th>FMV / akcję</th>
							<th>Zakres (low–high)</th>
							<th>Źródło</th>
							<th>Discount</th>
							<th>Notatki</th>
							<th class="text-right">Akcje</th>
						</tr>
					</thead>
					<tbody>
						{#each valuations as valuation (valuation.id)}
							<tr>
								<td class="font-semibold">{valuation.company}</td>
								<td>{new Date(valuation.date).toLocaleDateString('pl-PL')}</td>
								<td>{formatCurrency(valuation.fmv_per_share, valuation.currency)}</td>
								<td class="text-xs">
									{#if valuation.fmv_low !== null || valuation.fmv_high !== null}
										{valuation.fmv_low !== null
											? formatCurrency(valuation.fmv_low, valuation.currency)
											: '—'}
										–
										{valuation.fmv_high !== null
											? formatCurrency(valuation.fmv_high, valuation.currency)
											: '—'}
									{:else}
										<span class="text-surface-700-300">—</span>
									{/if}
								</td>
								<td class="text-xs">{valuationSourceLabels[valuation.source]}</td>
								<td class="text-xs">
									{#if valuation.common_stock_discount_pct !== null}
										{valuation.common_stock_discount_pct}%
									{:else}
										<span class="text-surface-700-300">—</span>
									{/if}
								</td>
								<td class="text-sm text-surface-700-300">{valuation.notes ?? ''}</td>
								<td class="text-right whitespace-nowrap">
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Edytuj"
										onclick={() => openEditValuationModal(valuation)}
									>
										<Pencil size={16} />
									</button>
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Usuń"
										onclick={() => deleteValuation(valuation.id)}
									>
										<Trash2 size={16} />
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</div>
</div>

<Modal
	open={showNewSalaryModal}
	title={editingSalary ? 'Edytuj wynagrodzenie' : 'Nowe wynagrodzenie'}
	onConfirm={saveSalary}
	onCancel={closeSalaryModal}
	confirmText={savingSalary ? 'Zapisywanie...' : 'Zapisz'}
	confirmDisabled={savingSalary}
>
	<form
		onsubmit={(event) => {
			event.preventDefault();
			saveSalary();
		}}
		class="space-y-4"
	>
		{#if salaryError}
			<div class="card preset-filled-error-500 p-3 text-sm">{salaryError}</div>
		{/if}

		<label class="label">
			<span class="font-semibold text-sm">Data zmiany*</span>
			<input type="date" class="input" bind:value={salaryFormData.date} max={today} required />
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Pensja brutto (PLN)*</span>
			<input
				type="number"
				class="input"
				bind:value={salaryFormData.gross_amount}
				min="0"
				step="0.01"
				required
			/>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Rodzaj umowy*</span>
			<select class="select" bind:value={salaryFormData.contract_type} required>
				<option value="UOP">UOP</option>
				<option value="UZ">UZ</option>
				<option value="UoD">UoD</option>
				<option value="B2B">B2B</option>
			</select>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Firma*</span>
			<input
				type="text"
				class="input"
				bind:value={salaryFormData.company}
				placeholder="Nazwa firmy"
				required
			/>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Właściciel*</span>
			<select class="select" bind:value={salaryFormData.owner_user_id} required>
				{#each owners as owner (owner.id)}
					<option value={owner.id}>{owner.name}</option>
				{/each}
			</select>
		</label>
	</form>
</Modal>

<Modal
	open={showBonusModal}
	title={editingBonus ? 'Edytuj bonus' : 'Nowy bonus'}
	onConfirm={saveBonus}
	onCancel={closeBonusModal}
	confirmText={savingBonus ? 'Zapisywanie...' : 'Zapisz'}
	confirmDisabled={savingBonus}
>
	<form
		onsubmit={(event) => {
			event.preventDefault();
			saveBonus();
		}}
		class="space-y-4"
	>
		{#if bonusError}
			<div class="card preset-filled-error-500 p-3 text-sm">{bonusError}</div>
		{/if}

		<label class="label">
			<span class="font-semibold text-sm">Data wypłaty*</span>
			<input type="date" class="input" bind:value={bonusFormData.date} max={today} required />
		</label>

		<div class="grid grid-cols-3 gap-2">
			<label class="label col-span-2">
				<span class="font-semibold text-sm">Kwota*</span>
				<input
					type="number"
					class="input"
					bind:value={bonusFormData.amount}
					min="0"
					step="0.01"
					required
				/>
			</label>
			<label class="label">
				<span class="font-semibold text-sm">Waluta*</span>
				<select class="select" bind:value={bonusFormData.currency} required>
					<option value="PLN">PLN</option>
					<option value="USD">USD</option>
					<option value="EUR">EUR</option>
					<option value="GBP">GBP</option>
					<option value="CHF">CHF</option>
				</select>
			</label>
		</div>

		<label class="label">
			<span class="font-semibold text-sm">Typ*</span>
			<select class="select" bind:value={bonusFormData.type} required>
				<option value="annual">Roczny</option>
				<option value="signon">Powitalny</option>
				<option value="spot">Uznaniowy</option>
				<option value="retention">Retencyjny</option>
			</select>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Firma*</span>
			<input
				type="text"
				class="input"
				bind:value={bonusFormData.company}
				placeholder="Nazwa firmy"
				required
			/>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Właściciel*</span>
			<select class="select" bind:value={bonusFormData.owner_user_id} required>
				{#each owners as owner (owner.id)}
					<option value={owner.id}>{owner.name}</option>
				{/each}
			</select>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Rodzaj umowy*</span>
			<select class="select" bind:value={bonusFormData.contract_type} required>
				<option value="UOP">UOP</option>
				<option value="UZ">UZ</option>
				<option value="UoD">UoD</option>
				<option value="B2B">B2B</option>
			</select>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Notatki</span>
			<input
				type="text"
				class="input"
				bind:value={bonusFormData.notes}
				placeholder="np. Q4 performance bonus"
			/>
		</label>
	</form>
</Modal>

<Modal
	open={showEquityModal}
	title={editingGrant ? 'Edytuj grant' : 'Nowy grant'}
	onConfirm={saveEquityGrant}
	onCancel={closeEquityModal}
	confirmText={savingEquity ? 'Zapisywanie...' : 'Zapisz'}
	confirmDisabled={savingEquity}
>
	<form
		onsubmit={(event) => {
			event.preventDefault();
			saveEquityGrant();
		}}
		class="space-y-4"
	>
		{#if equityError}
			<div class="card preset-filled-error-500 p-3 text-sm">{equityError}</div>
		{/if}

		<div class="grid grid-cols-2 gap-2">
			<label class="label">
				<span class="font-semibold text-sm">Typ*</span>
				<select class="select" bind:value={equityFormData.type} required>
					<option value="rsu">RSU</option>
					<option value="option">Opcje</option>
				</select>
			</label>
			<label class="label">
				<span class="font-semibold text-sm">Data grantu*</span>
				<input
					type="date"
					class="input"
					bind:value={equityFormData.grant_date}
					max={today}
					required
				/>
			</label>
		</div>

		<label class="label">
			<span class="font-semibold text-sm">Firma*</span>
			<input
				type="text"
				class="input"
				bind:value={equityFormData.company}
				placeholder="Nazwa firmy"
				required
			/>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Właściciel*</span>
			<select class="select" bind:value={equityFormData.owner_user_id} required>
				{#each owners as owner (owner.id)}
					<option value={owner.id}>{owner.name}</option>
				{/each}
			</select>
		</label>

		<div class="grid grid-cols-2 gap-2">
			<label class="label">
				<span class="font-semibold text-sm">Liczba akcji*</span>
				<input
					type="number"
					class="input"
					bind:value={equityFormData.total_shares}
					min="1"
					step="1"
					required
				/>
			</label>
			<label class="label">
				<span class="font-semibold text-sm">Waluta*</span>
				<select class="select" bind:value={equityFormData.currency} required>
					<option value="USD">USD</option>
					<option value="EUR">EUR</option>
					<option value="PLN">PLN</option>
					<option value="GBP">GBP</option>
					<option value="CHF">CHF</option>
				</select>
			</label>
		</div>

		{#if equityFormData.type === 'option'}
			<label class="label">
				<span class="font-semibold text-sm">Strike price (cena wykonania)*</span>
				<input
					type="number"
					class="input"
					bind:value={equityFormData.strike_price}
					min="0"
					step="0.0001"
					required
				/>
			</label>
		{/if}

		<fieldset class="card preset-tonal-surface p-3 space-y-3">
			<legend class="font-semibold text-sm px-1">Harmonogram vestingu</legend>

			<label class="label">
				<span class="font-semibold text-sm">Schemat</span>
				<select
					class="select"
					bind:value={equityFormData.preset}
					onchange={() => applyPreset(equityFormData.preset)}
				>
					{#each Object.entries(vestingPresets) as [key, preset]}
						<option value={key}>{preset.label}</option>
					{/each}
				</select>
			</label>

			<label class="label">
				<span class="font-semibold text-sm">Data startu vestingu*</span>
				<input type="date" class="input" bind:value={equityFormData.vest_start_date} required />
			</label>

			<div class="grid grid-cols-3 gap-2">
				<label class="label">
					<span class="font-semibold text-sm">Cliff (msc)</span>
					<input
						type="number"
						class="input"
						bind:value={equityFormData.vest_cliff_months}
						min="0"
						step="1"
					/>
				</label>
				<label class="label">
					<span class="font-semibold text-sm">Całość (msc)*</span>
					<input
						type="number"
						class="input"
						bind:value={equityFormData.vest_total_months}
						min="1"
						step="1"
						required
					/>
				</label>
				<label class="label">
					<span class="font-semibold text-sm">Częstotliwość</span>
					<select class="select" bind:value={equityFormData.vest_frequency}>
						<option value="monthly">Miesięczna</option>
						<option value="quarterly">Kwartalna</option>
						<option value="yearly">Roczna</option>
					</select>
				</label>
			</div>

			{#if equityFormData.preset === 'custom'}
				<div class="text-xs text-surface-700-300">
					Niestandardowy harmonogram: lista zdarzeń (miesiąc + % od całości).
				</div>
				{#each equityFormData.vest_custom_schedule ?? [] as event, idx (idx)}
					<div class="grid grid-cols-[1fr,1fr,auto] gap-2 items-end">
						<label class="label">
							<span class="text-xs">Miesiąc</span>
							<input type="number" class="input" bind:value={event.month} min="0" step="1" />
						</label>
						<label class="label">
							<span class="text-xs">% od całości</span>
							<input type="number" class="input" bind:value={event.pct} min="0" step="0.1" />
						</label>
						<button
							type="button"
							class="btn-icon btn-icon-sm"
							aria-label="Usuń wiersz"
							onclick={() => {
								equityFormData.vest_custom_schedule =
									equityFormData.vest_custom_schedule?.filter((_, i) => i !== idx) ?? null;
							}}
						>
							<Trash2 size={14} />
						</button>
					</div>
				{/each}
				<button
					type="button"
					class="btn preset-tonal-surface btn-sm"
					onclick={() => {
						const next = [...(equityFormData.vest_custom_schedule ?? []), { month: 0, pct: 0 }];
						equityFormData.vest_custom_schedule = next;
					}}
				>
					<Plus size={14} /> Dodaj zdarzenie
				</button>
			{/if}
		</fieldset>

		<fieldset class="card preset-tonal-surface p-3 space-y-3">
			<legend class="font-semibold text-sm px-1">Liquidity event (double-trigger)</legend>
			<label class="flex items-center gap-2 cursor-pointer">
				<input
					type="checkbox"
					class="checkbox"
					bind:checked={equityFormData.requires_liquidity_event}
				/>
				<span class="text-sm">Wymaga liquidity event (IPO / akwizycja)</span>
			</label>
			{#if equityFormData.requires_liquidity_event}
				<label class="label">
					<span class="font-semibold text-sm">Data liquidity event</span>
					<input type="date" class="input" bind:value={equityFormData.liquidity_event_date} />
					<span class="text-xs text-surface-700-300"
						>Puste = jeszcze nie wystąpiło. Vested = 0 dopóki nie wystąpi.</span
					>
				</label>
			{/if}
		</fieldset>

		<label class="label">
			<span class="font-semibold text-sm">Traktowanie podatkowe (PL)</span>
			<select class="select" bind:value={equityFormData.tax_treatment}>
				{#each Object.entries(taxTreatmentLabels) as [key, label]}
					<option value={key}>{label}</option>
				{/each}
			</select>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Notatki</span>
			<input
				type="text"
				class="input"
				bind:value={equityFormData.notes}
				placeholder="np. ESOP 2024, double-trigger RSU"
			/>
		</label>
	</form>
</Modal>

<Modal
	open={showValuationModal}
	title={editingValuation ? 'Edytuj wycenę' : 'Nowa wycena'}
	onConfirm={saveValuation}
	onCancel={closeValuationModal}
	confirmText={savingValuation ? 'Zapisywanie...' : 'Zapisz'}
	confirmDisabled={savingValuation}
>
	<form
		onsubmit={(event) => {
			event.preventDefault();
			saveValuation();
		}}
		class="space-y-4"
	>
		{#if valuationError}
			<div class="card preset-filled-error-500 p-3 text-sm">{valuationError}</div>
		{/if}

		<label class="label">
			<span class="font-semibold text-sm">Firma*</span>
			<input
				type="text"
				class="input"
				bind:value={valuationFormData.company}
				placeholder="Nazwa firmy"
				required
			/>
		</label>

		<div class="grid grid-cols-2 gap-2">
			<label class="label">
				<span class="font-semibold text-sm">Data wyceny*</span>
				<input type="date" class="input" bind:value={valuationFormData.date} required />
			</label>
			<label class="label">
				<span class="font-semibold text-sm">Waluta*</span>
				<select class="select" bind:value={valuationFormData.currency} required>
					<option value="USD">USD</option>
					<option value="EUR">EUR</option>
					<option value="PLN">PLN</option>
					<option value="GBP">GBP</option>
					<option value="CHF">CHF</option>
				</select>
			</label>
		</div>

		<label class="label">
			<span class="font-semibold text-sm">FMV per share (bazowa)*</span>
			<input
				type="number"
				class="input"
				bind:value={valuationFormData.fmv_per_share}
				min="0"
				step="0.0001"
				required
			/>
		</label>

		<div class="grid grid-cols-2 gap-2">
			<label class="label">
				<span class="font-semibold text-sm">FMV low (opcjonalna)</span>
				<input
					type="number"
					class="input"
					bind:value={valuationFormData.fmv_low}
					min="0"
					step="0.0001"
				/>
			</label>
			<label class="label">
				<span class="font-semibold text-sm">FMV high (opcjonalna)</span>
				<input
					type="number"
					class="input"
					bind:value={valuationFormData.fmv_high}
					min="0"
					step="0.0001"
				/>
			</label>
		</div>

		<label class="label">
			<span class="font-semibold text-sm">Źródło*</span>
			<select class="select" bind:value={valuationFormData.source} required>
				<option value="409a">409A</option>
				<option value="preferred_round">Runda preferred</option>
				<option value="tender">Tender / wykup</option>
				<option value="estimate">Estymacja</option>
			</select>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Common stock discount (%) — opcjonalne</span>
			<input
				type="number"
				class="input"
				bind:value={valuationFormData.common_stock_discount_pct}
				min="0"
				max="100"
				step="0.1"
				placeholder="np. 30"
			/>
			<span class="text-xs text-surface-700-300"
				>Stosowane przy wycenie preferred → common (zwykle 20–40%)</span
			>
		</label>

		<label class="label">
			<span class="font-semibold text-sm">Notatki</span>
			<input
				type="text"
				class="input"
				bind:value={valuationFormData.notes}
				placeholder="np. Series C post-money"
			/>
		</label>
	</form>
</Modal>
