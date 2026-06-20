package transactions

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/validation"
	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

// The canonical set of transaction_type values lives in types.go.
// IsValid + ValidTypes are the only authority.

type response struct {
	ID              int           `json:"id"`
	AccountID       int           `json:"account_id"`
	AccountName     string        `json:"account_name"`
	Amount          wire.PyFloat  `json:"amount"`
	Date            wire.IsoDate  `json:"date"`
	OwnerUserID     *int          `json:"owner_user_id"`
	TransactionType *string       `json:"transaction_type"`
	CreatedAt       wire.IsoNaive `json:"created_at"`
}

type listResponse struct {
	Transactions     []response   `json:"transactions"`
	TotalInvested    wire.PyFloat `json:"total_invested"`
	TransactionCount int          `json:"transaction_count"`
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

func toResponse(t Transaction, accountName string) response {
	amt, _ := t.Amount.Float64()
	return response{
		ID:              t.ID,
		AccountID:       t.AccountID,
		AccountName:     accountName,
		Amount:          wire.PyFloat(amt),
		Date:            wire.IsoDate(t.Date),
		OwnerUserID:     t.OwnerUserID,
		TransactionType: t.TransactionType,
		CreatedAt:       wire.IsoNaive(t.CreatedAt.UTC()),
	}
}

// ListForAccount serves GET /api/accounts/{id}/transactions.
func (h *Handler) ListForAccount(w http.ResponseWriter, r *http.Request) {
	accountID, ok := parseAccountID(w, r)
	if !ok {
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
	rows, err := h.store.ListForAccount(r.Context(), accountID)
	if err != nil {
		h.logger.Error("list account transactions", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	h.writeList(w, rows, func(t Transaction) string { return account.Name })
}

// ListAll serves GET /api/transactions.
func (h *Handler) ListAll(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := ListFilter{}
	accountID, ok := httputil.OptionalQueryInt(w, q, "account_id")
	if !ok {
		return
	}
	f.AccountID = accountID
	ownerUserID, ok := httputil.OptionalQueryInt(w, q, "owner_user_id")
	if !ok {
		return
	}
	f.OwnerUserID = ownerUserID
	for _, p := range []struct {
		key  string
		dest **time.Time
	}{
		{"date_from", &f.DateFrom},
		{"date_to", &f.DateTo},
	} {
		t, ok := httputil.OptionalQueryDate(w, q, p.key)
		if !ok {
			return
		}
		*p.dest = t
	}
	rows, err := h.store.ListAll(r.Context(), f)
	if err != nil {
		h.logger.Error("list all transactions", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := listResponse{Transactions: make([]response, 0, len(rows))}
	totalCents := decimal.Zero
	for i := range rows {
		tr := &rows[i]
		out.Transactions = append(out.Transactions, toResponse(tr.Transaction, tr.AccountName))
		if tr.Transaction.TransactionType != nil && *tr.Transaction.TransactionType == "withdrawal" {
			totalCents = totalCents.Sub(tr.Transaction.Amount)
		} else {
			totalCents = totalCents.Add(tr.Transaction.Amount)
		}
	}
	out.TransactionCount = len(out.Transactions)
	total, _ := totalCents.Float64()
	out.TotalInvested = wire.PyFloat(total)
	httputil.WriteJSON(w, http.StatusOK, out)
}

// Create serves POST /api/accounts/{id}/transactions.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	accountID, ok := parseAccountID(w, r)
	if !ok {
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
	if !httputil.DecodeJSON(w, r, 1<<16, &raw) {
		return
	}
	req, vErr := buildCreateRequest(raw)
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	created, err := h.store.Create(r.Context(), &Transaction{
		AccountID:       accountID,
		Amount:          req.Amount,
		Date:            req.Date,
		OwnerUserID:     req.OwnerUserID,
		TransactionType: req.TransactionType,
	})
	if err != nil {
		h.logger.Error("create transaction", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toResponse(*created, account.Name))
}

// Delete serves DELETE /api/accounts/{id}/transactions/{transaction_id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	accountID, ok := parseAccountID(w, r)
	if !ok {
		return
	}
	transactionID, ok := httputil.PathInt(w, r, "transaction_id")
	if !ok {
		return
	}
	if err := h.store.SoftDelete(r.Context(), accountID, transactionID); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			httputil.WriteDetailError(w, http.StatusNotFound,
				fmt.Sprintf("Transaction with id %d not found", transactionID))
		case errors.Is(err, ErrCrossAccount):
			httputil.WriteDetailError(w, http.StatusForbidden,
				fmt.Sprintf("Transaction %d does not belong to account %d", transactionID, accountID))
		default:
			h.logger.Error("delete transaction", "err", err)
			httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Types serves GET /api/transactions/types — the canonical enum the
// frontend dropdown should render. Ordered the same way as ValidTypes()
// so the UI is deterministic.
func (h *Handler) Types(w http.ResponseWriter, _ *http.Request) {
	types := ValidTypes()
	out := make([]typesItem, 0, len(types))
	for _, t := range types {
		out = append(out, typesItem{Value: string(t), Label: LabelPL(t)})
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

// Counts serves GET /api/transactions/counts.
func (h *Handler) Counts(w http.ResponseWriter, r *http.Request) {
	counts, err := h.store.CountsByAccount(r.Context())
	if err != nil {
		h.logger.Error("transaction counts", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	// Pydantic dict[int, int] serializes int keys as JSON strings.
	out := make(map[string]int, len(counts))
	for k, v := range counts {
		out[strconv.Itoa(k)] = v
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) writeList(w http.ResponseWriter, rows []Transaction, name func(Transaction) string) {
	out := listResponse{Transactions: make([]response, 0, len(rows))}
	total := decimal.Zero
	for _, t := range rows {
		out.Transactions = append(out.Transactions, toResponse(t, name(t)))
		if t.TransactionType != nil && *t.TransactionType == "withdrawal" {
			total = total.Sub(t.Amount)
		} else {
			total = total.Add(t.Amount)
		}
	}
	out.TransactionCount = len(out.Transactions)
	f, _ := total.Float64()
	out.TotalInvested = wire.PyFloat(f)
	httputil.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) writeAccountError(w http.ResponseWriter, err error, id int, name string) {
	switch {
	case errors.Is(err, ErrAccountNotFound):
		httputil.WriteDetailError(w, http.StatusNotFound,
			fmt.Sprintf("Account with id %d not found", id))
	case errors.Is(err, ErrAccountNotInvestment):
		httputil.WriteDetailError(w, http.StatusBadRequest,
			fmt.Sprintf("Account '%s' cannot have transactions. "+
				"Only investment or retirement accounts (IKE/IKZE/PPK) can track transactions.", name))
	default:
		h.logger.Error("account check", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
	}
}

func parseAccountID(w http.ResponseWriter, r *http.Request) (int, bool) {
	return httputil.PathInt(w, r, "account_id")
}

// --- request parsing ---

type createRequest struct {
	Amount          decimal.Decimal
	Date            time.Time
	OwnerUserID     *int
	TransactionType *string
}

func buildCreateRequest(raw map[string]json.RawMessage) (createRequest, *httputil.ValidationError) {
	var r createRequest
	amount, vErr := validation.RequiredPositiveDecimal(raw, "amount", "Field required", "Amount must be greater than 0")
	if vErr != nil {
		return r, vErr
	}
	r.Amount = amount

	t, vErr := validation.RequiredDateNotFuture(raw, "date", time.Now)
	if vErr != nil {
		return r, vErr
	}
	r.Date = t

	ownerID, vErr := validation.RequiredIntOrNull(raw, "owner_user_id")
	if vErr != nil {
		return r, vErr
	}
	r.OwnerUserID = ownerID

	if v, ok := raw["transaction_type"]; ok && !validation.IsNull(v) {
		s, err := validation.RawString(v)
		if err != nil {
			return r, &httputil.ValidationError{Field: "transaction_type", Msg: "must be a string"}
		}
		if !IsValid(s) {
			return r, &httputil.ValidationError{Field: "transaction_type", Msg: fmt.Sprintf("invalid value %q", s)}
		}
		r.TransactionType = &s
	}
	return r, nil
}
