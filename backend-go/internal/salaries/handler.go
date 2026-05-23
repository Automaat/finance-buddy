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

var validContractTypes = map[string]struct{}{
	"UOP": {}, "UZ": {}, "UoD": {}, "B2B": {},
}

// response mirrors backend/app/schemas/salary_records.SalaryRecordResponse.
// OwnerUserID is the owning household member; nil means jointly owned.
type response struct {
	ID           int      `json:"id"`
	Date         isoDate  `json:"date"`
	GrossAmount  pyFloat  `json:"gross_amount"`
	ContractType string   `json:"contract_type"`
	Company      string   `json:"company"`
	OwnerUserID  *int     `json:"owner_user_id"`
	IsActive     bool     `json:"is_active"`
	CreatedAt    isoNaive `json:"created_at"`
}

// inflationContext mirrors backend/app/schemas/salary_records.InflationContext.
type inflationContext struct {
	OwnerUserID              int      `json:"owner_user_id"`
	LastChangeDate           isoDate  `json:"last_change_date"`
	PreviousChangeDate       *isoDate `json:"previous_change_date"`
	PreviousSalary           *pyFloat `json:"previous_salary"`
	PreviousSalaryInTodayPLN *pyFloat `json:"previous_salary_in_today_pln"`
	CurrentSalary            pyFloat  `json:"current_salary"`
	RealChangePLN            *pyFloat `json:"real_change_pln"`
	RealChangePct            *pyFloat `json:"real_change_pct"`
	CPIAsOfYear              int      `json:"cpi_as_of_year"`
}

// listResponse keys current_salaries and inflation_context by owner_user_id
// (JSON serializes the integer keys as strings).
type listResponse struct {
	SalaryRecords      []response               `json:"salary_records"`
	TotalCount         int                      `json:"total_count"`
	CurrentSalaries    map[int]*pyFloat         `json:"current_salaries"`
	InflationContext   map[int]inflationContext `json:"inflation_context"`
	AvailableCompanies []string                 `json:"available_companies"`
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
		OwnerUserID:  r.OwnerUserID,
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
	userIDs, err := h.store.ActiveOwnerUserIDs(r.Context())
	if err != nil {
		h.logger.Error("owner user ids", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	currentSalaries, err := h.store.CurrentSalaryByUser(r.Context(), userIDs, today)
	if err != nil {
		h.logger.Error("current salaries", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	inflationCtx, err := h.buildInflationContext(r.Context(), userIDs, today)
	if err != nil {
		h.logger.Error("inflation context", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := listResponse{
		SalaryRecords:      make([]response, 0, len(rows)),
		TotalCount:         len(rows),
		CurrentSalaries:    salariesToFloatMap(userIDs, currentSalaries),
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
		OwnerUserID:  req.OwnerUserID,
	})
	if err != nil {
		if errors.Is(err, ErrDuplicate) {
			writeDetailError(w, http.StatusConflict,
				fmt.Sprintf("A salary record on %s already exists for this owner",
					req.Date.Format("2006-01-02")))
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
		if errors.Is(err, ErrDuplicate) {
			// Need the merged record's date for the message; refetch.
			current, getErr := h.store.Get(r.Context(), id)
			date := ""
			if getErr == nil {
				if patch.Date != nil {
					date = patch.Date.Format("2006-01-02")
				} else {
					date = current.Date.Format("2006-01-02")
				}
			}
			writeDetailError(w, http.StatusConflict,
				fmt.Sprintf("A salary record on %s conflicts with an existing record",
					date))
			return
		}
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
	ctx context.Context, userIDs []int, today time.Time,
) (map[int]inflationContext, error) {
	if len(userIDs) == 0 {
		return map[int]inflationContext{}, nil
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
		return map[int]inflationContext{}, nil
	}
	indexMap := cpi.CumulativeIndex(yoyMap)
	recent, err := h.store.RecentTwoByUser(ctx, userIDs, today)
	if err != nil {
		return nil, err
	}
	out := map[int]inflationContext{}
	for ownerID, recs := range recent {
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
		out[ownerID] = inflationContext{
			OwnerUserID:              ownerID,
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
	userIDs []int, values map[int]decimal.Decimal,
) map[int]*pyFloat {
	out := map[int]*pyFloat{}
	for _, id := range userIDs {
		v, ok := values[id]
		if !ok {
			out[id] = nil
			continue
		}
		f, _ := v.Float64()
		pf := pyFloat(f)
		out[id] = &pf
	}
	return out
}

func parseListFilter(q map[string][]string) (ListFilter, *validationError) {
	f := ListFilter{}
	if v := first(q["owner_user_id"]); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return f, &validationError{Field: "owner_user_id", Msg: "must be an integer"}
		}
		f.OwnerUserID = &n
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
