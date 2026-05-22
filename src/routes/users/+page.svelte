<script lang="ts">
	import { env } from '$env/dynamic/public';
	import { invalidateAll } from '$app/navigation';
	import { toast } from '$lib/stores/toast.svelte';
	import type { PageData } from './$types';

	interface AppUser {
		id: number;
		username: string;
		is_admin: boolean;
		created_at: string;
	}

	let { data }: { data: PageData } = $props();
	const users = $derived(data.users as AppUser[]);

	let username = $state('');
	let password = $state('');
	let saving = $state(false);

	async function createUser(event: Event): Promise<void> {
		event.preventDefault();
		saving = true;
		try {
			const apiUrl = env.PUBLIC_API_URL_BROWSER ?? '';
			const response = await fetch(`${apiUrl}/api/auth/users`, {
				method: 'POST',
				headers: { 'content-type': 'application/json' },
				body: JSON.stringify({ username, password })
			});
			if (!response.ok) {
				const body = await response.json().catch(() => null);
				const detail =
					body && typeof body === 'object' && 'detail' in body && body.detail
						? String(body.detail)
						: 'Nie udało się utworzyć użytkownika';
				throw new Error(detail);
			}
			toast.success(`Użytkownik „${username}" został utworzony`);
			username = '';
			password = '';
			await invalidateAll();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Wystąpił błąd');
		} finally {
			saving = false;
		}
	}
</script>

<div class="space-y-6">
	<h1 class="h2 font-bold">Użytkownicy</h1>

	<div class="card preset-filled-surface-50-950 p-5 space-y-4">
		<h2 class="h4 font-semibold">Dodaj użytkownika</h2>
		<form class="flex flex-col sm:flex-row sm:items-end gap-3" onsubmit={createUser}>
			<label class="label flex-1">
				<span class="font-semibold text-sm">Nazwa użytkownika</span>
				<input bind:value={username} type="text" class="input" required />
			</label>
			<label class="label flex-1">
				<span class="font-semibold text-sm">Hasło (min. 8 znaków)</span>
				<input bind:value={password} type="password" class="input" minlength="8" required />
			</label>
			<button type="submit" class="btn preset-filled-primary-500" disabled={saving}>
				{saving ? 'Dodawanie...' : 'Dodaj'}
			</button>
		</form>
	</div>

	<div class="card preset-filled-surface-50-950 p-5">
		<table class="table">
			<thead>
				<tr>
					<th>Nazwa użytkownika</th>
					<th>Rola</th>
					<th>Utworzono</th>
				</tr>
			</thead>
			<tbody>
				{#each users as appUser (appUser.id)}
					<tr>
						<td>{appUser.username}</td>
						<td>
							{#if appUser.is_admin}
								<span class="badge preset-filled-primary-500">Administrator</span>
							{:else}
								<span class="badge preset-tonal-surface">Użytkownik</span>
							{/if}
						</td>
						<td>{appUser.created_at.slice(0, 10)}</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>
