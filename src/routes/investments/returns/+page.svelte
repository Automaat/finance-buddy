<script lang="ts">
	import ContributionAdjustedReturns from '$lib/components/ContributionAdjustedReturns.svelte';
	import type { PageData } from './$types';

	interface Props {
		data: PageData;
	}

	let { data }: Props = $props();

	type ScopeKind = 'all' | 'category' | 'wrapper' | 'account';

	// Fixed scope presets (categories + wrappers); per-account options come from
	// the loaded investment accounts. Selection is local state — the returns card
	// re-fetches reactively on scope change, no page reload needed.
	const PRESETS: { kind: ScopeKind; value?: string; label: string }[] = [
		{ kind: 'all', label: 'Całość' },
		{ kind: 'category', value: 'stock', label: 'Akcje' },
		{ kind: 'category', value: 'etf', label: 'ETF' },
		{ kind: 'category', value: 'bond', label: 'Obligacje' },
		{ kind: 'category', value: 'fund', label: 'Fundusze' },
		{ kind: 'wrapper', value: 'IKE', label: 'IKE' },
		{ kind: 'wrapper', value: 'IKZE', label: 'IKZE' },
		{ kind: 'wrapper', value: 'PPK', label: 'PPK' }
	];

	let selectedKind = $state<ScopeKind>('all');
	let selectedValue = $state<string | undefined>(undefined);
	let selectedAccountId = $state<number | undefined>(undefined);

	function selectPreset(kind: ScopeKind, value?: string) {
		selectedKind = kind;
		selectedValue = value;
		selectedAccountId = undefined;
	}

	function selectAccount(event: Event) {
		const raw = (event.currentTarget as HTMLSelectElement).value;
		if (raw === '') {
			selectPreset('all');
			return;
		}
		selectedKind = 'account';
		selectedValue = undefined;
		selectedAccountId = Number(raw);
	}

	const isPresetActive = (kind: ScopeKind, value?: string): boolean =>
		selectedKind === kind && selectedValue === value && selectedAccountId === undefined;

	const scope = $derived({
		type: selectedKind,
		value: selectedValue,
		account_id: selectedAccountId
	});

	const scopeTitle = $derived.by(() => {
		if (selectedKind === 'account') {
			return data.accounts.find((a) => a.id === selectedAccountId)?.name ?? 'Konto';
		}
		return (
			PRESETS.find((p) => p.kind === selectedKind && p.value === selectedValue)?.label ?? 'Zwroty'
		);
	});
</script>

<svelte:head>
	<title>Zwroty - Finance Buddy</title>
</svelte:head>

<div class="space-y-4">
	<div role="group" aria-label="Zakres zwrotów" class="flex flex-wrap gap-2">
		{#each PRESETS as preset (preset.kind + (preset.value ?? ''))}
			<button
				type="button"
				class="btn btn-sm {isPresetActive(preset.kind, preset.value)
					? 'preset-filled-primary-500'
					: 'preset-tonal-surface'}"
				aria-pressed={isPresetActive(preset.kind, preset.value)}
				onclick={() => selectPreset(preset.kind, preset.value)}
			>
				{preset.label}
			</button>
		{/each}

		{#if data.accounts.length > 0}
			<label class="label inline-flex items-center gap-2">
				<span class="sr-only">Konto</span>
				<select
					class="select w-48"
					value={selectedKind === 'account' ? String(selectedAccountId) : ''}
					onchange={selectAccount}
				>
					<option value="">Pojedyncze konto…</option>
					{#each data.accounts as account (account.id)}
						<option value={String(account.id)}>{account.name}</option>
					{/each}
				</select>
			</label>
		{/if}
	</div>

	<ContributionAdjustedReturns {scope} title={scopeTitle} />
</div>
