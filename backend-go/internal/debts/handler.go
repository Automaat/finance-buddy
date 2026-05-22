package debts

import (
	"bytes"
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
)

var validDebtTypes = map[string]struct{}{
	"mortgage": {}, "installment_0percent": {},
}

type response struct {
	ID                 int      `json:"id"`
	AccountID          int      `json:"account_id"`
	AccountName        string   `json:"account_name"`
	AccountOwnerUserID *int     `json:"account_owner_user_id"`
	Name               string   `json:"name"`
	DebtType           string   `json:"debt_type"`
	StartDate          isoDate  `json:"start_date"`
	InitialAmount      pyFloat  `json:"initial_amount"`
	InterestRate       pyFloat  `json:"interest_rate"`
	Currency           string   `json:"currency"`
	Notes              *string  `json:"notes"`
	IsActive           bool     `json:"is_active"`
	CreatedAt          isoNaive `json:"created_at"`
	LatestBalance      *pyFloat `json:"latest_balance"`
	LatestBalanceDate  *isoDate `json:"latest_balance_date"`
	TotalPaid          pyFloat  `json:"total_paid"`
	InterestPaid       pyFloat  `json:"interest_paid"`
}

type listResponse struct {
	Debts              []response `json:"debts"`
	TotalCount         int        `json:"total_count"`
	TotalInitialAmount pyFloat    `json:"total_initial_amount"`
	ActiveDebtsCount   int        `json:"active_debts_count"`
}

// Handler is the HTTP boundary.
type Handler struct {
	store  *Store
	logger *slog.Logger
}

// NewHandler wires the store + logger.
func NewHandler(store *Store, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, logger: logger}
}

type debtMetrics struct {
	latestBalance     *decimal.Decimal
	latestBalanceDate *time.Time
	totalPaid         decimal.Decimal
}

func computeMetrics(initial decimal.Decimal, balance *decimal.Decimal, totalPaid decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	if balance == nil {
		return decimal.Zero, decimal.Zero
	}
	principalPaid := initial.Sub(*balance)
	return principalPaid, totalPaid.Sub(principalPaid)
}

func toResponse(d Debt, a AccountInfo, m debtMetrics) response {
	initial, _ := d.InitialAmount.Float64()
	rate, _ := d.InterestRate.Float64()
	totalPaid, _ := m.totalPaid.Float64()
	_, interestPaid := computeMetrics(d.InitialAmount, m.latestBalance, m.totalPaid)
	ip, _ := interestPaid.Float64()
	out := response{
		ID: d.ID, AccountID: d.AccountID, AccountName: a.Name, AccountOwnerUserID: a.OwnerUserID,
		Name: d.Name, DebtType: d.DebtType, StartDate: isoDate(d.StartDate),
		InitialAmount: pyFloat(initial), InterestRate: pyFloat(rate),
		Currency: d.Currency, Notes: d.Notes, IsActive: d.IsActive,
		CreatedAt: isoNaive(d.CreatedAt.UTC()),
		TotalPaid: pyFloat(totalPaid), InterestPaid: pyFloat(ip),
	}
	if m.latestBalance != nil {
		f, _ := m.latestBalance.Float64()
		pf := pyFloat(f)
		out.LatestBalance = &pf
	}
	if m.latestBalanceDate != nil {
		d := isoDate(*m.latestBalanceDate)
		out.LatestBalanceDate = &d
	}
	return out
}

