<script lang="ts">
	import { env } from '$env/dynamic/public';
	import { invalidateAll } from '$app/navigation';
	import { toast } from '$lib/stores/toast.svelte';
	import Modal from '$lib/components/Modal.svelte';
	import type { PageData } from './$types';

	interface AppUser {
		id: number;
		username: string;
		is_admin: boolean;
		name: string | null;
		surname: string | null;
		ppk_employee_rate: string | null;
		ppk_employer_rate: string | null;
		created_at: string;
	}

	let { data }: { data: PageData } = $props();
	const users = $derived(data.users as AppUser[]);
	const apiUrl = env.PUBLIC_API_URL_BROWSER ?? '';

	let username = $state('');
	let password = $state('');
	let name = $state('');
	let surname = $state('');
	let ppkEmployee = $state(2);
	let ppkEmployer = $state(1.5);
	let creating = $state(false);

	let editUser = $state<AppUser | null>(null);
	let editName = $state('');
	let editSurname = $state('');
	let editPpkEmployee = $state(2);
	let editPpkEmployer = $state(1.5);
	let editSaving = $state(false);

	async function request(method: string, path: string, body: unknown): Promise<void> {
		const response = await fetch(`${apiUrl}${path}`, {
			method,
			headers: { 'content-type': 'application/json' },
			body: JSON.stringify(body)
		});
		if (!response.ok) {
			const err = await response.json().catch(() => null);
			const detail =
				err && typeof err === 'object' && 'detail' in err && err.detail
					? String(err.detail)
					: 'Operacja nie powiodła się';
			throw new Error(detail);
		}
	}

	async function createUser(event: Event): Promise<void> {
		event.preventDefault();
		creating = true;
		try {
			await request('POST', '/api/auth/users', {
				username,
				password,
				name: name.trim() || null,
				surname: surname.trim() || null,
				ppk_employee_rate: ppkEmployee,
				ppk_employer_rate: ppkEmployer
			});
			toast.success(`Użytkownik „${username}” został utworzony`);
			username = '';
			password = '';
			name = '';
			surname = '';
			ppkEmployee = 2;
			ppkEmployer = 1.5;
			await invalidateAll();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Wystąpił błąd');
		} finally {
			creating = false;
		}
	}

	function openEdit(appUser: AppUser): void {
		editUser = appUser;
		editName = appUser.name ?? '';
		editSurname = appUser.surname ?? '';
		editPpkEmployee = appUser.ppk_employee_rate ? Number(appUser.ppk_employee_rate) : 2;
		editPpkEmployer = appUser.ppk_employer_rate ? Number(appUser.ppk_employer_rate) : 1.5;
	}

	async function saveEdit(): Promise<void> {
		if (!editUser) {
			return;
		}
		editSaving = true;
		try {
			await request('PATCH', `/api/auth/users/${editUser.id}`, {
				name: editName.trim() || null,
				surname: editSurname.trim() || null,
				ppk_employee_rate: editPpkEmployee,
				ppk_employer_rate: editPpkEmployer
			});
			toast.success('Zapisano zmiany');
			editUser = null;
			await invalidateAll();
		} catch (err) {
			toast.error(err instanceof Error ? err.message : 'Wystąpił błąd');
		} finally {
			editSaving = false;
		}
	}

	function fullName(appUser: AppUser): string {
		return [appUser.name, appUser.surname].filter(Boolean).join(' ') || '—';
	}
</script>

<div class="space-y-6">
	<h1 class="h2 font-bold">Użytkownicy</h1>

	<div class="card preset-filled-surface-50-950 p-5 space-y-4">
		<h2 class="h4 font-semibold">Dodaj użytkownika</h2>
		<form class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3" onsubmit={createUser}>
			<label class="label">
				<span class="font-semibold text-sm">Nazwa użytkownika</span>
				<input bind:value={username} type="text" class="input" required />
			</label>
			<label class="label">
				<span class="font-semibold text-sm">Hasło (min. 8 znaków)</span>
				<input bind:value={password} type="password" class="input" minlength="8" required />
			</label>
			<label class="label">
				<span class="font-semibold text-sm">Imię</span>
				<input bind:value={name} type="text" class="input" />
			</label>
			<label class="label">
				<span class="font-semibold text-sm">Nazwisko</span>
				<input bind:value={surname} type="text" class="input" />
			</label>
			<label class="label">
				<span class="font-semibold text-sm">PPK pracownik (%)</span>
				<input bind:value={ppkEmployee} type="number" step="0.01" min="0.5" max="4" class="input" />
			</label>
			<label class="label">
				<span class="font-semibold text-sm">PPK pracodawca (%)</span>
				<input bind:value={ppkEmployer} type="number" step="0.01" min="0.5" max="4" class="input" />
			</label>
			<div class="sm:col-span-2 lg:col-span-3">
				<button type="submit" class="btn preset-filled-primary-500" disabled={creating}>
					{creating ? 'Dodawanie...' : 'Dodaj'}
				</button>
			</div>
		</form>
	</div>

	<div class="card preset-filled-surface-50-950 p-5">
		<table class="table">
			<thead>
				<tr>
					<th>Nazwa użytkownika</th>
					<th>Imię i nazwisko</th>
					<th>PPK (prac. / prac.)</th>
					<th>Rola</th>
					<th>Utworzono</th>
					<th></th>
				</tr>
			</thead>
			<tbody>
				{#each users as appUser (appUser.id)}
					<tr>
						<td>{appUser.username}</td>
						<td>{fullName(appUser)}</td>
						<td>{appUser.ppk_employee_rate ?? '—'} / {appUser.ppk_employer_rate ?? '—'}</td>
						<td>
							{#if appUser.is_admin}
								<span class="badge preset-filled-primary-500">Administrator</span>
							{:else}
								<span class="badge preset-tonal-surface">Użytkownik</span>
							{/if}
						</td>
						<td>{appUser.created_at.slice(0, 10)}</td>
						<td>
							<button
								type="button"
								class="btn btn-sm preset-tonal-surface"
								onclick={() => openEdit(appUser)}
							>
								Edytuj
							</button>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
</div>

<Modal
	open={editUser !== null}
	title="Edytuj użytkownika"
	confirmDisabled={editSaving}
	onConfirm={saveEdit}
	onCancel={() => (editUser = null)}
>
	<div class="space-y-3">
		<label class="label">
			<span class="font-semibold text-sm">Imię</span>
			<input bind:value={editName} type="text" class="input" />
		</label>
		<label class="label">
			<span class="font-semibold text-sm">Nazwisko</span>
			<input bind:value={editSurname} type="text" class="input" />
		</label>
		<label class="label">
			<span class="font-semibold text-sm">PPK pracownik (%)</span>
			<input
				bind:value={editPpkEmployee}
				type="number"
				step="0.01"
				min="0.5"
				max="4"
				class="input"
			/>
		</label>
		<label class="label">
			<span class="font-semibold text-sm">PPK pracodawca (%)</span>
			<input
				bind:value={editPpkEmployer}
				type="number"
				step="0.01"
				min="0.5"
				max="4"
				class="input"
			/>
		</label>
	</div>
</Modal>
