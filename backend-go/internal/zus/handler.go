package zus

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
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
	Year        int          `json:"year"`
	AnnualGross wire.PyFloat `json:"annual_gross"`
}

type inputsResponse struct {
	OwnerUserID               *int                 `json:"owner_user_id"`
	BirthDate                 wire.IsoDate         `json:"birth_date"`
	Gender                    string               `json:"gender"`
	RetirementAge             int                  `json:"retirement_age"`
	CurrentGrossMonthlySalary wire.PyFloat         `json:"current_gross_monthly_salary"`
	SalaryGrowthRate          wire.PyFloat         `json:"salary_growth_rate"`
	InflationRate             wire.PyFloat         `json:"inflation_rate"`
	ValorizationRateKonto     wire.PyFloat         `json:"valorization_rate_konto"`
	ValorizationRateSubkonto  wire.PyFloat         `json:"valorization_rate_subkonto"`
	HasOFE                    bool                 `json:"has_ofe"`
	KapitalPoczatkowy         wire.PyFloat         `json:"kapital_poczatkowy"`
	WorkStartYear             int                  `json:"work_start_year"`
	SalaryHistory             []salaryHistoryEntry `json:"salary_history"`
}

type projectionResponse struct {
	Year                 int          `json:"year"`
	Age                  int          `json:"age"`
	AnnualGrossSalary    wire.PyFloat `json:"annual_gross_salary"`
	SalaryCapped         bool         `json:"salary_capped"`
	ContributionKonto    wire.PyFloat `json:"contribution_konto"`
	ContributionSubkonto wire.PyFloat `json:"contribution_subkonto"`
	KontoBalance         wire.PyFloat `json:"konto_balance"`
	SubkontoBalance      wire.PyFloat `json:"subkonto_balance"`
	TotalBalance         wire.PyFloat `json:"total_balance"`
}

type sensitivityResponse struct {
	Label                string       `json:"label"`
	ValorizationKonto    wire.PyFloat `json:"valorization_konto"`
	ValorizationSubkonto wire.PyFloat `json:"valorization_subkonto"`
	MonthlyPensionGross  wire.PyFloat `json:"monthly_pension_gross"`
	MonthlyPensionNet    wire.PyFloat `json:"monthly_pension_net"`
	ReplacementRate      wire.PyFloat `json:"replacement_rate"`
}

type calculateResponse struct {
	Inputs                     inputsResponse        `json:"inputs"`
	YearlyProjections          []projectionResponse  `json:"yearly_projections"`
	LifeExpectancyMonths       wire.PyFloat          `json:"life_expectancy_months"`
	KontoAtRetirement          wire.PyFloat          `json:"konto_at_retirement"`
	SubkontoAtRetirement       wire.PyFloat          `json:"subkonto_at_retirement"`
	KapitalPoczatkowyValorized wire.PyFloat          `json:"kapital_poczatkowy_valorized"`
	TotalCapital               wire.PyFloat          `json:"total_capital"`
	MonthlyPensionGross        wire.PyFloat          `json:"monthly_pension_gross"`
	MonthlyPensionNet          wire.PyFloat          `json:"monthly_pension_net"`
	ReplacementRate            wire.PyFloat          `json:"replacement_rate"`
	LastGrossSalary            wire.PyFloat          `json:"last_gross_salary"`
	Sensitivity                []sensitivityResponse `json:"sensitivity"`
}

type prefillResponse struct {
	BirthDate                 *wire.IsoDate        `json:"birth_date"`
	RetirementAge             int                  `json:"retirement_age"`
	Gender                    string               `json:"gender"`
	CurrentGrossMonthlySalary *wire.PyFloat        `json:"current_gross_monthly_salary"`
	OwnerUserID               *int                 `json:"owner_user_id"`
	SalaryHistory             []salaryHistoryEntry `json:"salary_history"`
	WorkStartYear             *int                 `json:"work_start_year"`
}

