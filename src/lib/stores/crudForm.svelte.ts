// CrudForm is the shared reactive state for a create/edit form on a CRUD page:
// open/close, the item being edited, a saving flag, and an error string. Each
// page (or each independent form on a page — holdings has several) owns its own
// instance, so this is a class factory rather than a singleton store.
//
// `submit` wraps the saving/try-catch/close bookkeeping that every page
// otherwise re-implements. It returns true on success (and closes the form) and
// false on failure, leaving `error` populated so the caller can render it inline
// or raise a toast.
export class CrudForm<T = unknown> {
	open = $state(false);
	editing = $state<T | null>(null);
	saving = $state(false);
	error = $state('');

	get isEditing(): boolean {
		return this.editing !== null;
	}

	openCreate(): void {
		this.editing = null;
		this.error = '';
		this.open = true;
	}

	openEdit(item: T): void {
		this.editing = item;
		this.error = '';
		this.open = true;
	}

	close(): void {
		this.open = false;
		this.editing = null;
		this.error = '';
	}

	async submit(action: () => Promise<void>): Promise<boolean> {
		this.saving = true;
		this.error = '';
		try {
			await action();
			this.close();
			return true;
		} catch (err) {
			this.error = err instanceof Error ? err.message : String(err);
			return false;
		} finally {
			this.saving = false;
		}
	}
}