// List serves GET /api/debts.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := ListFilter{}
	if v := q.Get("account_id"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			writeValidationError(w, "account_id", "must be an integer", v)
			return
		}
		f.AccountID = &n
	}
	if v := q.Get("debt_type"); v != "" {
		s := v
		f.DebtType = &s
	}
	rows, err := h.store.ListAll(r.Context(), f)
	if err != nil {
		h.logger.Error("list debts", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	ids := make([]int, 0, len(rows))
	for i := range rows {
		ids = append(ids, rows[i].Debt.AccountID)
	}
	balances, err := h.store.LatestBalancesByAccount(r.Context(), ids)
	if err != nil {
		h.logger.Error("latest balances", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	totals, err := h.store.TotalPaidByAccount(r.Context(), ids)
	if err != nil {
		h.logger.Error("total paid", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := listResponse{Debts: make([]response, 0, len(rows))}
	totalInitial := decimal.Zero
	for i := range rows {
		dwa := &rows[i]
		m := metricsFor(dwa.Debt.AccountID, balances, totals)
		out.Debts = append(out.Debts, toResponse(dwa.Debt, dwa.Account, m))
		totalInitial = totalInitial.Add(dwa.Debt.InitialAmount)
	}
	out.TotalCount = len(out.Debts)
	out.ActiveDebtsCount = len(out.Debts)
	f64, _ := totalInitial.Float64()
	out.TotalInitialAmount = pyFloat(f64)
	writeJSON(w, http.StatusOK, out)
}

// Get serves GET /api/debts/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r, "id", "debt_id")
	if !ok {
		return
	}
	dwa, err := h.store.Get(r.Context(), id)
	if err != nil {
		h.writeDebtError(w, err, id)
		return
	}
	m, err := h.metricsForDebt(r.Context(), dwa.Debt.AccountID)
	if err != nil {
		h.logger.Error("debt metrics", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toResponse(dwa.Debt, dwa.Account, m))
}

// Create serves POST /api/accounts/{account_id}/debts.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	accountID, err := strconv.Atoi(chi.URLParam(r, "account_id"))
	if err != nil {
		writeValidationError(w, "account_id", "must be an integer", chi.URLParam(r, "account_id"))
		return
	}
	account, err := h.store.LoadAccount(r.Context(), accountID)
	if err != nil {
		name := ""
		if account != nil {
			name = account.Name
		}
		h.writeAccountError(w, err, accountID, name)
		return
	}
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	req, vErr := buildCreateRequest(raw)
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	created, err := h.store.Create(r.Context(), &Debt{
		AccountID: accountID,
		Name:      req.Name, DebtType: req.DebtType, StartDate: req.StartDate,
		InitialAmount: req.InitialAmount, InterestRate: req.InterestRate,
		Currency: req.Currency, Notes: req.Notes,
	})
	if err != nil {
		if errors.Is(err, ErrDuplicateForAccount) {
			writeDetailError(w, http.StatusConflict,
				fmt.Sprintf("Debt already exists for account '%s'", account.Name))
			return
		}
		h.logger.Error("create debt", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	m, err := h.metricsForDebt(r.Context(), accountID)
	if err != nil {
		h.logger.Error("debt metrics", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(*created, *account, m))
}

// Update serves PUT /api/debts/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r, "id", "debt_id")
	if !ok {
		return
	}
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	patch, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	dwa, err := h.store.Update(r.Context(), id, patch)
	if err != nil {
		h.writeDebtError(w, err, id)
		return
	}
	m, err := h.metricsForDebt(r.Context(), dwa.Debt.AccountID)
	if err != nil {
		h.logger.Error("debt metrics", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toResponse(dwa.Debt, dwa.Account, m))
}

// Delete serves DELETE /api/debts/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r, "id", "debt_id")
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		h.writeDebtError(w, err, id)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) metricsForDebt(ctx context.Context, accountID int) (debtMetrics, error) {
	lb, found, err := h.store.LatestBalanceForAccount(ctx, accountID)
	if err != nil {
		return debtMetrics{}, err
	}
	totals, err := h.store.TotalPaidByAccount(ctx, []int{accountID})
	if err != nil {
		return debtMetrics{}, err
	}
	m := debtMetrics{totalPaid: totals[accountID]}
	if found {
		balance := lb.Value
		date := lb.Date
		m.latestBalance = &balance
		m.latestBalanceDate = &date
	}
	return m, nil
}

func metricsFor(accountID int, balances map[int]LatestBalance, totals map[int]decimal.Decimal) debtMetrics {
	m := debtMetrics{totalPaid: totals[accountID]}
	if lb, ok := balances[accountID]; ok {
		balance := lb.Value
		date := lb.Date
		m.latestBalance = &balance
		m.latestBalanceDate = &date
	}
	return m
}

func (h *Handler) writeAccountError(w http.ResponseWriter, err error, id int, name string) {
	switch {
	case errors.Is(err, ErrAccountNotFound):
		writeDetailError(w, http.StatusNotFound,
			fmt.Sprintf("Account with id %d not found", id))
	case errors.Is(err, ErrAccountInactive):
		writeDetailError(w, http.StatusNotFound, "Account not found")
	case errors.Is(err, ErrAccountNotLiability):
		writeDetailError(w, http.StatusBadRequest,
			fmt.Sprintf("Account '%s' is not a liability account. "+
				"Only liability accounts can have debts.", name))
	default:
		h.logger.Error("account check", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
	}
}

func (h *Handler) writeDebtError(w http.ResponseWriter, err error, id int) {
	if errors.Is(err, ErrNotFound) {
		writeDetailError(w, http.StatusNotFound,
			fmt.Sprintf("Debt with id %d not found", id))
		return
	}
	h.logger.Error("debt store", "err", err)
	writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
}

func parseIDParam(w http.ResponseWriter, r *http.Request, urlKey, errField string) (int, bool) {
	raw := chi.URLParam(r, urlKey)
	id, err := strconv.Atoi(raw)
	if err != nil {
		writeValidationError(w, errField, "must be an integer", raw)
		return 0, false
	}
	return id, true
}

// --- request parsing ---

type createRequest struct {
	Name          string
	DebtType      string
	StartDate     time.Time
	InitialAmount decimal.Decimal
	InterestRate  decimal.Decimal
	Currency      string
	Notes         *string
}

func buildCreateRequest(raw map[string]json.RawMessage) (createRequest, *validationError) {
	var r createRequest
	name, vErr := requireString(raw, "name", "Name cannot be empty")
	if vErr != nil {
		return r, vErr
	}
	r.Name = name

	dt, vErr := requireEnumString(raw, "debt_type", validDebtTypes)
	if vErr != nil {
		return r, vErr
	}
	r.DebtType = dt

	sd, vErr := requireDateNotFuture(raw, "start_date", "Start date cannot be in the future")
	if vErr != nil {
		return r, vErr
	}
	r.StartDate = sd

	ia, vErr := requirePositiveDecimal(raw, "initial_amount", "Initial amount must be greater than 0")
	if vErr != nil {
		return r, vErr
	}
	r.InitialAmount = ia

	ir, vErr := requireNonNegativeDecimal(raw, "interest_rate", "Interest rate must be greater than or equal to 0")
	if vErr != nil {
		return r, vErr
	}
	r.InterestRate = ir

	cur, vErr := optionalCurrencyPLN(raw)
	if vErr != nil {
		return r, vErr
	}
	r.Currency = cur

	if v, ok := raw["notes"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return r, &validationError{Field: "notes", Msg: "must be a string"}
		}
		r.Notes = &s
	}
	return r, nil
}

func buildUpdatePatch(raw map[string]json.RawMessage) (UpdatePatch, *validationError) {
	var p UpdatePatch
	if v, ok := raw["name"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &validationError{Field: "name", Msg: "must be a string"}
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return p, &validationError{Field: "name", Msg: "Name cannot be empty"}
		}
		p.Name = &s
	}
	if v, ok := raw["debt_type"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &validationError{Field: "debt_type", Msg: "must be a string"}
		}
		if _, ok := validDebtTypes[s]; !ok {
			return p, &validationError{Field: "debt_type", Msg: fmt.Sprintf("invalid value %q", s)}
		}
		p.DebtType = &s
	}
	if v, ok := raw["start_date"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &validationError{Field: "start_date", Msg: "must be a string"}
		}
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			return p, &validationError{Field: "start_date", Msg: "must be YYYY-MM-DD"}
		}
		if t.After(today()) {
			return p, &validationError{Field: "start_date", Msg: "Start date cannot be in the future"}
		}
		p.StartDate = &t
	}
	if v, ok := raw["initial_amount"]; ok && !isNull(v) {
		d, err := decimal.NewFromString(string(bytes.TrimSpace(v)))
		if err != nil {
			return p, &validationError{Field: "initial_amount", Msg: "must be a number"}
		}
		if !d.IsPositive() {
			return p, &validationError{Field: "initial_amount", Msg: "Initial amount must be greater than 0"}
		}
		p.InitialAmount = &d
	}
	if v, ok := raw["interest_rate"]; ok && !isNull(v) {
		d, err := decimal.NewFromString(string(bytes.TrimSpace(v)))
		if err != nil {
			return p, &validationError{Field: "interest_rate", Msg: "must be a number"}
		}
		if d.IsNegative() {
			return p, &validationError{Field: "interest_rate", Msg: "Interest rate must be greater than or equal to 0"}
		}
		p.InterestRate = &d
	}
	if v, ok := raw["currency"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &validationError{Field: "currency", Msg: "must be a string"}
		}
		if s != "PLN" {
			return p, &validationError{Field: "currency", Msg: "Currency must be 'PLN'"}
		}
		p.Currency = &s
	}
	if v, ok := raw["notes"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return p, &validationError{Field: "notes", Msg: "must be a string"}
		}
		p.NotesSet = true
		p.Notes = &s
	}
	return p, nil
}

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

