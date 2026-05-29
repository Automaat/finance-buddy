// Package rules is the typed source-of-truth for Polish financial constants
// (PIT thresholds, PPK / IKE / IKZE limits, minimum wage, withholding tax)
// surfaced to the UI alongside their citation metadata.
//
// Each rule carries the source URL, effective date, and last-checked date so
// the simulations / settings views can show "where does this number come
// from" — see issue #553. Issue #545 layers on top, replacing the duplicate
// numeric literals scattered through the codebase with imports from here.
package rules

import (
	"log/slog"
	"time"

	"github.com/shopspring/decimal"
)

// Rule describes one Polish financial constant with citation metadata.
//
// All exported fields are JSON-marshalable; see handler.go for the wire
// shape. `Value` is a decimal.Decimal so percentage rates and money limits
// keep exact precision (no float drift between input data and the form
// that displays them).
type Rule struct {
	// Key is the stable code-side identifier (e.g. "ike_limit_2026") used
	// by other Go packages that want to reference this rule. Doesn't change
	// across years — see Year + EffectiveDate for the time anchor.
	Key string

	// Name is the short Polish-language label shown in the UI.
	Name string

	// Category groups related rules so the UI can render them in sections
	// (e.g. "ike_limit", "pit", "ppk", "minimum_wage").
	Category string

	// Value is the canonical number. Unit explains what the number means
	// (PLN, PLN/month, %, etc.) so the UI doesn't have to guess.
	Value decimal.Decimal
	Unit  string

	// Year + EffectiveDate locate the rule in time. Year is the calendar
	// year the rule applies to; EffectiveDate is when it took effect (may
	// be mid-year for tax changes).
	Year          int
	EffectiveDate time.Time

	// SourceURL points at the official source (sejm.gov.pl, mf.gov.pl,
	// gov.pl, ZUS, etc.). LastCheckedDate is the date a maintainer last
	// verified the value still matches the source.
	SourceURL       string
	LastCheckedDate time.Time

	// Description is a longer Polish-language note explaining what the
	// rule covers (e.g. "annual contribution cap for IKE accounts").
	Description string
}

// lastChecked is the most-recent date a maintainer eyeballed the source
// pages and confirmed each value below. Bumped in lockstep with PR review
// when refreshing the table.
var lastChecked = time.Date(2026, time.May, 24, 0, 0, 0, 0, time.UTC)

// jan2026 is the EffectiveDate every 2026-anchored rule below uses (Polish
// tax/contribution rules typically take effect on Jan 1). Pulled out so the
// table reads as data rather than repeated time.Date calls.
var jan2026 = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

