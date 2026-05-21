package retirement

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

type yearlyStat struct {
	Year                int      `json:"year"`
	AccountWrapper      string   `json:"account_wrapper"`
	Owner               string   `json:"owner"`
	LimitAmount         *pyFloat `json:"limit_amount"`
	TotalContributed    pyFloat  `json:"total_contributed"`
	EmployeeContributed pyFloat  `json:"employee_contributed"`
	EmployerContributed pyFloat  `json:"employer_contributed"`
	Remaining           *pyFloat `json:"remaining"`
	PercentageUsed      *pyFloat `json:"percentage_used"`
	IsWarning           bool     `json:"is_warning"`
}

type ppkStat struct {
	Owner                 string  `json:"owner"`
	TotalValue            pyFloat `json:"total_value"`
	EmployeeContributed   pyFloat `json:"employee_contributed"`
	EmployerContributed   pyFloat `json:"employer_contributed"`
	GovernmentContributed pyFloat `json:"government_contributed"`
	TotalContributed      pyFloat `json:"total_contributed"`
	Returns               pyFloat `json:"returns"`
	ROIPercentage         pyFloat `json:"roi_percentage"`
}

type limitResponse struct {
	ID             int     `json:"id"`
	Year           int     `json:"year"`
	AccountWrapper string  `json:"account_wrapper"`
	Owner          string  `json:"owner"`
	LimitAmount    pyFloat `json:"limit_amount"`
	Notes          *string `json:"notes"`
}

type ppkGenerateResponse struct {
	Owner               string  `json:"owner"`
	Month               int     `json:"month"`
	Year                int     `json:"year"`
	GrossSalary         pyFloat `json:"gross_salary"`
	EmployeeAmount      pyFloat `json:"employee_amount"`
	EmployerAmount      pyFloat `json:"employer_amount"`
	TotalAmount         pyFloat `json:"total_amount"`
	TransactionsCreated []int   `json:"transactions_created"`
}

// Handler is the HTTP boundary for /api/retirement.
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

