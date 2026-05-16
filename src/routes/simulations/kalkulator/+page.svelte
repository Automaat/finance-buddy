<script lang="ts">
	import SalaryCalculator from '$lib/components/SalaryCalculator.svelte';
	import type { SalaryRecord } from '$lib/types/salaries';

	export let data: { latestSalaries: SalaryRecord[] };
</script>

<SalaryCalculator {data} />