// Polish2026 is the curated, year-anchored snapshot of Polish financial
// constants used by the simulations and dashboard. The list is intentionally
// short — it's the constants the app actually reads, not an exhaustive
// gov.pl mirror. Add a row when another module starts depending on a
// hand-coded literal; remove it only after every caller has migrated.
var Polish2026 = []Rule{
	{
		Key:             "ike_limit_2026",
		Name:            "Roczny limit wpłat IKE",
		Category:        "ike_limit",
		Value:           decimal.NewFromInt(28260),
		Unit:            "PLN",
		Year:            2026,
		EffectiveDate:   jan2026,
		SourceURL:       "https://www.gov.pl/web/finanse/limity-wplat-na-ike-i-ikze",
		LastCheckedDate: lastChecked,
		Description:     "Maksymalna roczna wpłata na konto IKE w 2026 r.",
	},
	{
		Key:             "ikze_limit_2026",
		Name:            "Roczny limit wpłat IKZE",
		Category:        "ikze_limit",
		Value:           decimal.NewFromInt(11304),
		Unit:            "PLN",
		Year:            2026,
		EffectiveDate:   jan2026,
		SourceURL:       "https://www.gov.pl/web/finanse/limity-wplat-na-ike-i-ikze",
		LastCheckedDate: lastChecked,
		Description:     "Maksymalna roczna wpłata na konto IKZE w 2026 r.",
	},
	{
		Key:             "ikze_limit_b2b_2026",
		Name:            "Roczny limit IKZE (samozatrudnieni)",
		Category:        "ikze_limit",
		Value:           decimal.NewFromInt(16956),
		Unit:            "PLN",
		Year:            2026,
		EffectiveDate:   jan2026,
		SourceURL:       "https://www.gov.pl/web/finanse/limity-wplat-na-ike-i-ikze",
		LastCheckedDate: lastChecked,
		Description:     "Podwyższony limit dla osób prowadzących pozarolniczą działalność gospodarczą.",
	},
	{
		Key:             "ppk_below_threshold_2026",
		Name:            "Próg wynagrodzenia PPK (1,2× minimalnej krajowej)",
		Category:        "ppk",
		Value:           decimal.NewFromInt(5767),
		Unit:            "PLN/miesiąc",
		Year:            2026,
		EffectiveDate:   jan2026,
		SourceURL:       "https://www.mojeppk.pl/",
		LastCheckedDate: lastChecked,
		Description:     "Pracownik zarabiający poniżej tego progu może obniżyć składkę pracownika z 2% do 0,5%.",
	},
	{
		Key:             "minimum_wage_2026",
		Name:            "Płaca minimalna brutto",
		Category:        "minimum_wage",
		Value:           decimal.NewFromInt(4806),
		Unit:            "PLN/miesiąc",
		Year:            2026,
		EffectiveDate:   jan2026,
		SourceURL:       "https://www.gov.pl/web/rodzina/place-minimalna",
		LastCheckedDate: lastChecked,
		Description:     "Minimalne wynagrodzenie obowiązujące w 2026 r.",
	},
	{
		Key:             "pit_threshold_first_2026",
		Name:            "Próg pierwszego progu PIT",
		Category:        "pit",
		Value:           decimal.NewFromInt(120000),
		Unit:            "PLN",
		Year:            2026,
		EffectiveDate:   jan2026,
		SourceURL:       "https://www.podatki.gov.pl/pit/abc-pit/skala-podatkowa/",
		LastCheckedDate: lastChecked,
		Description:     "Dochód, po przekroczeniu którego stosuje się drugi próg skali PIT.",
	},
	{
		Key:             "pit_rate_first_2026",
		Name:            "Stawka PIT — pierwszy próg",
		Category:        "pit",
		Value:           decimal.RequireFromString("0.12"),
		Unit:            "udział",
		Year:            2026,
		EffectiveDate:   jan2026,
		SourceURL:       "https://www.podatki.gov.pl/pit/abc-pit/skala-podatkowa/",
		LastCheckedDate: lastChecked,
		Description:     "Skala podatkowa: 12% do progu, pomniejszone o kwotę wolną.",
	},
	{
		Key:             "pit_rate_second_2026",
		Name:            "Stawka PIT — drugi próg",
		Category:        "pit",
		Value:           decimal.RequireFromString("0.32"),
		Unit:            "udział",
		Year:            2026,
		EffectiveDate:   jan2026,
		SourceURL:       "https://www.podatki.gov.pl/pit/abc-pit/skala-podatkowa/",
		LastCheckedDate: lastChecked,
		Description:     "Stawka stosowana do dochodu powyżej pierwszego progu.",
	},
	{
		Key:             "capital_gains_tax_2026",
		Name:            "Podatek Belki (od zysków kapitałowych)",
		Category:        "capital_gains",
		Value:           decimal.RequireFromString("0.19"),
		Unit:            "udział",
		Year:            2026,
		EffectiveDate:   jan2026,
		SourceURL:       "https://www.podatki.gov.pl/pit/abc-pit/podatki-od-zyskow-kapitalowych/",
		LastCheckedDate: lastChecked,
		Description:     "Zryczałtowany podatek od zysków z inwestycji (oszczędności, akcje, fundusze).",
	},
	{
		Key:             "pit_free_amount_2026",
		Name:            "Kwota wolna od podatku PIT",
		Category:        "pit",
		Value:           decimal.NewFromInt(30000),
		Unit:            "PLN",
		Year:            2026,
		EffectiveDate:   jan2026,
		SourceURL:       "https://www.podatki.gov.pl/pit/abc-pit/skala-podatkowa/",
		LastCheckedDate: lastChecked,
		Description:     "Roczny dochód wolny od podatku w skali PIT.",
	},
	{
		Key:             "pit_solidarity_threshold_2026",
		Name:            "Próg daniny solidarnościowej",
		Category:        "pit",
		Value:           decimal.NewFromInt(1000000),
		Unit:            "PLN",
		Year:            2026,
		EffectiveDate:   jan2026,
		SourceURL:       "https://www.podatki.gov.pl/pit/abc-pit/danina-solidarnosciowa/",
		LastCheckedDate: lastChecked,
		Description:     "Roczny dochód, powyżej którego doliczana jest danina solidarnościowa.",
	},
	{
		Key:             "pit_solidarity_rate_2026",
		Name:            "Stawka daniny solidarnościowej",
		Category:        "pit",
		Value:           decimal.RequireFromString("0.04"),
		Unit:            "udział",
		Year:            2026,
		EffectiveDate:   jan2026,
		SourceURL:       "https://www.podatki.gov.pl/pit/abc-pit/danina-solidarnosciowa/",
		LastCheckedDate: lastChecked,
		Description:     "Dodatkowe 4% od nadwyżki dochodu ponad 1 mln PLN.",
	},
	{
		Key:             "zus_cap_30x_2026",
		Name:            "Limit 30-krotności (podstawa ZUS)",
		Category:        "zus",
		Value:           decimal.NewFromInt(282600),
		Unit:            "PLN",
		Year:            2026,
		EffectiveDate:   jan2026,
		SourceURL:       "https://www.zus.pl/baza-wiedzy/skladki-wskazniki-odsetki/skladki/wysokosc-skladek-na-ubezpieczenia-spoleczne",
		LastCheckedDate: lastChecked,
		Description:     "Roczna podstawa wymiaru składek emerytalnych i rentowych — przyjmowane 30× prognozowane przeciętne wynagrodzenie.",
	},
	{
		Key:             "b2b_liniowy_rate_2026",
		Name:            "PIT liniowy B2B",
		Category:        "pit",
		Value:           decimal.RequireFromString("0.19"),
		Unit:            "udział",
		Year:            2026,
		EffectiveDate:   jan2026,
		SourceURL:       "https://www.podatki.gov.pl/pit/abc-pit/zasady-opodatkowania/",
		LastCheckedDate: lastChecked,
		Description:     "Liniowa stawka 19% dla osób prowadzących pozarolniczą działalność gospodarczą.",
	},
	{
		Key:             "ryczalt_it_rate_2026",
		Name:            "Ryczałt IT (12%)",
		Category:        "pit",
		Value:           decimal.RequireFromString("0.12"),
		Unit:            "udział",
		Year:            2026,
		EffectiveDate:   jan2026,
		SourceURL:       "https://www.podatki.gov.pl/pit/abc-pit/zryczaltowane-formy-opodatkowania/ryczalt-od-przychodow-ewidencjonowanych/",
		LastCheckedDate: lastChecked,
		Description:     "Stawka ryczałtu od przychodów ewidencjonowanych dla branży IT (PKWiU 62.0x).",
	},
}

