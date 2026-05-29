<script lang="ts">
	import { onMount, untrack } from 'svelte';
	import * as echarts from 'echarts';
	import type { EChartsOption } from 'echarts';
	import Modal from '$lib/components/Modal.svelte';
	import BondsMaturityLadder from '$lib/components/BondsMaturityLadder.svelte';
	import { formatPLN } from '$lib/utils/format';
	import { chartAccent, chartAccentGradient } from '$lib/utils/theme';
	import { createChart } from '$lib/utils/charts/lifecycle';
	import { Banknote, Pencil, Plus, Trash2, TrendingUp } from 'lucide-svelte';
	import { resolveApiUrl } from '$lib/api';
	import { api } from '$lib/apiClient';
	import { invalidateAll } from '$app/navigation';
	import { CrudForm } from '$lib/stores/crudForm.svelte';
	import { ownerName } from '$lib/types/owners';
	import type { TreasuryBond } from './+page';
	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}
	let { data }: Props = $props();
	const apiUrl = resolveApiUrl();

	const bondTypes = [
		{ value: 'EDO', label: 'EDO (10-letnie indeksowane)' },
		{ value: 'COI', label: 'COI (4-letnie indeksowane)' },
		{ value: 'ROR', label: 'ROR (roczne oszczędnościowe)' },
		{ value: 'TOZ', label: 'TOZ (3-letnie oszczędnościowe)' },
		{ value: 'DOS', label: 'DOS (2-letnie stałoprocentowe)' }
	];

	function defaultsForType(type: string) {
		switch (type) {
			case 'EDO':
				return { first_year_rate: 6.8, margin: 2.0, capitalize: true };
			case 'COI':
				return { first_year_rate: 6.55, margin: 1.25, capitalize: false };
			case 'ROR':
				return { first_year_rate: 6.0, margin: 0, capitalize: false };
			case 'TOZ':
				return { first_year_rate: 6.4, margin: 0, capitalize: false };
			case 'DOS':
				return { first_year_rate: 6.3, margin: 0, capitalize: true };
			default:
				return { first_year_rate: 0, margin: 0, capitalize: true };
		}
	}

	const bondForm = new CrudForm<TreasuryBond>();
	let lookingUp = $state(false);
	let showDeleteModal = $state(false);
	let bondToDelete: number | null = $state(null);

	type FormData = {
		type: string;
		series: string;
		face_value: number;
		purchase_date: string;
		owner_user_id: number | null;
		account_id: number | null;
		first_year_rate: number;
		margin: number;
		capitalize: boolean;
	};

	const today = new Date().toISOString().slice(0, 10);
	let formData: FormData = $state({
		type: 'EDO',
		series: '',
		face_value: 1000,
		purchase_date: today,
		owner_user_id: null,
		account_id: null,
		...defaultsForType('EDO')
	});

	function startCreate() {
		formData = {
			type: 'EDO',
			series: '',
			face_value: 1000,
			purchase_date: today,
			owner_user_id: null,
			account_id: null,
			...defaultsForType('EDO')
		};
		bondForm.openCreate();
	}

	function startEdit(bond: TreasuryBond) {
		formData = {
			type: bond.type,
			series: bond.series,
			face_value: bond.face_value,
			purchase_date: bond.purchase_date,
			owner_user_id: bond.owner_user_id,
			account_id: bond.account_id,
			first_year_rate: bond.first_year_rate,
			margin: bond.margin,
			capitalize: bond.capitalize
		};
		bondForm.openEdit(bond);
	}

	function cancelForm() {
		bondForm.close();
	}

	function applyTypeDefaults() {
		// Only refresh rate defaults when creating; editing preserves the
		// user-entered values that match the bond's actual coupon.
		if (bondForm.isEditing) return;
		const d = defaultsForType(formData.type);
		formData = { ...formData, ...d };
	}

	async function lookupRate() {
		bondForm.error = '';
		lookingUp = true;
		try {
			const body = await api.get<{ first_year_rate: number; margin: number }>('/api/bonds/lookup', {
				query: { type: formData.type, series: formData.series }
			});
			formData = {
				...formData,
				first_year_rate: body.first_year_rate,
				margin: body.margin
			};
		} catch (err) {
			if (err instanceof Error) bondForm.error = `Pobieranie stopy: ${err.message}`;
		} finally {
			lookingUp = false;
		}
	}

	async function handleSubmit() {
		const editing = bondForm.editing;
		await bondForm.submit(async () => {
			if (editing) {
				await api.put(`/api/bonds/${editing.id}`, formData);
			} else {
				await api.post('/api/bonds', formData);
			}
			await invalidateAll();
		});
	}

	function handleDelete(bondId: number) {
		bondToDelete = bondId;
		showDeleteModal = true;
	}
	function cancelDelete() {
		showDeleteModal = false;
		bondToDelete = null;
	}
	async function confirmDelete() {
		if (!bondToDelete) return;
		try {
			await api.del(`/api/bonds/${bondToDelete}`);
			await invalidateAll();
		} catch (err) {
			if (err instanceof Error) bondForm.error = err.message;
		} finally {
			showDeleteModal = false;
			bondToDelete = null;
		}
	}

	// Aggregate face + profit across all bonds. Face = "paid in" (each EDO
	// is bought at face); current_value = principal + accrued interest, so
	// profit is the straight delta. Same for COI in this app's model since
	// the calc returns face + accrued and doesn't deduct past coupon payouts
	// — fine for "where do I stand right now" tracking.
	const totalPaid = $derived(data.bonds.reduce((sum, b) => sum + b.face_value, 0));
	const totalProfit = $derived(data.total_value - totalPaid);
	const profitPct = $derived(totalPaid > 0 ? (totalProfit / totalPaid) * 100 : 0);

	// --- YTM chart ---
	type YTMPoint = { year: number; date: string; value: number; year_rate: number };
	let selectedBondId: number | null = $state(untrack(() => data.bonds[0]?.id ?? null));
	let ytm: YTMPoint[] = $state([]);
	let ytmLoading = $state(false);
	let ytmError = $state('');
	let chartContainer: HTMLDivElement | undefined = $state(undefined);
	let chart: echarts.ECharts | undefined = $state(undefined);

	$effect(() => {
		if (data.bonds.length > 0 && selectedBondId === null) {
			selectedBondId = data.bonds[0].id;
		}
		if (data.bonds.length === 0) {
			selectedBondId = null;
			ytm = [];
		}
	});

	$effect(() => {
		const id = selectedBondId;
		if (id === null) return;
		ytmLoading = true;
		ytmError = '';
		// Abort the in-flight YTM fetch when the selected bond changes or the
		// component unmounts, so a stale projection can't land on the chart.
		const controller = new AbortController();
		fetch(`${apiUrl}/api/bonds/${id}/ytm`, { signal: controller.signal })
			.then(async (r) => {
				if (!r.ok) throw new Error('Nie udało się załadować projekcji YTM');
				const body = await r.json();
				ytm = body.points as YTMPoint[];
			})
			.catch((err) => {
				if (controller.signal.aborted) return;
				ytmError = err instanceof Error ? err.message : 'Nieznany błąd';
			})
			.finally(() => {
				if (!controller.signal.aborted) ytmLoading = false;
			});
		return () => controller.abort();
	});

	onMount(() => {
		if (!chartContainer) return;
		const handle = createChart(chartContainer);
		chart = handle.chart;
		return () => {
			handle.dispose();
			chart = undefined;
		};
	});

	const ytmOption = $derived<EChartsOption>({
		title: { text: 'Wartość do wykupu (YTM)', left: 'center' },
		tooltip: {
			trigger: 'axis',
			formatter: (params: unknown) => {
				const arr = params as Array<{ value: [string, number]; data: YTMPoint }>;
				const p = arr[0];
				const date = new Date(p.value[0]).toLocaleDateString('pl-PL');
				const value = formatPLN(p.value[1]);
				const rate = (p.data.year_rate ?? 0).toFixed(2);
				return `${date}<br/>Wartość: ${value}<br/>Stopa roku: ${rate}%`;
			}
		},
		xAxis: { type: 'time' },
		yAxis: {
			type: 'value',
			axisLabel: { formatter: (v: number) => formatPLN(v) }
		},
		series: [
			{
				data: ytm.map((p) => ({ ...p, value: [p.date, p.value] })),
				type: 'line',
				smooth: true,
				symbol: 'circle',
				lineStyle: { color: chartAccent, width: 2 },
				areaStyle: {
					color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
						{ offset: 0, color: chartAccentGradient[0] },
						{ offset: 1, color: chartAccentGradient[1] }
					])
				}
			}
		]
	});

	$effect(() => {
		if (chart) chart.setOption(ytmOption, true);
	});
