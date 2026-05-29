<script lang="ts">
	import Modal from '$lib/components/Modal.svelte';
	import { formatPLN, formatDate } from '$lib/utils/format';
	import { Target, Plus, Pencil, Trash2, CheckCircle2, Calendar } from 'lucide-svelte';
	import { api } from '$lib/apiClient';
	import { invalidateAll } from '$app/navigation';
	import { CrudForm } from '$lib/stores/crudForm.svelte';
	import type { Goal, AccountOption } from './+page';
	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	const goals = $derived(data.goals as Goal[]);
	const accounts = $derived(data.accounts as AccountOption[]);
	const totalCount = $derived(data.total_count as number);
	const completedCount = $derived(data.completed_count as number);

	const goalForm = new CrudForm<Goal>();
	let showDeleteModal = $state(false);
	let goalToDelete: number | null = $state(null);

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
		const editing = goalForm.editing;
		if (editing) {
			formData = {
				name: editing.name,
				target_amount: editing.target_amount,
				target_date: editing.target_date,
				current_amount: editing.current_amount,
				monthly_contribution: editing.monthly_contribution,
				is_completed: editing.is_completed,
				account_id: editing.account_id,
				category: editing.category
			};
		} else if (goalForm.open) {
			formData = emptyForm();
		}
	});

	function startCreate() {
		goalForm.openCreate();
	}

	function startEdit(goal: Goal) {
		goalForm.openEdit(goal);
	}

	function cancelForm() {
		goalForm.close();
	}

	async function handleSubmit() {
		const editing = goalForm.editing;
		await goalForm.submit(async () => {
			if (editing) {
				await api.put(`/api/goals/${editing.id}`, formData);
			} else {
				await api.post('/api/goals', formData);
			}
			await invalidateAll();
		});
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
			await api.del(`/api/goals/${goalToDelete}`);
			await invalidateAll();
		} catch (err) {
			if (err instanceof Error) goalForm.error = err.message;
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

	{#if goalForm.error}
		<div class="card preset-tonal-error-500 p-3 text-sm" role="alert">{goalForm.error}</div>
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
	open={goalForm.open}
	title={goalForm.isEditing ? 'Edytuj cel' : 'Nowy cel'}
	confirmText={goalForm.saving ? 'Zapisywanie...' : 'Zapisz'}
	confirmDisabled={goalForm.saving || !formData.name || formData.target_amount <= 0}
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
		{#if goalForm.error}
			<div class="card preset-tonal-error-500 p-2 text-sm" role="alert">{goalForm.error}</div>
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
