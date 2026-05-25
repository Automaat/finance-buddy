package transactions

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

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
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	h.writeList(w, rows, func(t Transaction) string { return account.Name })
}

// ListAll serves GET /api/transactions.
func (h *Handler) ListAll(w http.ResponseWriter, r *http.Request) {
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
	if v := q.Get("owner_user_id"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			writeValidationError(w, "owner_user_id", "must be an integer", v)
			return
		}
		f.OwnerUserID = &n
	}
	for _, p := range []struct {
		key  string
		dest **time.Time
	}{
		{"date_from", &f.DateFrom},
		{"date_to", &f.DateTo},
	} {
		if v := q.Get(p.key); v != "" {
			t, err := time.Parse("2006-01-02", v)
			if err != nil {
				writeValidationError(w, p.key, "must be YYYY-MM-DD", v)
				return
			}
			*p.dest = &t
		}
	}
	rows, err := h.store.ListAll(r.Context(), f)
	if err != nil {
		h.logger.Error("list all transactions", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
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
	writeJSON(w, http.StatusOK, out)
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
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	req, vErr := buildCreateRequest(raw)
	if vErr != nil {
		writePydanticError(w, vErr)
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
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(*created, account.Name))
}

// Delete serves DELETE /api/accounts/{id}/transactions/{transaction_id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	accountID, ok := parseAccountID(w, r)
	if !ok {
		return
	}
	transactionID, err := strconv.Atoi(chi.URLParam(r, "transaction_id"))
	if err != nil {
		writeValidationError(w, "transaction_id", "must be an integer", chi.URLParam(r, "transaction_id"))
		return
	}
	if err := h.store.SoftDelete(r.Context(), accountID, transactionID); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			writeDetailError(w, http.StatusNotFound,
				fmt.Sprintf("Transaction with id %d not found", transactionID))
		case errors.Is(err, ErrCrossAccount):
			writeDetailError(w, http.StatusForbidden,
				fmt.Sprintf("Transaction %d does not belong to account %d", transactionID, accountID))
		default:
			h.logger.Error("delete transaction", "err", err)
			writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
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
	writeJSON(w, http.StatusOK, out)
}

// Counts serves GET /api/transactions/counts.
func (h *Handler) Counts(w http.ResponseWriter, r *http.Request) {
	counts, err := h.store.CountsByAccount(r.Context())
	if err != nil {
		h.logger.Error("transaction counts", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	// Pydantic dict[int, int] serializes int keys as JSON strings.
	out := make(map[string]int, len(counts))
	for k, v := range counts {
		out[strconv.Itoa(k)] = v
	}
	writeJSON(w, http.StatusOK, out)
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
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) writeAccountError(w http.ResponseWriter, err error, id int, name string) {
	switch {
	case errors.Is(err, ErrAccountNotFound):
		writeDetailError(w, http.StatusNotFound,
			fmt.Sprintf("Account with id %d not found", id))
	case errors.Is(err, ErrAccountNotInvestment):
		writeDetailError(w, http.StatusBadRequest,
			fmt.Sprintf("Account '%s' cannot have transactions. "+
				"Only investment or retirement accounts (IKE/IKZE/PPK) can track transactions.", name))
	default:
		h.logger.Error("account check", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
	}
}

func parseAccountID(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "account_id")
	id, err := strconv.Atoi(raw)
	if err != nil {
		writeValidationError(w, "account_id", "must be an integer", raw)
		return 0, false
	}
	return id, true
}

// --- request parsing ---

type createRequest struct {
	Amount          decimal.Decimal
	Date            time.Time
	OwnerUserID     *int
	TransactionType *string
}

func buildCreateRequest(raw map[string]json.RawMessage) (createRequest, *validationError) {
	var r createRequest
	amount, vErr := requirePositiveDecimal(raw, "amount", "Amount must be greater than 0")
	if vErr != nil {
		return r, vErr
	}
	r.Amount = amount

	t, vErr := requireDateNotFuture(raw, "date")
	if vErr != nil {
		return r, vErr
	}
	r.Date = t

	ownerID, vErr := requireIntOrNull(raw, "owner_user_id")
	if vErr != nil {
		return r, vErr
	}
	r.OwnerUserID = ownerID

	if v, ok := raw["transaction_type"]; ok && !isNull(v) {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return r, &validationError{Field: "transaction_type", Msg: "must be a string"}
		}
		if !IsValid(s) {
			return r, &validationError{Field: "transaction_type", Msg: fmt.Sprintf("invalid value %q", s)}
		}
		r.TransactionType = &s
	}
	return r, nil
}

// requireIntOrNull reads an integer key that must be present; an explicit
// null is allowed and yields nil (the "Shared" owner).
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

func requireDateNotFuture(raw map[string]json.RawMessage, key string) (time.Time, *validationError) {
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
	today := time.Now().UTC()
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	if t.After(today) {
		return time.Time{}, &validationError{Field: key, Msg: "Date cannot be in the future"}
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

func isNull(v json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(v), []byte("null"))
}
