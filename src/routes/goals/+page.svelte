<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import { formatPLN, formatDate } from '$lib/utils/format';
	import { Target, Plus, Pencil, Trash2, CheckCircle2, Calendar } from 'lucide-svelte';
	import { invalidateAll } from '$app/navigation';
	import { getApiUrlOrThrow } from '$lib/utils/api';
	import type { Goal, AccountOption } from './+page';
	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	const apiUrl = () => getApiUrlOrThrow();

	const goals = $derived(data.goals as Goal[]);
	const accounts = $derived(data.accounts as AccountOption[]);
	const totalCount = $derived(data.total_count as number);
	const completedCount = $derived(data.completed_count as number);

	let showForm = $state(false);
	let editingGoal: Goal | null = $state(null);
	let showDeleteModal = $state(false);
	let goalToDelete: number | null = $state(null);
	let error = $state('');
	let saving = $state(false);

	const categoryLabels: Record<string, string> = {
		bank: 'Konto bankowe',
		saving_account: 'Konto oszczędnościowe',
		stock: 'Akcje',
		bond: 'Obligacje',
		gold: 'Złoto',
		real_estate: 'Nieruchomość',
		ppk: 'PPK',
		fund: 'Fundusz',
		etf: 'ETF',
		vehicle: 'Pojazd',
		mortgage: 'Hipoteka',
		installment: 'Raty',
		other: 'Inne'
	};

	const emptyForm = () => ({
		name: '',
		target_amount: 0,
		target_date: new Date().toISOString().split('T')[0],
		current_amount: 0,
		monthly_contribution: 0,
		is_completed: false,
		account_id: null as number | null,
		category: null as string | null
	});

	let formData = $state(emptyForm());

	$effect(() => {
		if (editingGoal) {
			formData = {
				name: editingGoal.name,
				target_amount: editingGoal.target_amount,
				target_date: editingGoal.target_date,
				current_amount: editingGoal.current_amount,
				monthly_contribution: editingGoal.monthly_contribution,
				is_completed: editingGoal.is_completed,
				account_id: editingGoal.account_id,
				category: editingGoal.category
			};
		} else if (showForm) {
			formData = emptyForm();
		}
	});

	function startCreate() {
		editingGoal = null;
		showForm = true;
	}

	function startEdit(goal: Goal) {
		editingGoal = goal;
		showForm = true;
	}

	function cancelForm() {
		showForm = false;
		editingGoal = null;
		error = '';
	}

	async function handleSubmit() {
		error = '';
		saving = true;
		try {
			const endpoint = editingGoal
				? `${apiUrl()}/api/goals/${editingGoal.id}`
				: `${apiUrl()}/api/goals`;
			const method = editingGoal ? 'PUT' : 'POST';
			const response = await fetch(endpoint, {
				method,
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(formData)
			});
			if (!response.ok) {
				const errorData = await response.json();
				throw new Error(errorData.detail || 'Nie udało się zapisać celu');
			}
			await invalidateAll();
			cancelForm();
		} catch (err) {
			if (err instanceof Error) error = err.message;
		} finally {
			saving = false;
		}
	}

	function handleDelete(goalId: number) {
		goalToDelete = goalId;
		showDeleteModal = true;
	}

	function cancelDelete() {
		showDeleteModal = false;
		goalToDelete = null;
	}

	async function confirmDelete() {
		if (!goalToDelete) return;
		try {
			const response = await fetch(`${apiUrl()}/api/goals/${goalToDelete}`, {
				method: 'DELETE'
			});
			if (!response.ok) throw new Error('Nie udało się usunąć celu');
			await invalidateAll();
		} catch (err) {
			if (err instanceof Error) error = err.message;
		} finally {
			showDeleteModal = false;
			goalToDelete = null;
		}
	}

	function progressColor(percent: number, isCompleted: boolean): string {
		if (isCompleted || percent >= 100) return 'bg-success-500';
		if (percent >= 66) return 'bg-primary-500';
		if (percent >= 33) return 'bg-warning-500';
		return 'bg-surface-400-600';
	}

	function polishPlural(n: number, one: string, few: string, many: string): string {
		const mod10 = n % 10;
		const mod100 = n % 100;
		if (n === 1) return one;
		if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return few;
		return many;
	}

	const completedLabel = $derived(
		`${polishPlural(completedCount, 'cel ukończony', 'cele ukończone', 'celów ukończonych')}`
	);
</script>

