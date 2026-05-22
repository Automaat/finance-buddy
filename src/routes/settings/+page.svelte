<script lang="ts">
	import { Users } from 'lucide-svelte';
</script>

<svelte:head>
	<title>Ustawienia | Finansowa Forteca</title>
</svelte:head>

<div class="mb-6 space-y-1">
	<h1 class="h2">Ustawienia</h1>
	<p class="text-surface-700-300 text-sm">Zarządzaj preferencjami aplikacji</p>
</div>

<div class="card preset-filled-surface-100-900 p-4 space-y-3">
	<header class="flex items-center gap-2">
		<Users size={20} />
		<h3 class="h3">Właściciele</h3>
	</header>
	<p class="text-sm text-surface-700-300">
		Właściciele kont i ich składki PPK są zarządzane jako konta użytkowników. Administrator może
		dodawać i edytować użytkowników w panelu kont użytkowników.
	</p>
</div>
