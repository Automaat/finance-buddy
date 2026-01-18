export interface SalaryRecord {
	id: number;
	date: string;
	gross_amount: number;
	contract_type: string;
	company: string;
	owner: string;
	is_active: boolean;
	created_at: string;
}

export interface SalariesData {
	salary_records: SalaryRecord[];
	total_count: number;
	current_salary_marcin: number | null;
	current_salary_ewa: number | null;
}
