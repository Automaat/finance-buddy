package debts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

type response struct {
	ID                 int           `json:"id"`
	AccountID          int           `json:"account_id"`
	AccountName        string        `json:"account_name"`
	AccountOwnerUserID *int          `json:"account_owner_user_id"`
	Name               string        `json:"name"`
	DebtType           string        `json:"debt_type"`
	StartDate          wire.IsoDate  `json:"start_date"`
	InitialAmount      wire.PyFloat  `json:"initial_amount"`
	InterestRate       wire.PyFloat  `json:"interest_rate"`
	Currency           string        `json:"currency"`
	Notes              *string       `json:"notes"`
	IsActive           bool          `json:"is_active"`
	CreatedAt          wire.IsoNaive `json:"created_at"`
	LatestBalance      *wire.PyFloat `json:"latest_balance"`
	LatestBalanceDate  *wire.IsoDate `json:"latest_balance_date"`
	TotalPaid          wire.PyFloat  `json:"total_paid"`
	InterestPaid       wire.PyFloat  `json:"interest_paid"`
}

type listResponse struct {
	Debts              []response   `json:"debts"`
	TotalCount         int          `json:"total_count"`
	TotalInitialAmount wire.PyFloat `json:"total_initial_amount"`
	ActiveDebtsCount   int          `json:"active_debts_count"`
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

// List serves GET /api/debts.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := ListFilter{}
	if v := q.Get("account_id"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			httputil.WriteBodyValidationError(w, "account_id", "must be an integer", v)
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
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	ids := make([]int, 0, len(rows))
	for i := range rows {
		ids = append(ids, rows[i].Debt.AccountID)
	}
	balances, err := h.store.LatestBalancesByAccount(r.Context(), ids)
	if err != nil {
		h.logger.Error("latest balances", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	totals, err := h.store.TotalPaidByAccount(r.Context(), ids)
	if err != nil {
		h.logger.Error("total paid", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
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
	out.TotalInitialAmount = wire.PyFloat(f64)
	httputil.WriteJSON(w, http.StatusOK, out)
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
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toResponse(dwa.Debt, dwa.Account, m))
}

// CreateWithAccount serves POST /api/debts. Creates the liability account
// and the debt in one DB transaction so a failed debt validation leaves no
// orphan account behind. Account fields beyond name/owner/currency are
// derived from debt_type (mortgage -> mortgage category, installment_0percent
// -> installment category).
func (h *Handler) CreateWithAccount(w http.ResponseWriter, r *http.Request) {
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		httputil.WriteBodyValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	req, vErr := buildCreateRequest(raw)
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	ownerID, vErr := requireOwnerUserID(raw)
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	acc := AccountSpec{
		Name:        req.Name,
		Category:    debtTypeToCategory[req.DebtType],
		OwnerUserID: ownerID,
		Currency:    req.Currency,
	}
	created, accInfo, err := h.store.CreateWithAccount(r.Context(), acc, &Debt{
		Name: req.Name, DebtType: req.DebtType, StartDate: req.StartDate,
		InitialAmount: req.InitialAmount, InterestRate: req.InterestRate,
		Currency: req.Currency, Notes: req.Notes,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrDuplicateName):
			httputil.WriteDetailError(w, http.StatusConflict,
				fmt.Sprintf("Account with name '%s' already exists", req.Name))
		case errors.Is(err, ErrDuplicateForAccount):
			httputil.WriteDetailError(w, http.StatusConflict,
				fmt.Sprintf("Debt already exists for account '%s'", req.Name))
		default:
			h.logger.Error("create debt with account", "err", err)
			httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		}
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toResponse(*created, *accInfo, debtMetrics{}))
}

// Create serves POST /api/accounts/{account_id}/debts.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	accountID, err := strconv.Atoi(chi.URLParam(r, "account_id"))
	if err != nil {
		httputil.WriteBodyValidationError(w, "account_id", "must be an integer", chi.URLParam(r, "account_id"))
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
		httputil.WriteBodyValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	req, vErr := buildCreateRequest(raw)
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
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
			httputil.WriteDetailError(w, http.StatusConflict,
				fmt.Sprintf("Debt already exists for account '%s'", account.Name))
			return
		}
		h.logger.Error("create debt", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	m, err := h.metricsForDebt(r.Context(), accountID)
	if err != nil {
		h.logger.Error("debt metrics", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toResponse(*created, *account, m))
}

// Update serves PUT /api/debts/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r, "id", "debt_id")
	if !ok {
		return
	}
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		httputil.WriteBodyValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	patch, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
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
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toResponse(dwa.Debt, dwa.Account, m))
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

func (h *Handler) writeAccountError(w http.ResponseWriter, err error, id int, name string) {
	switch {
	case errors.Is(err, ErrAccountNotFound):
		httputil.WriteDetailError(w, http.StatusNotFound,
			fmt.Sprintf("Account with id %d not found", id))
	case errors.Is(err, ErrAccountInactive):
		httputil.WriteDetailError(w, http.StatusNotFound, "Account not found")
	case errors.Is(err, ErrAccountNotLiability):
		httputil.WriteDetailError(w, http.StatusBadRequest,
			fmt.Sprintf("Account '%s' is not a liability account. "+
				"Only liability accounts can have debts.", name))
	default:
		h.logger.Error("account check", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
	}
}

func (h *Handler) writeDebtError(w http.ResponseWriter, err error, id int) {
	if errors.Is(err, ErrNotFound) {
		httputil.WriteDetailError(w, http.StatusNotFound,
			fmt.Sprintf("Debt with id %d not found", id))
		return
	}
	h.logger.Error("debt store", "err", err)
	httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
}

func parseIDParam(w http.ResponseWriter, r *http.Request, urlKey, errField string) (int, bool) {
	raw := chi.URLParam(r, urlKey)
	id, err := strconv.Atoi(raw)
	if err != nil {
		httputil.WriteBodyValidationError(w, errField, "must be an integer", raw)
		return 0, false
	}
	return id, true
}
