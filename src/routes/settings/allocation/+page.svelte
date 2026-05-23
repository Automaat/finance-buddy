<script lang="ts">
	import { resolveApiUrl } from '$lib/api';
	import { invalidateAll } from '$app/navigation';
	import { toast } from '$lib/stores/toast.svelte';
	import { ownerName, type OwnerOption } from '$lib/types/owners';
	import type { PageData } from './$types';

	interface Target {
		id: number;
		category: string;
		owner_user_id: number | null;
		target_pct: number;
		created_at: string;
	}

	type Scope = number | null;

	const CATEGORIES = [
		{ value: 'bank', label: 'Bank' },
		{ value: 'saving_account', label: 'Konto oszczędnościowe' },
		{ value: 'stock', label: 'Akcje' },
		{ value: 'bond', label: 'Obligacje' },
		{ value: 'gold', label: 'Złoto' },
		{ value: 'real_estate', label: 'Nieruchomości' },
		{ value: 'ppk', label: 'PPK' },
		{ value: 'fund', label: 'Fundusze' },
		{ value: 'etf', label: 'ETF' },
		{ value: 'vehicle', label: 'Pojazdy' }
	];

	let { data }: { data: PageData } = $props();

	const targets = $derived(data.targets.targets as Target[]);
	const owners = $derived((data.owners ?? []) as OwnerOption[]);
	const apiUrl = resolveApiUrl();

	let selectedScope: Scope = $state(null);

	const scopeTargets = $derived(targets.filter((t) => t.owner_user_id === selectedScope));

	let draft = $state<{ category: string; target_pct: number }[]>([]);
	let dirty = $state(false);
	let saving = $state(false);

	$effect(() => {
		const list = targets
			.filter((t) => t.owner_user_id === selectedScope)
			.map((t) => ({ category: t.category, target_pct: t.target_pct }));
		draft = list;
		dirty = false;
	});

	const draftSum = $derived(draft.reduce((acc, row) => acc + (Number(row.target_pct) || 0), 0));
	const sumOk = $derived(Math.abs(draftSum - 100) < 0.01 || draft.length === 0);

	function addRow(): void {
		const used = new Set(draft.map((d) => d.category));
		const next = CATEGORIES.find((c) => !used.has(c.value));
		if (!next) {
			toast.error('Wszystkie kategorie są już dodane');
			return;
		}
		draft = [...draft, { category: next.value, target_pct: 0 }];
		dirty = true;
	}

	function removeRow(index: number): void {
		draft = draft.filter((_, i) => i !== index);
		dirty = true;
	}

	function markDirty(): void {
		dirty = true;
	}

	async function save(): Promise<void> {
		if (!sumOk) {
			toast.error(`Suma musi wynosić 100% (aktualnie: ${draftSum.toFixed(2)}%)`);
			return;
		}
		const used = new Set<string>();
		for (const row of draft) {
			if (used.has(row.category)) {
				toast.error(`Kategoria „${row.category}" powtarza się`);
				return;
			}
			used.add(row.category);
		}
		saving = true;
		try {
			const res = await fetch(`${apiUrl}/api/allocation/targets/replace`, {
				method: 'PUT',
				headers: { 'content-type': 'application/json' },
				body: JSON.stringify({
					owner_user_id: selectedScope,
					targets: draft.map((d) => ({
						category: d.category,
						target_pct: Number(d.target_pct)
					}))
				})
			});
			if (!res.ok) {
				const err = await res.json().catch(() => null);
				const detail =
					err && typeof err === 'object' && 'detail' in err && err.detail
						? typeof err.detail === 'string'
							? err.detail
							: JSON.stringify(err.detail)
						: 'Zapis nie powiódł się';
				throw new Error(detail);
			}
			toast.success('Cele alokacji zapisane');
			dirty = false;
			await invalidateAll();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Wystąpił błąd');
		} finally {
			saving = false;
		}
	}

	function scopeLabel(scope: Scope): string {
		if (scope === null) return 'Wspólne (gospodarstwo)';
		return ownerName(owners, scope);
	}

	function categoryLabel(value: string): string {
		return CATEGORIES.find((c) => c.value === value)?.label ?? value;
	}