<div class="space-y-6">
	<header class="flex items-center justify-between gap-4 flex-wrap">
		<div class="flex items-center gap-3">
			<Target class="text-primary-500" size={28} />
			<div>
				<h1 class="h2 font-bold">Cele finansowe</h1>
				<p class="text-sm text-surface-700-300">
					{completedCount} z {totalCount}
					{completedLabel}
				</p>
			</div>
		</div>
		<button type="button" class="btn preset-filled-primary-500" onclick={startCreate}>
			<Plus size={18} />
			<span>Nowy cel</span>
		</button>
	</header>

	{#if error}
		<div class="card preset-tonal-error-500 p-3 text-sm" role="alert">{error}</div>
	{/if}

	{#if goals.length === 0}
		<div class="card preset-tonal-surface p-8 text-center">
			<Target class="mx-auto mb-3 text-surface-500" size={40} />
			<p class="text-surface-700-300">Brak celów. Dodaj pierwszy cel finansowy.</p>
		</div>
	{:else}
		<div class="grid grid-cols-1 md:grid-cols-2 gap-4">
			{#each goals as goal (goal.id)}
				<article class="card preset-tonal-surface p-4 space-y-3">
					<div class="flex items-start justify-between gap-2">
						<div class="min-w-0">
							<h2 class="h5 font-bold flex items-center gap-2">
								{goal.name}
								{#if goal.is_completed}
									<CheckCircle2 class="text-success-500" size={18} />
								{/if}
							</h2>
							<p class="text-xs text-surface-700-300">
								Cel: {formatPLN(goal.target_amount)} do {formatDate(goal.target_date)}
							</p>
						</div>
						<div class="flex gap-1 shrink-0">
							<button
								type="button"
								class="btn-icon btn-icon-sm"
								aria-label="Edytuj"
								onclick={() => startEdit(goal)}
							>
								<Pencil size={16} />
							</button>
							<button
								type="button"
								class="btn-icon btn-icon-sm preset-tonal-error"
								aria-label="Usuń"
								onclick={() => handleDelete(goal.id)}
							>
								<Trash2 size={16} />
							</button>
						</div>
					</div>

					<div>
						<div class="flex justify-between text-sm mb-1">
							<span>{formatPLN(goal.current_amount)}</span>
							<span class="font-mono">{goal.progress_percent.toFixed(1)}%</span>
						</div>
						<div class="h-3 bg-surface-200-800 rounded-full overflow-hidden">
							<div
								class="h-full transition-all {progressColor(
									goal.progress_percent,
									goal.is_completed
								)}"
								style="width: {Math.min(100, goal.progress_percent)}%"
								role="progressbar"
								aria-valuenow={goal.progress_percent}
								aria-valuemin="0"
								aria-valuemax="100"
							></div>
						</div>
						<p class="text-xs text-surface-700-300 mt-1">
							Pozostało: {formatPLN(goal.remaining_amount)}
						</p>
					</div>

					<div class="grid grid-cols-2 gap-2 text-xs">
						<div>
							<span class="text-surface-700-300">Miesięcznie</span>
							<p class="font-semibold">{formatPLN(goal.monthly_contribution)}</p>
						</div>
						<div>
							<span class="text-surface-700-300 flex items-center gap-1">
								<Calendar size={12} />
								Prognoza
							</span>
							<p class="font-semibold">
								{goal.projected_hit_date ? formatDate(goal.projected_hit_date) : '—'}
							</p>
						</div>
						{#if goal.account_name}
							<div class="col-span-2">
								<span class="text-surface-700-300">Konto</span>
								<p class="font-semibold">{goal.account_name}</p>
							</div>
						{/if}
						{#if goal.category}
							<div class="col-span-2">
								<span class="text-surface-700-300">Kategoria</span>
								<p class="font-semibold">{categoryLabels[goal.category] ?? goal.category}</p>
							</div>
						{/if}
					</div>
				</article>
			{/each}
		</div>
	{/if}
</div>

<Modal
	open={showForm}
	title={editingGoal ? 'Edytuj cel' : 'Nowy cel'}
	confirmText={saving ? 'Zapisywanie...' : 'Zapisz'}
	confirmDisabled={saving || !formData.name || formData.target_amount <= 0}
	onConfirm={handleSubmit}
	onCancel={cancelForm}
>
	<form class="space-y-3" onsubmit={(e) => e.preventDefault()}>
		<label class="label">
			<span>Nazwa</span>
			<input class="input" type="text" bind:value={formData.name} required />
		</label>
		<div class="grid grid-cols-2 gap-3">
			<label class="label">
				<span>Cel (PLN)</span>
				<input
					class="input"
					type="number"
					min="0.01"
					step="0.01"
					bind:value={formData.target_amount}
					required
				/>
			</label>
			<label class="label">
				<span>Data celu</span>
				<input class="input" type="date" bind:value={formData.target_date} required />
			</label>
		</div>
		<div class="grid grid-cols-2 gap-3">
			<label class="label">
				<span>Obecna kwota</span>
				<input
					class="input"
					type="number"
					min="0"
					step="0.01"
					bind:value={formData.current_amount}
				/>
			</label>
			<label class="label">
				<span>Wkład miesięczny</span>
				<input
					class="input"
					type="number"
					min="0"
					step="0.01"
					bind:value={formData.monthly_contribution}
				/>
			</label>
		</div>
		<label class="label">
			<span>Konto (opcjonalnie)</span>
			<select class="select" bind:value={formData.account_id}>
				<option value={null}>—</option>
				{#each accounts as account (account.id)}
					<option value={account.id}>{account.name}</option>
				{/each}
			</select>
		</label>
		<label class="label">
			<span>Kategoria (opcjonalnie)</span>
			<select class="select" bind:value={formData.category}>
				<option value={null}>—</option>
				{#each Object.entries(categoryLabels) as [value, label] (value)}
					<option {value}>{label}</option>
				{/each}
			</select>
		</label>
		<label class="flex items-center gap-2 cursor-pointer">
			<input type="checkbox" class="checkbox" bind:checked={formData.is_completed} />
			<span>Cel ukończony</span>
		</label>
		{#if error}
			<div class="card preset-tonal-error-500 p-2 text-sm" role="alert">{error}</div>
		{/if}
	</form>
</Modal>

<Modal
	open={showDeleteModal}
	title="Usunąć cel?"
	confirmText="Usuń"
	confirmVariant="danger"
	onConfirm={confirmDelete}
	onCancel={cancelDelete}
>
	<p>Czy na pewno chcesz usunąć ten cel? Tej operacji nie da się cofnąć.</p>
</Modal>
