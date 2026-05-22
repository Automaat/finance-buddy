// OwnerOption is a household member offered as an owner picker choice,
// served by GET /api/users. owner_user_id fields reference its `id`;
// a null owner_user_id means jointly owned ("Wspólne").
export interface OwnerOption {
	id: number;
	name: string;
}

// ownerName resolves an owner_user_id to a display name.
export function ownerName(owners: OwnerOption[], id: number | null): string {
	if (id === null) {
		return 'Wspólne';
	}
	return owners.find((o) => o.id === id)?.name ?? '—';
}
