<script lang="ts">
	import { enhance } from '$app/forms';
	import { ShieldCheck } from 'lucide-svelte';
	import type { ActionData } from './$types';

	let { form }: { form: ActionData } = $props();
	let submitting = $state(false);
</script>

<div
	class="flex min-h-screen items-center justify-center bg-surface-50-950 text-surface-950-50 p-4"
>
	<div class="card preset-filled-surface-50-950 w-full max-w-sm shadow-xl p-6 space-y-5">
		<div class="flex items-center gap-2 text-lg font-bold">
			<ShieldCheck class="text-primary-500" size={24} />
			<span>Finansowa Forteca</span>
		</div>

		<form
			method="POST"
			class="space-y-4"
			use:enhance={() => {
				submitting = true;
				return async ({ update }) => {
					await update();
					submitting = false;
				};
			}}
		>
			<label class="label">
				<span class="font-semibold text-sm">Nazwa użytkownika</span>
				<input
					name="username"
					type="text"
					class="input"
					autocomplete="username"
					value={form?.username ?? ''}
					required
				/>
			</label>

			<label class="label">
				<span class="font-semibold text-sm">Hasło</span>
				<input
					name="password"
					type="password"
					class="input"
					autocomplete="current-password"
					required
				/>
			</label>

			<label class="flex items-center gap-2 text-sm">
				<input name="remember_me" type="checkbox" class="checkbox" />
				<span>Zapamiętaj mnie</span>
			</label>

			{#if form?.error}
				<div class="card preset-filled-error-500 p-3 text-sm">{form.error}</div>
			{/if}

			<button type="submit" class="btn preset-filled-primary-500 w-full" disabled={submitting}>
				{submitting ? 'Logowanie...' : 'Zaloguj'}
			</button>
		</form>
	</div>
</div>
