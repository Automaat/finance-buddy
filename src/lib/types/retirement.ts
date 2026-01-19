export interface PPKStats {
	owner: string;
	total_value: number;
	employee_contributed: number;
	employer_contributed: number;
	government_contributed: number;
	total_contributed: number;
	returns: number;
	roi_percentage: number;
}

export interface PPKContributionGenerateRequest {
	owner: string;
	month: number;
	year: number;
}

export interface PPKContributionGenerateResponse {
	owner: string;
	month: number;
	year: number;
	gross_salary: number;
	employee_amount: number;
	employer_amount: number;
	total_amount: number;
	transactions_created: number[];
}
