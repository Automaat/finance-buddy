<script lang="ts">
	import { goto } from '$app/navigation';
	import { page } from '$app/stores';
	import {
		RANGE_PRESETS,
		PRESET_LABEL,
		RANGE_LABEL,
		isRangeValue,
		type RangePreset,
		type RangeValue
	} from '$lib/utils/dateRange';

	interface Props {
		// path is the route to navigate to when the user changes selection.
		// Defaults to the current pathname so the picker works on any page.
		path?: string;
	}

	let { path }: Props = $props();

	const targetPath = $derived(path ?? $page.url.pathname);

	const rawRange = $derived($page.url.searchParams.get('range'));
	const explicitFrom = $derived($page.url.searchParams.get('date_from') ?? '');
	const explicitTo = $derived($page.url.searchParams.get('date_to') ?? '');

	const activeRange = $derived<RangeValue>(
		isRangeValue(rawRange) ? rawRange : explicitFrom || explicitTo ? 'custom' : 'all'
	);

	let customFrom = $state('');
	let customTo = $state('');

	$effect(() => {
		// Sync URL → local state. The user can then edit before Apply.
		customFrom = explicitFrom;
		customTo = explicitTo;
	});

	function selectPreset(preset: RangePreset) {
		if (preset === activeRange) return;
		const params = new URLSearchParams($page.url.searchParams);
		params.delete('date_from');
		params.delete('date_to');
		if (preset === 'all') {
			params.delete('range');
		} else {
			params.set('range', preset);
		}
		const qs = params.toString();
		goto(qs ? `${targetPath}?${qs}` : targetPath, { keepFocus: true });
	}

	function selectCustom() {
		const params = new URLSearchParams($page.url.searchParams);
		params.set('range', 'custom');
		if (customFrom) params.set('date_from', customFrom);
		else params.delete('date_from');
		if (customTo) params.set('date_to', customTo);
		else params.delete('date_to');
		goto(`${targetPath}?${params.toString()}`, { keepFocus: true });
	}

	function applyCustom(event: SubmitEvent) {
		event.preventDefault();
		selectCustom();
	}
</script>

<div class="date-range-picker space-y-2" data-testid="date-range-picker">
	<div role="group" aria-label="Zakres dat" class="flex flex-wrap gap-2">
		{#each RANGE_PRESETS as preset (preset)}
			<button
				type="button"
				class="chip preset-tonal-surface"
				class:preset-filled-primary-500={activeRange === preset}
				aria-pressed={activeRange === preset}
				onclick={() => selectPreset(preset)}
			>
				{PRESET_LABEL[preset]}
			</button>
		{/each}
		<button
			type="button"
			class="chip preset-tonal-surface"
			class:preset-filled-primary-500={activeRange === 'custom'}
			aria-pressed={activeRange === 'custom'}
			onclick={() => selectCustom()}
		>
			{RANGE_LABEL.custom}
		</button>
	</div>

	{#if activeRange === 'custom'}
		<form class="flex flex-wrap items-end gap-2" onsubmit={applyCustom}>
			<label class="label">
				<span class="text-xs font-semibold">Od</span>
				<input
					type="date"
					class="input"
					bind:value={customFrom}
					aria-label="Data od"
					max={customTo || undefined}
				/>
			</label>
			<label class="label">
				<span class="text-xs font-semibold">Do</span>
				<input
					type="date"
					class="input"
					bind:value={customTo}
					aria-label="Data do"
					min={customFrom || undefined}
				/>
			</label>
			<button type="submit" class="btn preset-filled-primary-500">Zastosuj</button>
		</form>
	{/if}
</div>

<style>
	.chip {
		padding: 0.25rem 0.75rem;
		border-radius: 9999px;
		font-size: 0.875rem;
		font-weight: 600;
		cursor: pointer;
		transition: background-color 120ms ease;
	}
</style>