</script>

<svelte:head>
	<title>Obligacje skarbowe | Finansowa Forteca</title>
</svelte:head>

<div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4 mb-6">
	<div class="space-y-1">
		<h1 class="h2">Obligacje skarbowe</h1>
		<p class="text-surface-700-300 text-sm">
			Śledź obligacje EDO, COI, ROR, TOZ, DOS z auto-wyliczaną wartością wg CPI.
		</p>
	</div>
	<button
		type="button"
		class="btn preset-filled-primary-500 w-full sm:w-auto gap-2"
		onclick={startCreate}
	>
		<Plus size={16} />
		Dodaj obligację
	</button>
</div>

<div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
	<div class="card preset-filled-surface-100-900 p-4 space-y-1">
		<header class="text-sm text-surface-700-300">Wpłacono</header>
		<div class="text-2xl font-bold">{formatPLN(totalPaid)}</div>
	</div>
	<div class="card preset-filled-surface-100-900 p-4 space-y-1">
		<header class="text-sm text-surface-700-300">Wartość bieżąca</header>
		<div class="text-2xl font-bold text-primary-600-400">{formatPLN(data.total_value)}</div>
	</div>
	<div class="card preset-filled-surface-100-900 p-4 space-y-1">
		<header class="text-sm text-surface-700-300">Zysk</header>
		<div class="text-2xl font-bold {totalProfit >= 0 ? 'text-success-500' : 'text-error-500'}">
			{totalProfit >= 0 ? '+' : ''}{formatPLN(totalProfit)}
		</div>
		<div class="text-xs text-surface-600-400">
			{totalProfit >= 0 ? '+' : ''}{profitPct.toFixed(2)}%
		</div>
	</div>
	<div class="card preset-filled-surface-100-900 p-4 space-y-1">
		<header class="text-sm text-surface-700-300">Liczba obligacji</header>
		<div class="text-2xl font-bold">{data.total_count}</div>
		<div class="text-xs text-surface-600-400">
			{[...new Set(data.bonds.map((b) => b.type))].join(', ') || '—'}
		</div>
	</div>
