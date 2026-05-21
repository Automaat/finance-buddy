package zus

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Handler is the HTTP boundary for /api/zus.
type Handler struct {
	store  *Store
	logger *slog.Logger
	now    func() time.Time
}

// NewHandler wires the store + logger.
func NewHandler(store *Store, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, logger: logger, now: time.Now}
}

// --- wire types ---

type salaryHistoryEntry struct {
	Year        int     `json:"year"`
	AnnualGross pyFloat `json:"annual_gross"`
}

type inputsResponse struct {
	Owner                     string               `json:"owner"`
	BirthDate                 isoDate              `json:"birth_date"`
	Gender                    string               `json:"gender"`
	RetirementAge             int                  `json:"retirement_age"`
	CurrentGrossMonthlySalary pyFloat              `json:"current_gross_monthly_salary"`
	SalaryGrowthRate          pyFloat              `json:"salary_growth_rate"`
	InflationRate             pyFloat              `json:"inflation_rate"`
	ValorizationRateKonto     pyFloat              `json:"valorization_rate_konto"`
	ValorizationRateSubkonto  pyFloat              `json:"valorization_rate_subkonto"`
	HasOFE                    bool                 `json:"has_ofe"`
	KapitalPoczatkowy         pyFloat              `json:"kapital_poczatkowy"`
	WorkStartYear             int                  `json:"work_start_year"`
	SalaryHistory             []salaryHistoryEntry `json:"salary_history"`
}

type projectionResponse struct {
	Year                 int     `json:"year"`
	Age                  int     `json:"age"`
	AnnualGrossSalary    pyFloat `json:"annual_gross_salary"`
	SalaryCapped         bool    `json:"salary_capped"`
	ContributionKonto    pyFloat `json:"contribution_konto"`
	ContributionSubkonto pyFloat `json:"contribution_subkonto"`
	KontoBalance         pyFloat `json:"konto_balance"`
	SubkontoBalance      pyFloat `json:"subkonto_balance"`
	TotalBalance         pyFloat `json:"total_balance"`
}

type sensitivityResponse struct {
	Label                string  `json:"label"`
	ValorizationKonto    pyFloat `json:"valorization_konto"`
	ValorizationSubkonto pyFloat `json:"valorization_subkonto"`
	MonthlyPensionGross  pyFloat `json:"monthly_pension_gross"`
	MonthlyPensionNet    pyFloat `json:"monthly_pension_net"`
	ReplacementRate      pyFloat `json:"replacement_rate"`
}

type calculateResponse struct {
	Inputs                     inputsResponse        `json:"inputs"`
	YearlyProjections          []projectionResponse  `json:"yearly_projections"`
	LifeExpectancyMonths       pyFloat               `json:"life_expectancy_months"`
	KontoAtRetirement          pyFloat               `json:"konto_at_retirement"`
	SubkontoAtRetirement       pyFloat               `json:"subkonto_at_retirement"`
	KapitalPoczatkowyValorized pyFloat               `json:"kapital_poczatkowy_valorized"`
	TotalCapital               pyFloat               `json:"total_capital"`
	MonthlyPensionGross        pyFloat               `json:"monthly_pension_gross"`
	MonthlyPensionNet          pyFloat               `json:"monthly_pension_net"`
	ReplacementRate            pyFloat               `json:"replacement_rate"`
	LastGrossSalary            pyFloat               `json:"last_gross_salary"`
	Sensitivity                []sensitivityResponse `json:"sensitivity"`
}

type prefillResponse struct {
	BirthDate                 *isoDate             `json:"birth_date"`
	RetirementAge             int                  `json:"retirement_age"`
	Gender                    string               `json:"gender"`
	CurrentGrossMonthlySalary *pyFloat             `json:"current_gross_monthly_salary"`
	Owner                     *string              `json:"owner"`
	SalaryHistory             []salaryHistoryEntry `json:"salary_history"`
	WorkStartYear             *int                 `json:"work_start_year"`
}