// Get returns the rule for the given Key, or false if no rule matches.
// Callers in other packages use this to swap hand-coded literals for
// metadata-backed constants without round-tripping through the API.
func Get(key string) (Rule, bool) {
	for i := range Polish2026 {
		if Polish2026[i].Key == key {
			return Polish2026[i], true
		}
	}
	return Rule{}, false
}

// All returns a defensive copy of the Polish2026 list so callers can sort
// or filter it without mutating the package-level table.
func All() []Rule {
	out := make([]Rule, len(Polish2026))
	copy(out, Polish2026)
	return out
}

// Float64Or returns the rule's Value as a float64, or the supplied fallback
// (logged at WARN) when the key is unknown. Use to pin a Go constant to the
// centralized table without crashing the process if the table ever drifts —
// callers pass the statutory default as the fallback, so a missing rule
// degrades to a documented value instead of a panic.
func Float64Or(key string, fallback float64) float64 {
	r, ok := Get(key)
	if !ok {
		slog.Default().Warn("rules: missing rule, using fallback",
			"key", key, "fallback", fallback)
		return fallback
	}
	v, _ := r.Value.Float64()
	return v
}

// ByCategory returns every rule whose Category matches. Preserves the
// declaration order from Polish2026.
func ByCategory(category string) []Rule {
	out := make([]Rule, 0)
	for i := range Polish2026 {
		if Polish2026[i].Category == category {
			out = append(out, Polish2026[i])
		}
	}
	return out
}
