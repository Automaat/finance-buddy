package salaries

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/cpi"
)

var (
	validContractTypes = map[string]struct{}{
		"UOP": {}, "UZ": {}, "UoD": {}, "B2B": {},
	}
)

// response mirrors backend/app/schemas/salary_records.SalaryRecordResponse.
type response struct {
	ID           int      `json:"id"`
	Date         isoDate  `json:"date"`
	GrossAmount  pyFloat  `json:"gross_amount"`
	ContractType string   `json:"contract_type"`
	Company      string   `json:"company"`
	Owner        string   `json:"owner"`
	IsActive     bool     `json:"is_active"`
	CreatedAt    isoNaive `json:"created_at"`
}

// inflationContext mirrors backend/app/schemas/salary_records.InflationContext.
type inflationContext struct {
	Owner                    string   `json:"owner"`
	LastChangeDate           isoDate  `json:"last_change_date"`
	PreviousChangeDate       *isoDate `json:"previous_change_date"`
	PreviousSalary           *pyFloat `json:"previous_salary"`
	PreviousSalaryInTodayPLN *pyFloat `json:"previous_salary_in_today_pln"`
	CurrentSalary            pyFloat  `json:"current_salary"`
	RealChangePLN            *pyFloat `json:"real_change_pln"`
	RealChangePct            *pyFloat `json:"real_change_pct"`
	CPIAsOfYear              int      `json:"cpi_as_of_year"`
}

type listResponse struct {
	SalaryRecords      []response                  `json:"salary_records"`
	TotalCount         int                         `json:"total_count"`
	CurrentSalaries    map[string]*pyFloat         `json:"current_salaries"`
	InflationContext   map[string]inflationContext `json:"inflation_context"`
	AvailableCompanies []string                    `json:"available_companies"`
}

// Handler is the HTTP boundary for /api/salaries.
type Handler struct {
	store    *Store
	cpiStore *cpi.Store
	logger   *slog.Logger
	now      func() time.Time
}

// NewHandler wires the dependencies. The cpi.Store is read-only here —
// salaries reuse the CPI index to compute previous_salary_in_today_pln.
func NewHandler(store *Store, cpiStore *cpi.Store, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, cpiStore: cpiStore, logger: logger, now: time.Now}
}

func toResponse(r *SalaryRecord) response {
	gross, _ := r.GrossAmount.Float64()
	return response{
		ID:           r.ID,
		Date:         isoDate(r.Date),
		GrossAmount:  pyFloat(gross),
		ContractType: r.ContractType,
		Company:      r.Company,
		Owner:        r.Owner,
		IsActive:     r.IsActive,
		CreatedAt:    isoNaive(r.CreatedAt.UTC()),
	}
}

