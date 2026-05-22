<script lang="ts">
	import SalaryCalculator from '$lib/components/SalaryCalculator.svelte';
	import type { SalaryRecord } from '$lib/types/salaries';
	import type { OwnerOption } from '$lib/types/owners';

	interface Props {
		data: { latestSalaries: SalaryRecord[]; owners: OwnerOption[] };
	}

	let { data }: Props = $props();
</script>

<SalaryCalculator {data} />
