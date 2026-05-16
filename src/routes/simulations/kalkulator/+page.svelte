<script lang="ts">
	import SalaryCalculator from '$lib/components/SalaryCalculator.svelte';
	import type { SalaryRecord } from '$lib/types/salaries';

	interface Props {
		data: { latestSalaries: SalaryRecord[] };
	}

	let { data }: Props = $props();
</script>

<SalaryCalculator {data} />
