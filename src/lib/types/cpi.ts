export interface CpiPoint {
	year: number;
	yoy_rate: number;
	cumulative_index: number;
}

export interface CpiSeries {
	points: CpiPoint[];
	base_year: number | null;
	latest_year: number | null;
	source: string;
}