// Calculate serves POST /api/zus/calculate.
func (h *Handler) Calculate(w http.ResponseWriter, r *http.Request) {
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		httputil.WriteBodyValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	req, vErr := buildInputs(raw, h.now)
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	currentYear := h.now().UTC().Year()
	result := Calculate(req.Inputs, currentYear)

	hist := historyAsList(req.Inputs.SalaryHistory)
	resp := calculateResponse{
		Inputs: inputsResponse{
			OwnerUserID:               req.Inputs.OwnerUserID,
			BirthDate:                 wire.IsoDate(req.BirthDate),
			Gender:                    req.Inputs.Gender,
			RetirementAge:             req.Inputs.RetirementAge,
			CurrentGrossMonthlySalary: wire.PyFloat(req.Inputs.CurrentGrossMonthlySalary),
			SalaryGrowthRate:          wire.PyFloat(req.Inputs.SalaryGrowthRate),
			InflationRate:             wire.PyFloat(req.Inputs.InflationRate),
			ValorizationRateKonto:     wire.PyFloat(req.Inputs.ValorizationRateKonto),
			ValorizationRateSubkonto:  wire.PyFloat(req.Inputs.ValorizationRateSubkonto),
			HasOFE:                    req.Inputs.HasOFE,
			KapitalPoczatkowy:         wire.PyFloat(req.Inputs.KapitalPoczatkowy),
			WorkStartYear:             req.Inputs.WorkStartYear,
			SalaryHistory:             hist,
		},
		LifeExpectancyMonths:       wire.PyFloat(result.LifeExpectancyMonths),
		KontoAtRetirement:          wire.PyFloat(result.KontoAtRetirement),
		SubkontoAtRetirement:       wire.PyFloat(result.SubkontoAtRetirement),
		KapitalPoczatkowyValorized: wire.PyFloat(result.KapitalPoczatkowyValorized),
		TotalCapital:               wire.PyFloat(result.TotalCapital),
		MonthlyPensionGross:        wire.PyFloat(result.MonthlyPensionGross),
		MonthlyPensionNet:          wire.PyFloat(result.MonthlyPensionNet),
		ReplacementRate:            wire.PyFloat(result.ReplacementRate),
		LastGrossSalary:            wire.PyFloat(result.LastGrossSalary),
	}
	resp.YearlyProjections = make([]projectionResponse, 0, len(result.YearlyProjections))
	for _, p := range result.YearlyProjections {
		resp.YearlyProjections = append(resp.YearlyProjections, projectionResponse{
			Year: p.Year, Age: p.Age,
			AnnualGrossSalary:    wire.PyFloat(p.AnnualGrossSalary),
			SalaryCapped:         p.SalaryCapped,
			ContributionKonto:    wire.PyFloat(p.ContributionKonto),
			ContributionSubkonto: wire.PyFloat(p.ContributionSubkonto),
			KontoBalance:         wire.PyFloat(p.KontoBalance),
			SubkontoBalance:      wire.PyFloat(p.SubkontoBalance),
			TotalBalance:         wire.PyFloat(p.TotalBalance),
		})
	}
	resp.Sensitivity = make([]sensitivityResponse, 0, len(result.Sensitivity))
	for _, s := range result.Sensitivity {
		resp.Sensitivity = append(resp.Sensitivity, sensitivityResponse{
			Label:                s.Label,
			ValorizationKonto:    wire.PyFloat(s.ValorizationKonto),
			ValorizationSubkonto: wire.PyFloat(s.ValorizationSubkonto),
			MonthlyPensionGross:  wire.PyFloat(s.MonthlyPensionGross),
			MonthlyPensionNet:    wire.PyFloat(s.MonthlyPensionNet),
			ReplacementRate:      wire.PyFloat(s.ReplacementRate),
		})
	}
	httputil.WriteJSON(w, http.StatusOK, resp)
}

// Prefill serves GET /api/zus/prefill.
func (h *Handler) Prefill(w http.ResponseWriter, r *http.Request) {
	// An empty owner_user_id query value falls through to the first-user
	// fallback, mirroring Python's `Query(None)` default.
	var ownerHint *int
	if v := r.URL.Query().Get("owner_user_id"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			httputil.WriteBodyValidationError(w, "owner_user_id", "must be an integer", v)
			return
		}
		ownerHint = &n
	}
	data, err := h.store.LoadPrefill(r.Context(), ownerHint)
	if err != nil {
		h.logger.Error("zus prefill", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	resp := prefillResponse{
		RetirementAge: data.RetirementAge,
		Gender:        "M",
	}
	if data.BirthDate != nil {
		d := wire.IsoDate(*data.BirthDate)
		resp.BirthDate = &d
	}
	resp.OwnerUserID = data.OwnerUserID
	if data.CurrentGrossMonthlySalary != nil {
		v := wire.PyFloat(*data.CurrentGrossMonthlySalary)
		resp.CurrentGrossMonthlySalary = &v
	}
	if data.WorkStartYear != nil {
		w := *data.WorkStartYear
		resp.WorkStartYear = &w
	}
	resp.SalaryHistory = historyAsList(data.YearlySalaryHistory)
	httputil.WriteJSON(w, http.StatusOK, resp)
}

func historyAsList(yearly map[int]float64) []salaryHistoryEntry {
	years := make([]int, 0, len(yearly))
	for y := range yearly {
		years = append(years, y)
	}
	sort.Ints(years)
	out := make([]salaryHistoryEntry, 0, len(years))
	for _, y := range years {
		out = append(out, salaryHistoryEntry{Year: y, AnnualGross: wire.PyFloat(yearly[y])})
	}
	return out
}