</script>

<div class="space-y-6">
	<div class="space-y-1">
		<h1 class="h2">Cele alokacji</h1>
		<p class="text-surface-700-300 text-sm">
			Ustaw docelowy procentowy podział aktywów. Suma musi wynosić 100% w obrębie wybranego zakresu.
		</p>
	</div>

	<div class="card preset-filled-surface-50-950 p-5 space-y-4">
		<label class="label max-w-md">
			<span class="font-semibold text-sm">Zakres</span>
			<select class="select" bind:value={selectedScope}>
				<option value={null}>Wspólne (gospodarstwo)</option>
				{#each owners as owner}
					<option value={owner.id}>{owner.name}</option>
				{/each}
			</select>
		</label>

		<div class="space-y-2">
			<h2 class="h4 font-semibold">{scopeLabel(selectedScope)}</h2>

			<table class="table">
				<thead>
					<tr>
						<th>Kategoria</th>
						<th>Docelowy %</th>
						<th class="w-20"></th>
					</tr>
				</thead>
				<tbody>
					{#each draft as row, idx (idx)}
						<tr>
							<td>
								<select class="select" bind:value={row.category} onchange={markDirty}>
									{#each CATEGORIES as cat}
										<option value={cat.value}>{cat.label}</option>
									{/each}
								</select>
							</td>
							<td>
								<input
									type="number"
									class="input"
									step="0.01"
									min="0"
									max="100"
									bind:value={row.target_pct}
									oninput={markDirty}
								/>
							</td>
							<td>
								<button
									type="button"
									class="btn btn-sm preset-tonal-error"
									onclick={() => removeRow(idx)}
								>
									Usuń
								</button>
							</td>
						</tr>
					{/each}
					{#if draft.length === 0}
						<tr>
							<td colspan="3" class="text-surface-700-300 italic">
								Brak celów — dodaj pierwszą kategorię.
							</td>
						</tr>
					{/if}
				</tbody>
				<tfoot>
					<tr>
						<th class="text-right">Suma</th>
						<th class={sumOk ? 'text-success-600-400 font-bold' : 'text-error-600-400 font-bold'}>
							{draftSum.toFixed(2)}%
						</th>
						<th></th>
					</tr>
				</tfoot>
			</table>

			<div class="flex flex-wrap gap-2">
				<button
					type="button"
					class="btn preset-tonal-surface"
					onclick={addRow}
					disabled={draft.length >= CATEGORIES.length}
				>
					Dodaj kategorię
				</button>
				<button
					type="button"
					class="btn preset-filled-primary-500"
					onclick={save}
					disabled={saving || !dirty || !sumOk}
				>
					{saving ? 'Zapisywanie...' : 'Zapisz'}
				</button>
				{#if !sumOk && draft.length > 0}
					<span class="text-error-600-400 text-sm self-center"
						>Suma musi wynosić dokładnie 100%.</span
					>
				{/if}
			</div>
		</div>

		{#if scopeTargets.length > 0}
			<p class="text-xs text-surface-700-300">
				Aktualnie zapisanych celów dla tego zakresu: {scopeTargets.length}.
			</p>
		{/if}
	</div>

	<div class="card preset-filled-surface-100-900 p-4 space-y-2">
		<h2 class="h4">Jak to działa?</h2>
		<ul class="text-sm text-surface-700-300 list-disc pl-6 space-y-1">
			<li>
				Cele można definiować osobno dla całego gospodarstwa (Wspólne) i dla każdego użytkownika.
			</li>
			<li>
				Suma celów musi wynosić dokładnie 100% (z tolerancją 0.01). Zapis bez tej sumy jest
				zablokowany.
			</li>
			<li>
				Widget „Dryft alokacji" na dashboardzie pokazuje porównanie celu z aktualnym stanem oraz
				sugerowane kwoty rebalansu, gdy odchylenie przekracza ±5 punktów procentowych.
			</li>
			<li>
				Nieoznaczone celami kategorie aktywów (np. spadek lub nowa inwestycja) pojawią się w
				widgecie jako „brak celu" — dopisz je tutaj, aby były uwzględnione w rebalansie.
			</li>
		</ul>
	</div>
</div>