// List serves GET /api/salaries.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter, vErr := parseListFilter(r.URL.Query())
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	rows, companies, err := h.store.List(r.Context(), filter)
	if err != nil {
		h.logger.Error("list salaries", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	today := truncateDay(h.now().UTC())
	personas, err := h.store.ActivePersonaNames(r.Context())
	if err != nil {
		h.logger.Error("persona names", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	currentSalaries, err := h.store.CurrentSalaryByPersona(r.Context(), personas, today)
	if err != nil {
		h.logger.Error("current salaries", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	inflationCtx, err := h.buildInflationContext(r.Context(), personas, today)
	if err != nil {
		h.logger.Error("inflation context", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := listResponse{
		SalaryRecords:      make([]response, 0, len(rows)),
		TotalCount:         len(rows),
		CurrentSalaries:    salariesToFloatMap(personas, currentSalaries),
		InflationContext:   inflationCtx,
		AvailableCompanies: companies,
	}
	for i := range rows {
		out.SalaryRecords = append(out.SalaryRecords, toResponse(&rows[i]))
	}
	writeJSON(w, http.StatusOK, out)
}

// Get serves GET /api/salaries/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	rec, err := h.store.Get(r.Context(), id)
	if err != nil {
		h.writeStoreError(w, err, id)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(rec))
}

// Create serves POST /api/salaries.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	req, vErr := buildCreateRequest(raw, h.now)
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	created, err := h.store.Create(r.Context(), &SalaryRecord{
		Date:         req.Date,
		GrossAmount:  req.GrossAmount,
		ContractType: req.ContractType,
		Company:      req.Company,
		Owner:        req.Owner,
	})
	if err != nil {
		if errors.Is(err, ErrDuplicate) {
			writeDetailError(w, http.StatusConflict,
				fmt.Sprintf("Salary record for %s on %s already exists",
					req.Owner, req.Date.Format("2006-01-02")))
			return
		}
		h.logger.Error("create salary", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(created))
}

// Update serves PATCH /api/salaries/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	patch, vErr := buildUpdatePatch(raw, h.now)
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	updated, err := h.store.Update(r.Context(), id, patch)
	if err != nil {
		h.writeStoreError(w, err, id)
		return
	}
	writeJSON(w, http.StatusOK, toResponse(updated))
}

// Delete serves DELETE /api/salaries/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		h.writeStoreError(w, err, id)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeStoreError(w http.ResponseWriter, err error, id int) {
	if errors.Is(err, ErrNotFound) {
		writeDetailError(w, http.StatusNotFound,
			fmt.Sprintf("Salary record with id %d not found", id))
		return
	}
	h.logger.Error("salary store", "err", err)
	writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
}

// buildInflationContext mirrors Python's _build_inflation_context: for each
// persona with at least two recent records, compute previous_salary_in_today_pln
// via the CPI index, then derive real_change_pln / pct.
func (h *Handler) buildInflationContext(
	ctx context.Context, personaNames []string, today time.Time,
) (map[string]inflationContext, error) {
	if len(personaNames) == 0 {
		return map[string]inflationContext{}, nil
	}
	asOfYear, hasYear, err := h.cpiStore.LatestKnownYear(ctx)
	if err != nil {
		return nil, err
	}
	yoyMap, err := h.cpiStore.LoadYoYMap(ctx)
	if err != nil {
		return nil, err
	}
	if !hasYear || len(yoyMap) == 0 {
		return map[string]inflationContext{}, nil
	}
	indexMap := cpi.CumulativeIndex(yoyMap)
	recent, err := h.store.RecentTwoByPersona(ctx, personaNames, today)
	if err != nil {
		return nil, err
	}
	out := map[string]inflationContext{}
	for owner, recs := range recent {
		if len(recs) < 2 {
			continue
		}
		current := recs[0]
		previous := recs[1]
		prevAmount, _ := previous.GrossAmount.Float64()
		prevInToday, err := cpi.AdjustWithIndex(indexMap, prevAmount, previous.Date, today)
		if err != nil {
			continue
		}
		curAmount, _ := current.GrossAmount.Float64()
		realChangePLN := curAmount - prevInToday
		var realChangePct *pyFloat
		if prevInToday != 0 {
			pct := pyFloat(realChangePLN / prevInToday * 100)
			realChangePct = &pct
		}
		prevDate := isoDate(previous.Date)
		prevSalary := pyFloat(prevAmount)
		prevTodayPLN := pyFloat(prevInToday)
		realChange := pyFloat(realChangePLN)
		out[owner] = inflationContext{
			Owner:                    owner,
			LastChangeDate:           isoDate(current.Date),
			PreviousChangeDate:       &prevDate,
			PreviousSalary:           &prevSalary,
			PreviousSalaryInTodayPLN: &prevTodayPLN,
			CurrentSalary:            pyFloat(curAmount),
			RealChangePLN:            &realChange,
			RealChangePct:            realChangePct,
			CPIAsOfYear:              asOfYear,
		}
	}
	return out, nil
}

func salariesToFloatMap(
	personaNames []string, values map[string]decimal.Decimal,
) map[string]*pyFloat {
	out := map[string]*pyFloat{}
	for _, name := range personaNames {
		v, ok := values[name]
		if !ok {
			out[name] = nil
			continue
		}
		f, _ := v.Float64()
		pf := pyFloat(f)
		out[name] = &pf
	}
	return out
}

func parseListFilter(q map[string][]string) (ListFilter, *validationError) {
	f := ListFilter{}
	if v := first(q["owner"]); v != "" {
		s := v
		f.Owner = &s
	}
	if v := first(q["company"]); v != "" {
		s := v
		f.Company = &s
	}
	if v := first(q["date_from"]); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return f, &validationError{Field: "date_from", Msg: "must be YYYY-MM-DD"}
		}
		f.DateFrom = &t
	}
	if v := first(q["date_to"]); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return f, &validationError{Field: "date_to", Msg: "must be YYYY-MM-DD"}
		}
		f.DateTo = &t
	}
	return f, nil
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func parseIDParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.Atoi(raw)
	if err != nil {
		writeValidationError(w, "salary_id", "must be an integer", raw)
		return 0, false
	}
	return id, true
}

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// --- shared wire types ---

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

type isoNaive time.Time

func (t isoNaive) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(t).Format("2006-01-02T15:04:05.999999") + `"`), nil
}

type pyFloat float64

func (f pyFloat) MarshalJSON() ([]byte, error) {
	s := strconv.FormatFloat(float64(f), 'f', -1, 64)
	if !strings.ContainsRune(s, '.') {
		s += ".0"
	}
	return []byte(s), nil
}
