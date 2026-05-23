<script lang="ts">
	import Modal from './Modal.svelte';
	import { confirmDialog } from '$lib/stores/confirm.svelte';

	const req = $derived(confirmDialog.current);
</script>

{#if req}
	<Modal
		open
		title={req.title}
		confirmText={req.confirmText ?? 'OK'}
		cancelText={req.cancelText ?? 'Anuluj'}
		confirmVariant={req.danger ? 'danger' : 'primary'}
		confirmDisabled={req.pending}
		onConfirm={confirmDialog.confirm}
		onCancel={confirmDialog.cancel}
	>
		<p class="whitespace-pre-line">{req.message}</p>
	</Modal>
{/if}
