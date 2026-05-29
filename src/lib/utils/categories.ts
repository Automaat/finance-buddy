// Canonical Polish labels for account/asset categories. Single source of
// truth — goals, holdings, snapshot forms etc. import this instead of each
// re-declaring its own map (which had drifted, e.g. "Oszczędności" vs
// "Konto oszczędnościowe").
export const CATEGORY_LABELS: Record<string, string> = {
	bank: 'Konto bankowe',
	saving_account: 'Konto oszczędnościowe',
	stock: 'Akcje',
	bond: 'Obligacje',
	gold: 'Złoto',
	real_estate: 'Nieruchomość',
	ppk: 'PPK',
	fund: 'Fundusz',
	etf: 'ETF',
	vehicle: 'Pojazd',
	mortgage: 'Hipoteka',
	installment: 'Raty',
	other: 'Inne'
};

// categoryLabel returns the Polish label for a category key, falling back to a
// capitalized form of the raw key for anything not in the map.
export function categoryLabel(category: string): string {
	return CATEGORY_LABELS[category] ?? category.charAt(0).toUpperCase() + category.slice(1);
}
