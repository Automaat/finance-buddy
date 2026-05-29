// Wire shapes for GET /api/exposure/currency. The response is per-currency
// aggregates across the investment portfolio (PLN vs USD/EUR/GBP/CHF…) with an
// optional drift band when a target PLN share is supplied.

export interface CurrencyBucket {
	currency: string;
	value_pln: number;
	percent: number;
}

export interface DriftBand {
	target_pln_pct: number;
	actual_pln_pct: number;
	drift_pln_pct: number;
	within_tolerance: boolean;
	tolerance_pct: number;
}

export interface CurrencyExposureReport {
	currencies: CurrencyBucket[];
	total_pln: number;
	pln_pct: number;
	foreign_pct: number;
	snapshot_date: string;
	drift?: DriftBand;
}
