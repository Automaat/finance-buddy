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
	OwnerUserID         *int     `json:"owner_user_id"`
	LimitAmount         *pyFloat `json:"limit_amount"`
	TotalContributed    pyFloat  `json:"total_contributed"`
	EmployeeContributed pyFloat  `json:"employee_contributed"`
	EmployerContributed pyFloat  `json:"employer_contributed"`
	Remaining           *pyFloat `json:"remaining"`
	PercentageUsed      *pyFloat `json:"percentage_used"`
	IsWarning           bool     `json:"is_warning"`
}

type ppkStat struct {
	OwnerUserID           *int    `json:"owner_user_id"`
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
	OwnerUserID    *int    `json:"owner_user_id"`
	LimitAmount    pyFloat `json:"limit_amount"`
	Notes          *string `json:"notes"`
}

type ppkGenerateResponse struct {
	OwnerUserID         *int    `json:"owner_user_id"`
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
	owners, err := h.resolveOwners(r.Context(), r.URL.Query().Get("owner_user_id"))
	if err != nil {
		if errors.Is(err, errBadOwnerParam) {
			writeValidationError(w, "owner_user_id", "must be an integer",
				r.URL.Query().Get("owner_user_id"))
			return
		}
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

func (h *Handler) buildYearlyStat(ctx context.Context, year int, wrapper string, ownerUserID *int) (yearlyStat, bool, error) {
	accountIDs, err := h.store.AccountIDsForWrapper(ctx, wrapper, ownerUserID)
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
	limit, configured, err := h.store.LimitConfigured(ctx, year, wrapper, ownerUserID)
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
	lAmt := pyFloat(limitF)
	rem := pyFloat(remaining)
	pct := pyFloat(math.Round(percentage*10) / 10)
	return yearlyStat{
		Year: year, AccountWrapper: wrapper, OwnerUserID: ownerUserID,
		LimitAmount:         &lAmt,
		TotalContributed:    pyFloat(totalF),
		EmployeeContributed: pyFloat(employeeF),
		EmployerContributed: pyFloat(employerF),
		Remaining:           &rem,
		PercentageUsed:      &pct,
		// is_warning uses the unrounded percentage — Python computes it from
		// the raw value, so 89.95% must NOT flip the flag even though it
		// rounds to 90.0 in percentage_used.
		IsWarning: percentage >= 90,
	}, true, nil
}

// PPKStats serves GET /api/retirement/ppk-stats.
func (h *Handler) PPKStats(w http.ResponseWriter, r *http.Request) {
	owners, err := h.resolveOwners(r.Context(), r.URL.Query().Get("owner_user_id"))
	if err != nil {
		if errors.Is(err, errBadOwnerParam) {
			writeValidationError(w, "owner_user_id", "must be an integer",
				r.URL.Query().Get("owner_user_id"))
			return
		}
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

func (h *Handler) buildPPKStat(ctx context.Context, ownerUserID *int) (ppkStat, bool, error) {
	accountIDs, err := h.store.AccountIDsForWrapper(ctx, "PPK", ownerUserID)
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
		OwnerUserID:           ownerUserID,
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
			OwnerUserID: l.OwnerUserID, LimitAmount: pyFloat(amt), Notes: l.Notes,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// UpsertLimit serves PUT /api/retirement/limits/{year}/{wrapper}/{owner_user_id}.
func (h *Handler) UpsertLimit(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		writeValidationError(w, "year", "must be an integer", chi.URLParam(r, "year"))
		return
	}
	wrapper := chi.URLParam(r, "wrapper")
	ownerParam := chi.URLParam(r, "owner_user_id")
	// The literal "null" addresses the jointly-owned (Shared) limit row.
	var ownerUserID *int
	if ownerParam != "null" {
		n, err := strconv.Atoi(ownerParam)
		if err != nil {
			writeValidationError(w, "owner_user_id", "must be an integer", ownerParam)
			return
		}
		ownerUserID = &n
	}
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
	l, err := h.store.UpsertLimit(r.Context(), year, wrapper, ownerUserID, req.LimitAmount, req.Notes)
	if err != nil {
		h.logger.Error("upsert limit", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	amt, _ := l.LimitAmount.Float64()
	writeJSON(w, http.StatusOK, limitResponse{
		ID: l.ID, Year: l.Year, AccountWrapper: l.AccountWrapper,
		OwnerUserID: l.OwnerUserID, LimitAmount: pyFloat(amt), Notes: l.Notes,
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
	gross, err := h.store.CurrentSalaryFor(r.Context(), req.OwnerUserID, firstDay)
	if err != nil {
		if errors.Is(err, ErrNoSalary) {
			writeDetailError(w, http.StatusBadRequest,
				fmt.Sprintf("No salary record found for user %s in %d/%d",
					ownerLabel(req.OwnerUserID), req.Month, req.Year))
			return
		}
		h.logger.Error("ppk salary", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	employeeRate, employerRate, err := h.store.UserPPKRates(r.Context(), req.OwnerUserID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			writeDetailError(w, http.StatusNotFound,
				fmt.Sprintf("User %s not found", ownerLabel(req.OwnerUserID)))
			return
		}
		h.logger.Error("ppk rates", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	hundred := decimal.NewFromInt(100)
	employeeAmt := gross.Mul(employeeRate).Div(hundred)
	employerAmt := gross.Mul(employerRate).Div(hundred)
	accountID, err := h.store.ActivePPKAccountForOwner(r.Context(), req.OwnerUserID)
	if err != nil {
		if errors.Is(err, ErrNoPPKAccount) {
			writeDetailError(w, http.StatusNotFound,
				fmt.Sprintf("No active PPK account found for user %s. "+
					"Mark one PPK account as receiving contributions.",
					ownerLabel(req.OwnerUserID)))
			return
		}
		h.logger.Error("ppk account", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	lastDay := time.Date(req.Year, time.Month(req.Month)+1, 0, 0, 0, 0, 0, time.UTC)
	ids, err := h.store.InsertPPKContributions(r.Context(), PPKContribution{
		AccountID: accountID, EmployeeAmt: employeeAmt, EmployerAmt: employerAmt,
		Date: lastDay, OwnerUserID: req.OwnerUserID,
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
		OwnerUserID: req.OwnerUserID, Month: req.Month, Year: req.Year,
		GrossSalary:         pyFloat(grossF),
		EmployeeAmount:      pyFloat(empF),
		EmployerAmount:      pyFloat(emprF),
		TotalAmount:         pyFloat(empF + emprF),
		TransactionsCreated: ids,
	})
}

// errBadOwnerParam marks a non-integer owner_user_id query value.
var errBadOwnerParam = errors.New("owner_user_id must be an integer")

func (h *Handler) resolveOwners(ctx context.Context, queryOwner string) ([]*int, error) {
	if queryOwner != "" {
		n, err := strconv.Atoi(queryOwner)
		if err != nil {
			return nil, errBadOwnerParam
		}
		return []*int{&n}, nil
	}
	ids, err := h.store.UserIDs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*int, 0, len(ids))
	for i := range ids {
		id := ids[i]
		out = append(out, &id)
	}
	return out, nil
}

// ownerLabel renders an owner_user_id for error messages.
func ownerLabel(id *int) string {
	if id == nil {
		return "Shared"
	}
	return strconv.Itoa(*id)
}

// --- request parsing ---

type limitRequest struct {
	Year           int
	AccountWrapper string
	OwnerUserID    *int
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

	owner, vErr := requireIntOrNull(raw, "owner_user_id")
	if vErr != nil {
		return r, vErr
	}
	r.OwnerUserID = owner

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
	OwnerUserID *int
	Month       int
	Year        int
}

func buildGenerateRequest(raw map[string]json.RawMessage, now func() time.Time) (generateRequest, *validationError) {
	var r generateRequest
	owner, vErr := requireIntOrNull(raw, "owner_user_id")
	if vErr != nil {
		return r, vErr
	}
	r.OwnerUserID = owner

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

// requireIntOrNull reads an integer key that must be present; an explicit
// null is allowed and yields nil (jointly owned).
func requireIntOrNull(raw map[string]json.RawMessage, key string) (*int, *validationError) {
	v, ok := raw[key]
	if !ok {
		return nil, &validationError{Field: key, Msg: "Field required"}
	}
	if isNull(v) {
		return nil, nil
	}
	var n int
	if err := json.Unmarshal(v, &n); err != nil {
		return nil, &validationError{Field: key, Msg: "must be an integer"}
	}
	return &n, nil
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