</div>

<div class="space-y-4">
	{#if bondForm.open}
		<div class="card preset-filled-surface-100-900 p-4 space-y-4">
			<header>
				<h3 class="h3 flex items-center gap-2">
					{#if bondForm.isEditing}
						<Pencil size={20} /> Edytuj obligację
					{:else}
						<Plus size={20} /> Nowa obligacja
					{/if}
				</h3>
			</header>
			<form
				class="space-y-4"
				onsubmit={(event) => {
					event.preventDefault();
					handleSubmit();
				}}
			>
				{#if bondForm.error}
					<div class="card preset-filled-error-500 p-3 text-sm">{bondForm.error}</div>
				{/if}

				<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
					<label class="label">
						<span class="font-semibold text-sm">Typ</span>
						<select class="select" bind:value={formData.type} onchange={applyTypeDefaults}>
							{#each bondTypes as t}
								<option value={t.value}>{t.label}</option>
							{/each}
						</select>
					</label>

					<label class="label">
						<span class="font-semibold text-sm">Seria</span>
						<div class="flex gap-2">
							<input
								type="text"
								class="input"
								bind:value={formData.series}
								required
								placeholder="np. EDO0535"
							/>
							<button
								type="button"
								class="btn preset-tonal-surface whitespace-nowrap"
								onclick={lookupRate}
								disabled={lookingUp || !formData.series || !formData.type}
								title="Pobierz stopę z obligacjeskarbowe.pl"
							>
								{lookingUp ? '…' : 'Pobierz stopę'}
							</button>
						</div>
					</label>

					<label class="label">
						<span class="font-semibold text-sm">Wartość nominalna (PLN)</span>
						<input
							type="number"
							class="input"
							bind:value={formData.face_value}
							step="0.01"
							min="0.01"
							required
						/>
					</label>

					<label class="label">
						<span class="font-semibold text-sm">Data zakupu</span>
						<input type="date" class="input" bind:value={formData.purchase_date} required />
					</label>

					<label class="label">
						<span class="font-semibold text-sm">Właściciel</span>
						<select class="select" bind:value={formData.owner_user_id}>
							<option value={null}>Wspólne</option>
							{#each data.owners as o}
								<option value={o.id}>{o.name}</option>
							{/each}
						</select>
					</label>

					<label class="label">
						<span class="font-semibold text-sm">Konto</span>
						<select class="select" bind:value={formData.account_id}>
							<option value={null}>—</option>
							{#each data.accounts as a}
								<option value={a.id}>{a.name}</option>
							{/each}
						</select>
					</label>

					<label class="label">
						<span class="font-semibold text-sm">Stopa rok 1 (%)</span>
						<input
							type="number"
							class="input"
							bind:value={formData.first_year_rate}
							step="0.01"
							min="0"
							max="100"
							required
						/>
					</label>

					<label class="label">
						<span class="font-semibold text-sm">Marża nad CPI (%)</span>
						<input
							type="number"
							class="input"
							bind:value={formData.margin}
							step="0.01"
							min="0"
							max="100"
							required
						/>
					</label>

					<label class="label flex items-end gap-2">
						<input type="checkbox" class="checkbox" bind:checked={formData.capitalize} />
						<span class="font-semibold text-sm">Kapitalizacja odsetek</span>
					</label>
				</div>

				<div class="flex flex-col-reverse sm:flex-row sm:justify-end gap-2">
					<button
						type="button"
						class="btn preset-tonal-surface"
						onclick={cancelForm}
						disabled={bondForm.saving}>Anuluj</button
					>
					<button type="submit" class="btn preset-filled-primary-500" disabled={bondForm.saving}>
						{bondForm.saving
							? 'Zapisywanie...'
							: bondForm.isEditing
								? 'Zapisz zmiany'
								: 'Dodaj obligację'}
					</button>
				</div>
			</form>
		</div>
	{/if}

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header>
			<h3 class="h3 flex items-center gap-2"><Banknote size={20} /> Obligacje</h3>
		</header>

		{#if data.bonds.length === 0}
			<div class="text-center py-12 text-surface-700-300">
				<p>Brak obligacji</p>
			</div>
		{:else}
			<div class="table-wrap">
				<table class="table table-hover">
					<thead>
						<tr>
							<th>Typ</th>
							<th>Seria</th>
							<th>Właściciel</th>
							<th>Konto</th>
							<th>Nominał</th>
							<th>Wartość</th>
							<th>Stopa roku</th>
							<th>Zakup</th>
							<th>Wykup</th>
							<th class="text-right">Akcje</th>
						</tr>
					</thead>
					<tbody>
						{#each data.bonds as bond}
							<tr>
								<td class="font-medium">{bond.type}</td>
								<td>{bond.series}</td>
								<td>{ownerName(data.owners, bond.owner_user_id)}</td>
								<td class="text-xs text-surface-700-300">
									{data.accounts.find((a) => a.id === bond.account_id)?.name ?? '—'}
								</td>
								<td>{formatPLN(bond.face_value)}</td>
								<td class="font-semibold text-primary-600-400">{formatPLN(bond.current_value)}</td>
								<td>{bond.current_yield.toFixed(2)}%</td>
								<td>{bond.purchase_date}</td>
								<td>{bond.maturity_date}</td>
								<td class="text-right whitespace-nowrap">
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Pokaż YTM"
										onclick={() => (selectedBondId = bond.id)}
									>
										<TrendingUp size={16} />
									</button>
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Edytuj"
										onclick={() => startEdit(bond)}
									>
										<Pencil size={16} />
									</button>
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Usuń"
										onclick={() => handleDelete(bond.id)}
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

	<BondsMaturityLadder
		events={data.ladder.events}
		nextMaturity={data.ladder.next_maturity}
		taxRatePct={data.ladder.tax_rate_pct}
	/>

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
			<h3 class="h3 flex items-center gap-2"><TrendingUp size={20} /> Projekcja YTM</h3>
			{#if data.bonds.length > 0}
				<label class="label flex items-center gap-2">
					<span class="text-sm">Obligacja:</span>
					<select class="select" bind:value={selectedBondId}>
						{#each data.bonds as bond}
							<option value={bond.id}>{bond.type} · {bond.series}</option>
						{/each}
					</select>
				</label>
			{/if}
		</header>

		{#if ytmError}
			<div class="card preset-filled-error-500 p-3 text-sm">{ytmError}</div>
		{/if}
		{#if data.bonds.length === 0}
			<p class="text-surface-700-300 text-sm">
				Dodaj obligację aby zobaczyć projekcję wartości do wykupu.
			</p>
		{:else}
			<div bind:this={chartContainer} class="w-full h-[360px]"></div>
			{#if ytmLoading}
				<p class="text-xs text-surface-700-300">Ładowanie projekcji…</p>
			{/if}
		{/if}
	</div>
</div>

<Modal
	open={showDeleteModal}
	title="Potwierdzenie usunięcia"
	onConfirm={confirmDelete}
	onCancel={cancelDelete}
	confirmText="Usuń"
	confirmVariant="danger"
>
	<p class="mb-2">Czy na pewno chcesz usunąć tę obligację?</p>
	<p class="text-sm text-surface-700-300">Operacja ustawi obligację jako nieaktywną.</p>
</Modal>
