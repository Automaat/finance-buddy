<script lang="ts">
	import SalaryCalculator from '#lib/components/SalaryCalculator.svelte';
	import type { SalaryRecord } from '#lib/types/salaries.ts';
	import type { OwnerOption } from '#lib/types/owners.ts';

	interface Props {
		data: { latestSalaries: SalaryRecord[]; owners: OwnerOption[] };
	}

	let { data }: Props = $props();
</script>

<SalaryCalculator {data} />
