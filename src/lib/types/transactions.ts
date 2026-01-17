export interface Transaction {
	id: number;
	account_id: number;
	account_name: string;
	amount: number;
	date: string;
	owner: string;
	created_at: string;
}

export interface TransactionsData {
	transactions: Transaction[];
	total_invested: number;
	transaction_count: number;
}