func requireDateNotFuture(raw map[string]json.RawMessage, key, msg string) (time.Time, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return time.Time{}, &validationError{Field: key, Msg: "Field required"}
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return time.Time{}, &validationError{Field: key, Msg: "must be a string"}
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, &validationError{Field: key, Msg: "must be YYYY-MM-DD"}
	}
	if t.After(today()) {
		return time.Time{}, &validationError{Field: key, Msg: msg}
	}
	return t, nil
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

func requireNonNegativeDecimal(raw map[string]json.RawMessage, key, msg string) (decimal.Decimal, *validationError) {
	v, ok := raw[key]
	if !ok || isNull(v) {
		return decimal.Decimal{}, &validationError{Field: key, Msg: "Field required"}
	}
	d, err := decimal.NewFromString(string(bytes.TrimSpace(v)))
	if err != nil {
		return decimal.Decimal{}, &validationError{Field: key, Msg: "must be a number"}
	}
	if d.IsNegative() {
		return decimal.Decimal{}, &validationError{Field: key, Msg: msg}
	}
	return d, nil
}

func optionalCurrencyPLN(raw map[string]json.RawMessage) (string, *validationError) {
	v, ok := raw["currency"]
	if !ok || isNull(v) {
		return "PLN", nil
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		return "", &validationError{Field: "currency", Msg: "must be a string"}
	}
	if s != "PLN" {
		return "", &validationError{Field: "currency", Msg: "Currency must be 'PLN'"}
	}
	return s, nil
}

func today() time.Time {
	t := time.Now().UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func isNull(v json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}

// wire types
type isoDate time.Time

func (d isoDate) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(d).Format("2006-01-02") + `"`), nil
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
