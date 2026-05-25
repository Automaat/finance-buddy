export type AccountType = 'asset' | 'liability';

export type AccountCategory =
	| 'bank'
	| 'saving_account'
	| 'stock'
	| 'bond'
	| 'gold'
	| 'real_estate'
	| 'ppk'
	| 'fund'
	| 'etf'
	| 'vehicle'
	| 'mortgage'
	| 'installment'
	| 'other';

export type AccountWrapper = 'IKE' | 'IKZE' | 'PPK';

export type AccountPurpose = 'retirement' | 'emergency_fund' | 'general';

export interface Account {
	id: number;
	name: string;
	type: AccountType;
	category: AccountCategory;
	owner_user_id: number | null;
	currency: string;
	account_wrapper: AccountWrapper | null;
	purpose: AccountPurpose;
	square_meters: number | null;
	is_active: boolean;
	receives_contributions: boolean;
	excluded_from_fire: boolean;
	created_at: string;
	current_value: number;
}

export interface Asset {
	id: number;
	name: string;
	is_active: boolean;
	created_at: string;
	current_value: number;
}

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
