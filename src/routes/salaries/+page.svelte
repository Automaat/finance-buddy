<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import * as echarts from 'echarts';
	import type { EChartsOption } from 'echarts';
	import Modal from '$lib/components/Modal.svelte';
	import { formatPLN } from '$lib/utils/format';
	import { buildCpiLookup, inflationAdjust, parseIsoDate } from '$lib/utils/inflation';
	import {
		Plus,
		Banknote,
		TrendingUp,
		Search,
		BarChart3,
		Pencil,
		Trash2,
		Scale
	} from 'lucide-svelte';
	import { env } from '$env/dynamic/public';
	import { goto, invalidateAll } from '$app/navigation';
	import { toast } from '$lib/stores/toast.svelte';
	import type { SalaryRecord } from '$lib/types/salaries';
	import type { Persona } from '$lib/types/personas';
	import type { CpiSeries } from '$lib/types/cpi';
	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	const apiUrl = env.PUBLIC_API_URL_BROWSER || 'http://localhost:8000';
	const personas = $derived(data.personas as Persona[]);
	const defaultOwner = $derived(personas.length > 0 ? personas[0].name : 'Marcin');
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

	let filterOwner = $state(untrack(() => data.filters.owner || ''));
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
		owner: untrack(() => defaultOwner)
	});
	let salaryError = $state('');
	let savingSalary = $state(false);

	function applyFilters() {
		const params = new URLSearchParams();
		if (filterOwner) params.set('owner', filterOwner);
		if (filterDateFrom) params.set('date_from', filterDateFrom);
		if (filterDateTo) params.set('date_to', filterDateTo);
		if (filterCompany) params.set('company', filterCompany);

		goto(`/salaries?${params.toString()}`);
	}

	function clearFilters() {
		filterOwner = '';
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
			owner: defaultOwner
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
			owner: record.owner
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
		<header>
			<h3 class="h3 flex items-center gap-2"><Banknote size={20} /> Aktualne wynagrodzenia</h3>
		</header>
		<div class="flex flex-wrap gap-6">
			{#each Object.entries(data.salaries.current_salaries) as [name, salary]}
				<div class="flex items-center gap-2">
					<span class="text-sm text-surface-700-300">{name}:</span>
					<strong class="text-lg">
						{salary !== null ? formatPLN(salary) : 'Brak danych'}
					</strong>
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
				{#each inflationEntries as ctx (ctx.owner)}
					{@const realPositive = (ctx.real_change_pln ?? 0) >= 0}
					{@const previousRecord = data.salaries.salary_records.find(
						(r) => r.owner === ctx.owner && r.date === ctx.previous_change_date
					)}
					<div class="card preset-tonal-surface p-4 space-y-2">
						<div class="flex items-baseline justify-between flex-wrap gap-2">
							<strong class="text-lg">{ctx.owner}</strong>
							<span class="text-xs text-surface-700-300">
								od {new Date(ctx.last_change_date).toLocaleDateString('pl-PL')}
								{#if previousRecord?.company}
									· {previousRecord.company}
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
								class:text-success-500={realPositive}
								class:text-error-500={!realPositive}
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
					<select class="select" bind:value={filterOwner}>
						<option value="">Wszystkie</option>
						{#each personas as persona}
							<option value={persona.name}>{persona.name}</option>
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
								<td>{record.owner}</td>
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
			<select class="select" bind:value={salaryFormData.owner} required>
				{#each personas as persona}
					<option value={persona.name}>{persona.name}</option>
				{/each}
			</select>
		</label>
	</form>
</Modal>
