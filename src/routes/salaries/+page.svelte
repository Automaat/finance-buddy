<script lang="ts">
	import { onMount } from 'svelte';
	import * as echarts from 'echarts';
	import type { EChartsOption } from 'echarts';
	import Card from '$lib/components/Card.svelte';
	import CardHeader from '$lib/components/CardHeader.svelte';
	import CardTitle from '$lib/components/CardTitle.svelte';
	import CardContent from '$lib/components/CardContent.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import { formatPLN } from '$lib/utils/format';
	import { env } from '$env/dynamic/public';
	import { goto, invalidateAll } from '$app/navigation';
	import type { SalaryRecord } from '$lib/types/salaries';

	export let data;

	const apiUrl = env.PUBLIC_API_URL_BROWSER || 'http://localhost:8000';
	const defaultOwner = env.PUBLIC_DEFAULT_OWNER || 'Marcin';

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

		// Group salary records by company for multi-line chart
		const companyMap = new Map<string, Array<[string, number]>>();

		data.salaries.salary_records.forEach((r) => {
			if (!companyMap.has(r.company)) {
				companyMap.set(r.company, []);
			}
			companyMap.get(r.company)!.push([r.date, r.gross_amount]);
		});

		// Sort each company's data by date
		companyMap.forEach((data) => {
			data.sort((a, b) => new Date(a[0]).getTime() - new Date(b[0]).getTime());
		});

		// Color palette for different companies
		const colors = [
			'#5E81AC', // nord10 - blue
			'#88C0D0', // nord8 - light blue
			'#A3BE8C', // nord14 - green
			'#EBCB8B', // nord13 - yellow
			'#D08770', // nord12 - orange
			'#B48EAD', // nord15 - purple
			'#BF616A' // nord11 - red
		];

		// Create series for each company
		const series: Array<{
			name: string;
			data: Array<[string, number]>;
			type: 'line';
			smooth: boolean;
			lineStyle: { color: string; width: number };
		}> = [];
		let colorIndex = 0;
		companyMap.forEach((data, company) => {
			series.push({
				name: company,
				data: data,
				type: 'line' as const,
				smooth: true,
				lineStyle: { color: colors[colorIndex % colors.length], width: 2 }
			});
			colorIndex++;
		});

		const option: EChartsOption = {
			title: {
				text: 'Progresja wynagrodzenia',
				left: 'center'
			},
			tooltip: {
				trigger: 'axis',
				formatter: (params: any) => {
					let result = `${new Date(params[0].value[0]).toLocaleDateString('pl-PL')}<br/>`;
					params.forEach((p: any) => {
						result += `${p.seriesName}: ${formatPLN(p.value[1])}<br/>`;
					});
					return result;
				}
			},
			legend: {
				top: 30,
				data: series.map((s) => s.name)
			},
			xAxis: {
				type: 'time'
			},
			yAxis: {
				type: 'value',
				axisLabel: {
					formatter: (value: number) => formatPLN(value)
				}
			},
			series,
			grid: {
				left: '80px',
				right: '40px',
				top: '80px'
			}
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

<div class="page-header">
	<div>
		<h1 class="page-title">Historia wynagrodzeń</h1>
		<p class="page-description">Śledź zmiany wynagrodzenia w czasie</p>
	</div>
	<button class="btn btn-primary" on:click={openNewSalaryModal}>+ Nowe Wynagrodzenie</button>
</div>

<Card>
	<CardHeader>
		<CardTitle>💰 Aktualne wynagrodzenia</CardTitle>
	</CardHeader>
	<CardContent>
		<div class="current-salaries">
			<div class="salary-item">
				<span class="salary-label">Marcin:</span>
				<strong class="salary-value">
					{data.salaries.current_salary_marcin !== null
						? formatPLN(data.salaries.current_salary_marcin)
						: 'Brak danych'}
				</strong>
			</div>
			<div class="salary-item">
				<span class="salary-label">Ewa:</span>
				<strong class="salary-value">
					{data.salaries.current_salary_ewa !== null
						? formatPLN(data.salaries.current_salary_ewa)
						: 'Brak danych'}
				</strong>
			</div>
		</div>
	</CardContent>
</Card>

<Card>
	<CardHeader>
		<CardTitle>📈 Progresja wynagrodzenia</CardTitle>
	</CardHeader>
	<CardContent>
		<div bind:this={chartContainer} style="width: 100%; height: 400px;"></div>
	</CardContent>
</Card>

<Card>
	<CardHeader>
		<CardTitle>🔍 Filtry</CardTitle>
	</CardHeader>
	<CardContent>
		<form class="filters-form" on:submit|preventDefault={applyFilters}>
			<div class="filters-row">
				<div class="form-group">
					<label for="filter-owner">Właściciel</label>
					<select id="filter-owner" bind:value={filterOwner}>
						<option value="">Wszystkie</option>
						<option value="Marcin">Marcin</option>
						<option value="Ewa">Ewa</option>
					</select>
				</div>

				<div class="form-group">
					<label for="filter-date-from">Data od</label>
					<input type="date" id="filter-date-from" bind:value={filterDateFrom} />
				</div>

				<div class="form-group">
					<label for="filter-date-to">Data do</label>
					<input type="date" id="filter-date-to" bind:value={filterDateTo} />
				</div>
			</div>

			<div class="filters-actions">
				<button type="submit" class="btn btn-primary">Filtruj</button>
				<button type="button" class="btn btn-secondary" on:click={clearFilters}>
					Wyczyść filtry
				</button>
			</div>
		</form>
	</CardContent>
</Card>

<Card>
	<CardHeader>
		<CardTitle>📊 Historia zmian</CardTitle>
	</CardHeader>
	<CardContent>
		{#if data.salaries.salary_records.length === 0}
			<div class="empty-state">
				<p>Brak rekordów wynagrodzeń</p>
			</div>
		{:else}
			<div class="table-container">
				<table class="transactions-table">
					<thead>
						<tr>
							<th>Data zmiany</th>
							<th>Właściciel</th>
							<th>Firma</th>
							<th>Pensja brutto</th>
							<th>Rodzaj umowy</th>
							<th>Akcje</th>
						</tr>
					</thead>
					<tbody>
						{#each data.salaries.salary_records as record}
							<tr>
								<td>{new Date(record.date).toLocaleDateString('pl-PL')}</td>
								<td>{record.owner}</td>
								<td>{record.company}</td>
								<td class="value-cell">{formatPLN(record.gross_amount)}</td>
								<td>{record.contract_type}</td>
								<td class="actions-cell">
									<button
										class="btn-icon"
										on:click={() => openEditSalaryModal(record)}
										title="Edytuj"
									>
										✏️
									</button>
									<button class="btn-icon" on:click={() => deleteSalary(record.id)} title="Usuń">
										🗑️
									</button>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		{/if}
	</CardContent>
</Card>

<Modal
	open={showNewSalaryModal}
	title={editingSalary ? 'Edytuj wynagrodzenie' : 'Nowe wynagrodzenie'}
	onConfirm={saveSalary}
	onCancel={closeSalaryModal}
	confirmText={savingSalary ? 'Zapisywanie...' : 'Zapisz'}
	confirmDisabled={savingSalary}
	confirmVariant="primary"
>
	<form on:submit|preventDefault={saveSalary} class="salary-form">
		{#if salaryError}
			<div class="error-message">{salaryError}</div>
		{/if}

		<div class="form-group">
			<label for="salary-date">Data zmiany*</label>
			<input type="date" id="salary-date" bind:value={salaryFormData.date} required />
		</div>

		<div class="form-group">
			<label for="salary-amount">Pensja brutto (PLN)*</label>
			<input
				type="number"
				id="salary-amount"
				bind:value={salaryFormData.gross_amount}
				min="0"
				step="0.01"
				required
			/>
		</div>

		<div class="form-group">
			<label for="salary-contract">Rodzaj umowy*</label>
			<select id="salary-contract" bind:value={salaryFormData.contract_type} required>
				<option value="UOP">UOP</option>
				<option value="UZ">UZ</option>
				<option value="UoD">UoD</option>
				<option value="B2B">B2B</option>
			</select>
		</div>

		<div class="form-group">
			<label for="salary-company">Firma*</label>
			<input
				type="text"
				id="salary-company"
				bind:value={salaryFormData.company}
				placeholder="Nazwa firmy"
				required
			/>
		</div>

		<div class="form-group">
			<label for="salary-owner">Właściciel*</label>
			<select id="salary-owner" bind:value={salaryFormData.owner} required>
				<option value="Marcin">Marcin</option>
				<option value="Ewa">Ewa</option>
			</select>
		</div>
	</form>
</Modal>

<style>
	.page-header {
		display: flex;
		justify-content: space-between;
		align-items: flex-start;
		margin-bottom: var(--size-6);
	}

	.page-title {
		font-size: var(--font-size-6);
		font-weight: var(--font-weight-7);
		color: var(--color-text);
		margin: 0 0 var(--size-2) 0;
	}

	.page-description {
		color: var(--color-text-secondary);
		font-size: var(--font-size-2);
		margin: 0;
	}

	.current-salaries {
		display: flex;
		gap: var(--size-6);
		flex-wrap: wrap;
	}

	.salary-item {
		display: flex;
		align-items: center;
		gap: var(--size-2);
	}

	.salary-label {
		font-size: var(--font-size-1);
		color: var(--color-text-2);
	}

	.salary-value {
		font-size: var(--font-size-3);
		color: var(--color-text-1);
	}

	.filters-form {
		display: flex;
		flex-direction: column;
		gap: var(--size-5);
	}

	.filters-row {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: var(--size-4);
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: var(--size-2);
	}

	.form-group label {
		font-weight: var(--font-weight-6);
		color: var(--color-text);
		font-size: var(--font-size-2);
	}

	.form-group input,
	.form-group select {
		padding: var(--size-3);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-2);
		background: var(--color-background);
		color: var(--color-text);
		font-size: var(--font-size-2);
	}

	.form-group input:focus,
	.form-group select:focus {
		outline: none;
		border-color: var(--color-primary);
	}

	.filters-actions {
		display: flex;
		gap: var(--size-3);
	}

	.btn {
		padding: var(--size-3) var(--size-5);
		border: none;
		border-radius: var(--radius-2);
		font-weight: var(--font-weight-6);
		font-size: var(--font-size-2);
		cursor: pointer;
		transition: all 0.2s;
	}

	.btn-primary {
		background: var(--color-primary);
		color: var(--nord6);
	}

	.btn-primary:hover {
		background: var(--nord9);
	}

	.btn-secondary {
		background: var(--color-surface);
		color: var(--color-text);
		border: 1px solid var(--color-border);
	}

	.btn-secondary:hover {
		background: var(--color-accent);
	}

	.btn-icon {
		background: transparent;
		border: none;
		cursor: pointer;
		font-size: var(--font-size-3);
		padding: var(--size-2);
		transition: transform 0.2s;
	}

	.btn-icon:hover {
		transform: scale(1.2);
	}

	.empty-state {
		text-align: center;
		padding: var(--size-8) var(--size-4);
		color: var(--color-text-secondary);
	}

	.table-container {
		overflow-x: auto;
	}

	.transactions-table {
		width: 100%;
		border-collapse: collapse;
	}

	.transactions-table thead {
		border-bottom: 2px solid var(--color-border);
	}

	.transactions-table th {
		text-align: left;
		padding: var(--size-3) var(--size-4);
		font-weight: var(--font-weight-6);
		color: var(--color-text);
		font-size: var(--font-size-2);
	}

	.transactions-table tbody tr {
		border-bottom: 1px solid var(--color-border);
		transition: background-color 0.2s;
	}

	.transactions-table tbody tr:hover {
		background-color: var(--color-accent);
	}

	.transactions-table td {
		padding: var(--size-4);
		font-size: var(--font-size-2);
	}

	.value-cell {
		font-weight: var(--font-weight-6);
		color: var(--color-primary);
	}

	.actions-cell {
		text-align: right;
		display: flex;
		gap: var(--size-2);
		justify-content: flex-end;
	}

	.salary-form {
		display: flex;
		flex-direction: column;
		gap: var(--size-4);
	}

	.error-message {
		padding: var(--size-3);
		background: var(--nord11);
		color: var(--nord6);
		border-radius: var(--radius-2);
		font-size: var(--font-size-2);
	}

	@media (max-width: 1024px) {
		.filters-row {
			grid-template-columns: repeat(2, 1fr);
		}
	}

	@media (max-width: 768px) {
		.filters-row {
			grid-template-columns: 1fr;
		}
	}
</style>
