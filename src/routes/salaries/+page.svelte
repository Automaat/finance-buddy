<script lang="ts">
	import { onMount } from 'svelte';
	import * as echarts from 'echarts';
	import type { EChartsOption } from 'echarts';
	import Modal from '$lib/components/Modal.svelte';
	import { formatPLN } from '$lib/utils/format';
	import { Plus, Banknote, TrendingUp, Search, BarChart3, Pencil, Trash2 } from 'lucide-svelte';
	import { env } from '$env/dynamic/public';
	import { goto, invalidateAll } from '$app/navigation';
	import type { SalaryRecord } from '$lib/types/salaries';
	import type { Persona } from '$lib/types/personas';

	export let data;

	const apiUrl = env.PUBLIC_API_URL_BROWSER || 'http://localhost:8000';
	$: personas = data.personas as Persona[];
	$: defaultOwner = personas.length > 0 ? personas[0].name : 'Marcin';

	let chartContainer: HTMLDivElement;

	let filterOwner = data.filters.owner || '';
	let filterDateFrom = data.filters.date_from || '';
	let filterDateTo = data.filters.date_to || '';

	let showNewSalaryModal = false;
	let editingSalary: SalaryRecord | null = null;
	let salaryFormData = {
		date: new Date().toISOString().split('T')[0],
		gross_amount: 0,
		contract_type: 'UOP',
		company: '',
		owner: defaultOwner
	};
	let salaryError = '';
	let savingSalary = false;

	function applyFilters() {
		const params = new URLSearchParams();
		if (filterOwner) params.set('owner', filterOwner);
		if (filterDateFrom) params.set('date_from', filterDateFrom);
		if (filterDateTo) params.set('date_to', filterDateTo);

		goto(`/salaries?${params.toString()}`);
	}

	function clearFilters() {
		filterOwner = '';
		filterDateFrom = '';
		filterDateTo = '';
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

	async function saveSalary() {
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
				throw new Error(errorData.detail || 'Failed to save salary record');
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
			alert('Nie udało się usunąć rekordu wynagrodzenia');
		}
	}

	onMount(() => {
		const chart = echarts.init(chartContainer);

		const companyMap = new Map<string, Array<[string, number]>>();

		data.salaries.salary_records.forEach((r) => {
			const companyName = (r.company ?? '').trim() || 'Nieokreślona firma';
			if (!companyMap.has(companyName)) {
				companyMap.set(companyName, []);
			}
			companyMap.get(companyName)!.push([r.date, r.gross_amount]);
		});

		companyMap.forEach((salaryData) => {
			salaryData.sort((a, b) => new Date(a[0]).getTime() - new Date(b[0]).getTime());
		});

		const colors = ['#5E81AC', '#88C0D0', '#A3BE8C', '#EBCB8B', '#D08770', '#B48EAD', '#BF616A'];

		const series: Array<{
			name: string;
			data: Array<[string, number]>;
			type: 'line';
			smooth: boolean;
			lineStyle: { color: string; width: number };
		}> = [];
		let colorIndex = 0;
		companyMap.forEach((salaryData, company) => {
			series.push({
				name: company,
				data: salaryData,
				type: 'line' as const,
				smooth: true,
				lineStyle: { color: colors[colorIndex % colors.length], width: 2 }
			});
			colorIndex++;
		});

		const option: EChartsOption = {
			title: { text: 'Progresja wynagrodzenia', left: 'center' },
			tooltip: {
				trigger: 'axis',
				formatter: (params: any) => {
					if (!params || !Array.isArray(params) || params.length === 0) return '';
					let result = `${new Date(params[0].value[0]).toLocaleDateString('pl-PL')}<br/>`;
					params.forEach((p: any) => {
						result += `${p.seriesName}: ${formatPLN(p.value[1])}<br/>`;
					});
					return result;
				}
			},
			legend: { top: 30, data: series.map((s) => s.name) },
			xAxis: { type: 'time' },
			yAxis: {
				type: 'value',
				axisLabel: { formatter: (value: number) => formatPLN(value) }
			},
			series,
			grid: { left: '80px', right: '40px', top: '80px' }
		};

		chart.setOption(option);

		return () => {
			chart.dispose();
		};
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
		on:click={openNewSalaryModal}
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
			<h3 class="h3 flex items-center gap-2"><TrendingUp size={20} /> Progresja wynagrodzenia</h3>
		</header>
		<div bind:this={chartContainer} style="width: 100%; height: 400px;"></div>
	</div>

	<div class="card preset-filled-surface-100-900 p-4 space-y-4">
		<header>
			<h3 class="h3 flex items-center gap-2"><Search size={20} /> Filtry</h3>
		</header>
		<form class="space-y-4" on:submit|preventDefault={applyFilters}>
			<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
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
				<button type="button" class="btn preset-tonal-surface" on:click={clearFilters}
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
										on:click={() => openEditSalaryModal(record)}
									>
										<Pencil size={16} />
									</button>
									<button
										type="button"
										class="btn-icon btn-icon-sm"
										aria-label="Usuń"
										on:click={() => deleteSalary(record.id)}
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
	<form on:submit|preventDefault={saveSalary} class="space-y-4">
		{#if salaryError}
			<div class="card preset-filled-error-500 p-3 text-sm">{salaryError}</div>
		{/if}

		<label class="label">
			<span class="font-semibold text-sm">Data zmiany*</span>
			<input type="date" class="input" bind:value={salaryFormData.date} required />
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
