export interface SalaryRecord {
	id: number;
	date: string;
	gross_amount: number;
	contract_type: string;
	company: string;
	owner_user_id: number | null;
	is_active: boolean;
	created_at: string;
}

export interface InflationContext {
	owner_user_id: number;
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

export type BonusType = 'annual' | 'signon' | 'spot' | 'retention';

export interface BonusEvent {
	id: number;
	date: string;
	amount: number;
	currency: string;
	type: BonusType;
	company: string;
	owner_user_id: number | null;
	contract_type: string;
	notes: string | null;
	is_active: boolean;
	created_at: string;
	amount_pln: number | null;
	fx_rate: number | null;
}

export interface BonusEventsData {
	bonus_events: BonusEvent[];
	total_count: number;
	available_companies: string[];
}

export type EquityGrantType = 'option' | 'rsu';
export type VestingFrequency = 'monthly' | 'quarterly' | 'yearly';
export type EquityTaxTreatment = 'capital_gains_19' | 'employment_income';

export interface CustomVestingEvent {
	month: number;
	pct: number;
}

export interface EquityGrant {
	id: number;
	grant_date: string;
	type: EquityGrantType;
	company: string;
	owner_user_id: number | null;
	total_shares: number;
	strike_price: number | null;
	currency: string;
	vest_start_date: string;
	vest_cliff_months: number;
	vest_total_months: number;
	vest_frequency: VestingFrequency;
	vest_custom_schedule: CustomVestingEvent[] | null;
	requires_liquidity_event: boolean;
	liquidity_event_date: string | null;
	tax_treatment: EquityTaxTreatment;
	notes: string | null;
	is_active: boolean;
	created_at: string;
	vested_shares_today: number;
	vesting_progress_pct: number;
	paper_value_base: number | null;
	paper_value_low: number | null;
	paper_value_high: number | null;
	paper_value_currency: string | null;
	paper_value_base_pln: number | null;
	paper_value_low_pln: number | null;
	paper_value_high_pln: number | null;
	fx_rate: number | null;
	valuation_date: string | null;
	valuation_source: string | null;
}

export interface EquityGrantsData {
	equity_grants: EquityGrant[];
	total_count: number;
	available_companies: string[];
}

export type ValuationSource = '409a' | 'preferred_round' | 'tender' | 'estimate';

export interface CompanyValuation {
	id: number;
	company: string;
	date: string;
	currency: string;
	fmv_per_share: number;
	fmv_low: number | null;
	fmv_high: number | null;
	source: ValuationSource;
	common_stock_discount_pct: number | null;
	notes: string | null;
	is_active: boolean;
	created_at: string;
}

export interface CompanyValuationsData {
	company_valuations: CompanyValuation[];
	total_count: number;
	available_companies: string[];
}
