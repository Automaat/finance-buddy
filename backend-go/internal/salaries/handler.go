package salaries

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/cpi"
	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

var validContractTypes = map[string]struct{}{
	"UOP": {}, "UZ": {}, "UoD": {}, "B2B": {},
}

// response mirrors backend/app/schemas/salary_records.SalaryRecordResponse.
// OwnerUserID is the owning household member; nil means jointly owned.
type response struct {
	ID           int           `json:"id"`
	Date         wire.IsoDate  `json:"date"`
	GrossAmount  wire.PyFloat  `json:"gross_amount"`
	ContractType string        `json:"contract_type"`
	Company      string        `json:"company"`
	OwnerUserID  *int          `json:"owner_user_id"`
	IsActive     bool          `json:"is_active"`
	CreatedAt    wire.IsoNaive `json:"created_at"`
}

// inflationContext mirrors backend/app/schemas/salary_records.InflationContext.
type inflationContext struct {
	OwnerUserID              int           `json:"owner_user_id"`
	LastChangeDate           wire.IsoDate  `json:"last_change_date"`
	PreviousChangeDate       *wire.IsoDate `json:"previous_change_date"`
	PreviousSalary           *wire.PyFloat `json:"previous_salary"`
	PreviousSalaryInTodayPLN *wire.PyFloat `json:"previous_salary_in_today_pln"`
	CurrentSalary            wire.PyFloat  `json:"current_salary"`
	RealChangePLN            *wire.PyFloat `json:"real_change_pln"`
	RealChangePct            *wire.PyFloat `json:"real_change_pct"`
	CPIAsOfYear              int           `json:"cpi_as_of_year"`
}

// listResponse keys current_salaries and inflation_context by owner_user_id
// (JSON serializes the integer keys as strings).
type listResponse struct {
	SalaryRecords      []response               `json:"salary_records"`
	TotalCount         int                      `json:"total_count"`
	CurrentSalaries    map[int]*wire.PyFloat    `json:"current_salaries"`
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
		Date:         wire.IsoDate(r.Date),
		GrossAmount:  wire.PyFloat(gross),
		ContractType: r.ContractType,
		Company:      r.Company,
		OwnerUserID:  r.OwnerUserID,
		IsActive:     r.IsActive,
		CreatedAt:    wire.IsoNaive(r.CreatedAt.UTC()),
	}
}

// List serves GET /api/salaries.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter, vErr := parseListFilter(r.URL.Query())
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	rows, companies, err := h.store.List(r.Context(), filter)
	if err != nil {
		h.logger.Error("list salaries", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	today := truncateDay(h.now().UTC())
	userIDs, err := h.store.ActiveOwnerUserIDs(r.Context())
	if err != nil {
		h.logger.Error("owner user ids", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	recentSalaries, err := h.store.RecentTwoByUser(r.Context(), userIDs, today)
	if err != nil {
		h.logger.Error("recent salaries", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	currentSalaries := currentSalaryFromRecent(recentSalaries)
	inflationCtx, err := h.buildInflationContext(r.Context(), recentSalaries, today)
	if err != nil {
		h.logger.Error("inflation context", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
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
	httputil.WriteJSON(w, http.StatusOK, out)
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
	httputil.WriteJSON(w, http.StatusOK, toResponse(rec))
}

// Create serves POST /api/salaries.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	raw := map[string]json.RawMessage{}
	if !httputil.DecodeJSON(w, r, 1<<16, &raw) {
		return
	}
	req, vErr := buildCreateRequest(raw, h.now)
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
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
			httputil.WriteDetailError(w, http.StatusConflict,
				fmt.Sprintf("A salary record on %s already exists for this owner",
					req.Date.Format("2006-01-02")))
			return
		}
		h.logger.Error("create salary", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toResponse(created))
}

// Update serves PATCH /api/salaries/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	raw := map[string]json.RawMessage{}
	if !httputil.DecodeJSON(w, r, 1<<16, &raw) {
		return
	}
	patch, vErr := buildUpdatePatch(raw, h.now)
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
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
			httputil.WriteDetailError(w, http.StatusConflict,
				fmt.Sprintf("A salary record on %s conflicts with an existing record",
					date))
			return
		}
		h.writeStoreError(w, err, id)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toResponse(updated))
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
		httputil.WriteDetailError(w, http.StatusNotFound,
			fmt.Sprintf("Salary record with id %d not found", id))
		return
	}
	h.logger.Error("salary store", "err", err)
	httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
}

// buildInflationContext mirrors Python's _build_inflation_context: for each
// persona with at least two recent records, compute previous_salary_in_today_pln
// via the CPI index, then derive real_change_pln / pct.
func (h *Handler) buildInflationContext(
	ctx context.Context, recent map[int][]SalaryRecord, today time.Time,
) (map[int]inflationContext, error) {
	if !hasPreviousSalary(recent) {
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
		var realChangePct *wire.PyFloat
		if prevInToday != 0 {
			pct := wire.PyFloat(realChangePLN / prevInToday * 100)
			realChangePct = &pct
		}
		prevDate := wire.IsoDate(previous.Date)
		prevSalary := wire.PyFloat(prevAmount)
		prevTodayPLN := wire.PyFloat(prevInToday)
		realChange := wire.PyFloat(realChangePLN)
		out[ownerID] = inflationContext{
			OwnerUserID:              ownerID,
			LastChangeDate:           wire.IsoDate(current.Date),
			PreviousChangeDate:       &prevDate,
			PreviousSalary:           &prevSalary,
			PreviousSalaryInTodayPLN: &prevTodayPLN,
			CurrentSalary:            wire.PyFloat(curAmount),
			RealChangePLN:            &realChange,
			RealChangePct:            realChangePct,
			CPIAsOfYear:              asOfYear,
		}
	}
	return out, nil
}

func currentSalaryFromRecent(recent map[int][]SalaryRecord) map[int]decimal.Decimal {
	out := map[int]decimal.Decimal{}
	for ownerID, recs := range recent {
		if len(recs) == 0 {
			continue
		}
		out[ownerID] = recs[0].GrossAmount
	}
	return out
}

func hasPreviousSalary(recent map[int][]SalaryRecord) bool {
	for _, recs := range recent {
		if len(recs) >= 2 {
			return true
		}
	}
	return false
}

func salariesToFloatMap(
	userIDs []int, values map[int]decimal.Decimal,
) map[int]*wire.PyFloat {
	out := map[int]*wire.PyFloat{}
	for _, id := range userIDs {
		v, ok := values[id]
		if !ok {
			out[id] = nil
			continue
		}
		f, _ := v.Float64()
		pf := wire.PyFloat(f)
		out[id] = &pf
	}
	return out
}

func parseListFilter(q map[string][]string) (ListFilter, *httputil.ValidationError) {
	parsed, vErr := validation.ParseOwnerCompanyDateFilter(q)
	if vErr != nil {
		return ListFilter{}, vErr
	}
	return ListFilter{
		OwnerUserID: parsed.OwnerUserID,
		DateFrom:    parsed.DateFrom,
		DateTo:      parsed.DateTo,
		Company:     parsed.Company,
	}, nil
}

func parseIDParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	return httputil.PathIntField(w, r, "id", "salary_id")
}

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
