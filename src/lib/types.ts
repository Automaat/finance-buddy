export interface SnapshotValueResponse {
	id: number;
	asset_id: number | null;
	asset_name: string | null;
	account_id: number | null;
	account_name: string | null;
	value: number;
}

export interface SnapshotResponse {
	id: number;
	date: string;
	notes: string | null;
	values: SnapshotValueResponse[];
}
