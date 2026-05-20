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

export interface InflationContext {
	owner: string;
	last_change_date: string;
	previous_change_date: string | null;
	previous_salary: number | null;
	previous_salary_in_today_pln: number | null;
	current_salary: number;
	real_change_pln: number | null;
	real_change_pct: number | null;
	cpi_as_of_year: number;
}

export interface SalariesData {
	salary_records: SalaryRecord[];
	total_count: number;
	current_salaries: Record<string, number | null>;
	inflation_context: Record<string, InflationContext>;
	available_companies: string[];
}