// Calculate serves POST /api/zus/calculate.
func (h *Handler) Calculate(w http.ResponseWriter, r *http.Request) {
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	req, vErr := buildInputs(raw, h.now)
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	currentYear := h.now().UTC().Year()
	result := Calculate(req.Inputs, currentYear)

	hist := historyAsList(req.Inputs.SalaryHistory)
	resp := calculateResponse{
		Inputs: inputsResponse{
			Owner:                     req.Inputs.Owner,
			BirthDate:                 isoDate(req.BirthDate),
			Gender:                    req.Inputs.Gender,
			RetirementAge:             req.Inputs.RetirementAge,
			CurrentGrossMonthlySalary: pyFloat(req.Inputs.CurrentGrossMonthlySalary),
			SalaryGrowthRate:          pyFloat(req.Inputs.SalaryGrowthRate),
			InflationRate:             pyFloat(req.Inputs.InflationRate),
			ValorizationRateKonto:     pyFloat(req.Inputs.ValorizationRateKonto),
			ValorizationRateSubkonto:  pyFloat(req.Inputs.ValorizationRateSubkonto),
			HasOFE:                    req.Inputs.HasOFE,
			KapitalPoczatkowy:         pyFloat(req.Inputs.KapitalPoczatkowy),
			WorkStartYear:             req.Inputs.WorkStartYear,
			SalaryHistory:             hist,
		},
		LifeExpectancyMonths:       pyFloat(result.LifeExpectancyMonths),
		KontoAtRetirement:          pyFloat(result.KontoAtRetirement),
		SubkontoAtRetirement:       pyFloat(result.SubkontoAtRetirement),
		KapitalPoczatkowyValorized: pyFloat(result.KapitalPoczatkowyValorized),
		TotalCapital:               pyFloat(result.TotalCapital),
		MonthlyPensionGross:        pyFloat(result.MonthlyPensionGross),
		MonthlyPensionNet:          pyFloat(result.MonthlyPensionNet),
		ReplacementRate:            pyFloat(result.ReplacementRate),
		LastGrossSalary:            pyFloat(result.LastGrossSalary),
	}
	resp.YearlyProjections = make([]projectionResponse, 0, len(result.YearlyProjections))
	for _, p := range result.YearlyProjections {
		resp.YearlyProjections = append(resp.YearlyProjections, projectionResponse{
			Year: p.Year, Age: p.Age,
			AnnualGrossSalary:    pyFloat(p.AnnualGrossSalary),
			SalaryCapped:         p.SalaryCapped,
			ContributionKonto:    pyFloat(p.ContributionKonto),
			ContributionSubkonto: pyFloat(p.ContributionSubkonto),
			KontoBalance:         pyFloat(p.KontoBalance),
			SubkontoBalance:      pyFloat(p.SubkontoBalance),
			TotalBalance:         pyFloat(p.TotalBalance),
		})
	}
	resp.Sensitivity = make([]sensitivityResponse, 0, len(result.Sensitivity))
	for _, s := range result.Sensitivity {
		resp.Sensitivity = append(resp.Sensitivity, sensitivityResponse{
			Label:                s.Label,
			ValorizationKonto:    pyFloat(s.ValorizationKonto),
			ValorizationSubkonto: pyFloat(s.ValorizationSubkonto),
			MonthlyPensionGross:  pyFloat(s.MonthlyPensionGross),
			MonthlyPensionNet:    pyFloat(s.MonthlyPensionNet),
			ReplacementRate:      pyFloat(s.ReplacementRate),
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

// Prefill serves GET /api/zus/prefill.
func (h *Handler) Prefill(w http.ResponseWriter, r *http.Request) {
	// Mirror Python's `if not owner` semantics: empty string from the query
	// param falls through to the first-persona fallback exactly like Python's
	// `Query(None)` default. Don't TrimSpace — Python doesn't.
	var ownerHint *string
	if r.URL.Query().Has("owner") {
		v := r.URL.Query().Get("owner")
		if v != "" {
			ownerHint = &v
		}
	}
	data, err := h.store.LoadPrefill(r.Context(), ownerHint)
	if err != nil {
		h.logger.Error("zus prefill", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	resp := prefillResponse{
		RetirementAge: data.RetirementAge,
		Gender:        "M",
	}
	if data.BirthDate != nil {
		d := isoDate(*data.BirthDate)
		resp.BirthDate = &d
	}
	if data.Owner != nil {
		o := *data.Owner
		resp.Owner = &o
	}
	if data.CurrentGrossMonthlySalary != nil {
		v := pyFloat(*data.CurrentGrossMonthlySalary)
		resp.CurrentGrossMonthlySalary = &v
	}
	if data.WorkStartYear != nil {
		w := *data.WorkStartYear
		resp.WorkStartYear = &w
	}
	resp.SalaryHistory = historyAsList(data.YearlySalaryHistory)
	writeJSON(w, http.StatusOK, resp)
}

func historyAsList(yearly map[int]float64) []salaryHistoryEntry {
	years := make([]int, 0, len(yearly))
	for y := range yearly {
		years = append(years, y)
	}
	sort.Ints(years)
	out := make([]salaryHistoryEntry, 0, len(years))
	for _, y := range years {
		out = append(out, salaryHistoryEntry{Year: y, AnnualGross: pyFloat(yearly[y])})
	}
	return out
}

// --- wire types (shared) ---

type isoDate time.Time

const isoDateLayout = "2006-01-02"

func (d isoDate) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(d).Format(isoDateLayout) + `"`), nil
}

func (d *isoDate) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	t, err := time.Parse(isoDateLayout, s)
	if err != nil {
		return err
	}
	*d = isoDate(t)
	return nil
}

type pyFloat float64

func (f pyFloat) MarshalJSON() ([]byte, error) {
	s := strconv.FormatFloat(float64(f), 'f', -1, 64)
	if !strings.ContainsRune(s, '.') {
		s += ".0"
	}
	return []byte(s), nil
}