// Stats serves GET /api/retirement/stats.
func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	year := h.now().UTC().Year()
	if v := r.URL.Query().Get("year"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			writeValidationError(w, "year", "must be an integer", v)
			return
		}
		year = n
	}
	owners, err := h.resolveOwners(r.Context(), r.URL.Query().Get("owner"))
	if err != nil {
		h.logger.Error("resolve owners", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := []yearlyStat{}
	for _, wrapper := range []string{"IKE", "IKZE"} {
		for _, owner := range owners {
			stat, included, err := h.buildYearlyStat(r.Context(), year, wrapper, owner)
			if err != nil {
				h.logger.Error("yearly stat", "err", err)
				writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
				return
			}
			if included {
				out = append(out, stat)
			}
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) buildYearlyStat(ctx context.Context, year int, wrapper, owner string) (yearlyStat, bool, error) {
	accountIDs, err := h.store.AccountIDsForWrapper(ctx, wrapper, owner)
	if err != nil {
		return yearlyStat{}, false, err
	}
	if len(accountIDs) == 0 {
		return yearlyStat{}, false, nil
	}
	totals, err := h.store.YearlyContributions(ctx, year, accountIDs)
	if err != nil {
		return yearlyStat{}, false, err
	}
	limit, configured, err := h.store.LimitConfigured(ctx, year, wrapper, owner)
	if err != nil {
		return yearlyStat{}, false, err
	}
	if !configured {
		return yearlyStat{}, false, nil
	}
	totalF, _ := totals.Total.Float64()
	employeeF, _ := totals.Employee.Float64()
	employerF, _ := totals.Employer.Float64()
	if totalF > employeeF+employerF {
		employeeF += totalF - (employeeF + employerF)
	}
	limitF, _ := limit.LimitAmount.Float64()
	remaining := limitF - totalF
	if remaining < 0 {
		remaining = 0
	}
	percentage := 0.0
	if limitF > 0 {
		percentage = totalF / limitF * 100
	}
	percentageRounded := math.Round(percentage*10) / 10
	lAmt := pyFloat(limitF)
	rem := pyFloat(remaining)
	pct := pyFloat(percentageRounded)
	return yearlyStat{
		Year: year, AccountWrapper: wrapper, Owner: owner,
		LimitAmount:         &lAmt,
		TotalContributed:    pyFloat(totalF),
		EmployeeContributed: pyFloat(employeeF),
		EmployerContributed: pyFloat(employerF),
		Remaining:           &rem,
		PercentageUsed:      &pct,
		IsWarning:           percentageRounded >= 90,
	}, true, nil
}

// PPKStats serves GET /api/retirement/ppk-stats.
func (h *Handler) PPKStats(w http.ResponseWriter, r *http.Request) {
	owners, err := h.resolveOwners(r.Context(), r.URL.Query().Get("owner"))
	if err != nil {
		h.logger.Error("resolve owners", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := []ppkStat{}
	for _, owner := range owners {
		stat, included, err := h.buildPPKStat(r.Context(), owner)
		if err != nil {
			h.logger.Error("ppk stat", "err", err)
			writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		if included {
			out = append(out, stat)
		}
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) buildPPKStat(ctx context.Context, owner string) (ppkStat, bool, error) {
	accountIDs, err := h.store.AccountIDsForWrapper(ctx, "PPK", owner)
	if err != nil {
		return ppkStat{}, false, err
	}
	if len(accountIDs) == 0 {
		return ppkStat{}, false, nil
	}
	totals, err := h.store.PPKContributions(ctx, accountIDs)
	if err != nil {
		return ppkStat{}, false, err
	}
	totalF, _ := totals.Total.Float64()
	employeeF, _ := totals.Employee.Float64()
	employerF, _ := totals.Employer.Float64()
	governmentF, _ := totals.Government.Float64()
	if totalF > employeeF+employerF+governmentF {
		employeeF += totalF - (employeeF + employerF + governmentF)
	}
	latest, err := h.store.LatestSnapshotValueSum(ctx, accountIDs)
	if err != nil {
		return ppkStat{}, false, err
	}
	totalValue, _ := latest.Float64()
	returns := totalValue - totalF
	roi := 0.0
	if totalF > 0 {
		roi = returns / totalF * 100
	}
	roiRounded := math.Round(roi*100) / 100
	return ppkStat{
		Owner:                 owner,
		TotalValue:            pyFloat(totalValue),
		EmployeeContributed:   pyFloat(employeeF),
		EmployerContributed:   pyFloat(employerF),
		GovernmentContributed: pyFloat(governmentF),
		TotalContributed:      pyFloat(totalF),
		Returns:               pyFloat(returns),
		ROIPercentage:         pyFloat(roiRounded),
	}, true, nil
}

// LimitsForYear serves GET /api/retirement/limits/{year}.
func (h *Handler) LimitsForYear(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		writeValidationError(w, "year", "must be an integer", chi.URLParam(r, "year"))
		return
	}
	limits, err := h.store.LimitsForYear(r.Context(), year)
	if err != nil {
		h.logger.Error("limits", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]limitResponse, 0, len(limits))
	for _, l := range limits {
		amt, _ := l.LimitAmount.Float64()
		out = append(out, limitResponse{
			ID: l.ID, Year: l.Year, AccountWrapper: l.AccountWrapper,
			Owner: l.Owner, LimitAmount: pyFloat(amt), Notes: l.Notes,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// UpsertLimit serves PUT /api/retirement/limits/{year}/{wrapper}/{owner}.
func (h *Handler) UpsertLimit(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		writeValidationError(w, "year", "must be an integer", chi.URLParam(r, "year"))
		return
	}
	wrapper := chi.URLParam(r, "wrapper")
	owner := chi.URLParam(r, "owner")
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	req, vErr := buildLimitRequest(raw, h.now)
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	l, err := h.store.UpsertLimit(r.Context(), year, wrapper, owner, req.LimitAmount, req.Notes)
	if err != nil {
		h.logger.Error("upsert limit", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	amt, _ := l.LimitAmount.Float64()
	writeJSON(w, http.StatusOK, limitResponse{
		ID: l.ID, Year: l.Year, AccountWrapper: l.AccountWrapper,
		Owner: l.Owner, LimitAmount: pyFloat(amt), Notes: l.Notes,
	})
}

// GeneratePPKContributions serves POST /api/retirement/ppk-contributions/generate.
func (h *Handler) GeneratePPKContributions(w http.ResponseWriter, r *http.Request) {
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	req, vErr := buildGenerateRequest(raw, h.now)
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	firstDay := time.Date(req.Year, time.Month(req.Month), 1, 0, 0, 0, 0, time.UTC)
	gross, err := h.store.CurrentSalaryFor(r.Context(), req.Owner, firstDay)
	if err != nil {
		if errors.Is(err, ErrNoSalary) {
			writeDetailError(w, http.StatusBadRequest,
				fmt.Sprintf("No salary record found for %s in %d/%d", req.Owner, req.Month, req.Year))
			return
		}
		h.logger.Error("ppk salary", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	employeeRate, employerRate, err := h.store.PersonaPPKRates(r.Context(), req.Owner)
	if err != nil {
		if errors.Is(err, ErrPersonaNotFound) {
			writeDetailError(w, http.StatusNotFound,
				fmt.Sprintf("Persona '%s' not found", req.Owner))
			return
		}
		h.logger.Error("ppk rates", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	hundred := decimal.NewFromInt(100)
	employeeAmt := gross.Mul(employeeRate).Div(hundred)
	employerAmt := gross.Mul(employerRate).Div(hundred)
	accountID, err := h.store.ActivePPKAccountForOwner(r.Context(), req.Owner)
	if err != nil {
		if errors.Is(err, ErrNoPPKAccount) {
			writeDetailError(w, http.StatusNotFound,
				fmt.Sprintf("No active PPK account found for %s. "+
					"Mark one PPK account as receiving contributions.", req.Owner))
			return
		}
		h.logger.Error("ppk account", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	lastDay := time.Date(req.Year, time.Month(req.Month)+1, 0, 0, 0, 0, 0, time.UTC)
	ids, err := h.store.InsertPPKContributions(r.Context(), PPKContribution{
		AccountID: accountID, EmployeeAmt: employeeAmt, EmployerAmt: employerAmt,
		Date: lastDay, Owner: req.Owner,
	})
	if err != nil {
		if errors.Is(err, ErrContributionsExist) {
			writeDetailError(w, http.StatusConflict,
				fmt.Sprintf("Contributions already exist for %d/%d", req.Month, req.Year))
			return
		}
		h.logger.Error("insert ppk contributions", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	grossF, _ := gross.Float64()
	empF, _ := employeeAmt.Float64()
	emprF, _ := employerAmt.Float64()
	writeJSON(w, http.StatusOK, ppkGenerateResponse{
		Owner: req.Owner, Month: req.Month, Year: req.Year,
		GrossSalary:         pyFloat(grossF),
		EmployeeAmount:      pyFloat(empF),
		EmployerAmount:      pyFloat(emprF),
		TotalAmount:         pyFloat(empF + emprF),
		TransactionsCreated: ids,
	})
}

func (h *Handler) resolveOwners(ctx context.Context, queryOwner string) ([]string, error) {
	if queryOwner != "" {
		return []string{queryOwner}, nil
	}
	return h.store.PersonaNames(ctx)
}

// --- request parsing ---

type limitRequest struct {
	Year           int
	AccountWrapper string
	Owner          string
	LimitAmount    decimal.Decimal
	Notes          *string
}

var validWrappers = map[string]struct{}{
	"IKE": {}, "IKZE": {}, "PPK": {},
}

func buildLimitRequest(raw map[string]json.RawMessage, now func() time.Time) (limitRequest, *validationError) {
	var r limitRequest
	currentYear := now().UTC().Year()
	year, vErr := requireIntRange(raw, "year", 2000, currentYear+10,
		fmt.Sprintf("Year must be between 2000 and %d", currentYear+10))
	if vErr != nil {
		return r, vErr
	}
	r.Year = year

	wrap, vErr := requireEnumString(raw, "account_wrapper", validWrappers)
	if vErr != nil {
		return r, vErr
	}
	r.AccountWrapper = wrap

	owner, vErr := requireString(raw, "owner", "Owner cannot be empty")
	if vErr != nil {
		return r, vErr
	}
	r.Owner = owner

	amt, vErr := requirePositiveDecimal(raw, "limit_amount", "Limit amount must be greater than 0")
	if vErr != nil {
		return r, vErr
	}
	r.LimitAmount = amt

	if v, ok := raw["notes"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return r, &validationError{Field: "notes", Msg: "must be a string"}
		}
		r.Notes = &s
	}
	return r, nil
}

type generateRequest struct {
	Owner string
	Month int
	Year  int
}

func buildGenerateRequest(raw map[string]json.RawMessage, now func() time.Time) (generateRequest, *validationError) {
	var r generateRequest
	owner, vErr := requireString(raw, "owner", "Owner cannot be empty")
	if vErr != nil {
		return r, vErr
	}
	r.Owner = owner

	month, vErr := requireIntRange(raw, "month", 1, 12, "Month must be between 1 and 12")
	if vErr != nil {
		return r, vErr
	}
	r.Month = month

	currentYear := now().UTC().Year()
	year, vErr := requireIntRange(raw, "year", 2019, currentYear+1,
		fmt.Sprintf("Year must be between 2019 and %d", currentYear+1))
	if vErr != nil {
		return r, vErr
	}
	r.Year = year
	return r, nil
}

// --- helpers ---

func requireString(raw map[string]json.RawMessage, key, emptyMsg string) (string, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return "", &validationError{Field: key, Msg: "Field required"}
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", &validationError{Field: key, Msg: "must be a string"}
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return "", &validationError{Field: key, Msg: emptyMsg}
	}
	return s, nil
}

func requireEnumString(raw map[string]json.RawMessage, key string, allowed map[string]struct{}) (string, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return "", &validationError{Field: key, Msg: "Field required"}
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", &validationError{Field: key, Msg: "must be a string"}
	}
	if _, ok := allowed[s]; !ok {
		return "", &validationError{Field: key, Msg: fmt.Sprintf("invalid value %q", s)}
	}
	return s, nil
}

func requireIntRange(raw map[string]json.RawMessage, key string, lo, hi int, msg string) (int, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return 0, &validationError{Field: key, Msg: "Field required"}
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return 0, &validationError{Field: key, Msg: "must be an integer"}
	}
	if n < lo || n > hi {
		return 0, &validationError{Field: key, Msg: msg}
	}
	return n, nil
}

func requirePositiveDecimal(raw map[string]json.RawMessage, key, msg string) (decimal.Decimal, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return decimal.Decimal{}, &validationError{Field: key, Msg: "Field required"}
	}
	d, err := decimal.NewFromString(string(bytes.TrimSpace(v)))
	if err != nil {
		return decimal.Decimal{}, &validationError{Field: key, Msg: "must be a number"}
	}
	if !d.IsPositive() {
		return decimal.Decimal{}, &validationError{Field: key, Msg: msg}
	}
	return d, nil
}

func isNull(v json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}

// wire type
type pyFloat float64

func (f pyFloat) MarshalJSON() ([]byte, error) {
	s := strconv.FormatFloat(float64(f), 'f', -1, 64)
	if !strings.ContainsRune(s, '.') {
		s += ".0"
	}
	return []byte(s), nil
}
