package debtpayments

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

type response struct {
	ID          int           `json:"id"`
	AccountID   int           `json:"account_id"`
	AccountName string        `json:"account_name"`
	Amount      wire.PyFloat  `json:"amount"`
	Date        wire.IsoDate  `json:"date"`
	OwnerUserID *int          `json:"owner_user_id"`
	CreatedAt   wire.IsoNaive `json:"created_at"`
}

type listResponse struct {
	Payments     []response   `json:"payments"`
	TotalPaid    wire.PyFloat `json:"total_paid"`
	PaymentCount int          `json:"payment_count"`
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

func toResponse(p DebtPayment, name string) response {
	amt, _ := p.Amount.Float64()
	return response{
		ID: p.ID, AccountID: p.AccountID, AccountName: name,
		Amount: wire.PyFloat(amt), Date: wire.IsoDate(p.Date), OwnerUserID: p.OwnerUserID,
		CreatedAt: wire.IsoNaive(p.CreatedAt.UTC()),
	}
}

// ListForAccount serves GET /api/accounts/{id}/payments.
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
		h.logger.Error("list account payments", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := listResponse{Payments: make([]response, 0, len(rows))}
	total := decimal.Zero
	for _, p := range rows {
		out.Payments = append(out.Payments, toResponse(p, account.Name))
		total = total.Add(p.Amount)
	}
	out.PaymentCount = len(out.Payments)
	f, _ := total.Float64()
	out.TotalPaid = wire.PyFloat(f)
	writeJSON(w, http.StatusOK, out)
}

// ListAll serves GET /api/payments.
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
		h.logger.Error("list all payments", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := listResponse{Payments: make([]response, 0, len(rows))}
	total := decimal.Zero
	for _, p := range rows {
		out.Payments = append(out.Payments, toResponse(p.Payment, p.AccountName))
		total = total.Add(p.Payment.Amount)
	}
	out.PaymentCount = len(out.Payments)
	f64, _ := total.Float64()
	out.TotalPaid = wire.PyFloat(f64)
	writeJSON(w, http.StatusOK, out)
}

// Create serves POST /api/accounts/{id}/payments.
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
	created, err := h.store.Create(r.Context(), &DebtPayment{
		AccountID: accountID, Amount: req.Amount, Date: req.Date, OwnerUserID: req.OwnerUserID,
	})
	if err != nil {
		if errors.Is(err, ErrDuplicate) {
			writeDetailError(w, http.StatusConflict,
				fmt.Sprintf("Payment for account %d on %s already exists",
					accountID, req.Date.Format("2006-01-02")))
			return
		}
		h.logger.Error("create payment", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toResponse(*created, account.Name))
}

// Delete serves DELETE /api/accounts/{id}/payments/{payment_id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	accountID, ok := parseAccountID(w, r)
	if !ok {
		return
	}
	paymentID, err := strconv.Atoi(chi.URLParam(r, "payment_id"))
	if err != nil {
		writeValidationError(w, "payment_id", "must be an integer", chi.URLParam(r, "payment_id"))
		return
	}
	if err := h.store.SoftDelete(r.Context(), accountID, paymentID); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			writeDetailError(w, http.StatusNotFound,
				fmt.Sprintf("Payment with id %d not found", paymentID))
		case errors.Is(err, ErrCrossAccount):
			writeDetailError(w, http.StatusForbidden,
				fmt.Sprintf("Payment %d does not belong to account %d", paymentID, accountID))
		default:
			h.logger.Error("delete payment", "err", err)
			writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Counts serves GET /api/payments/counts.
func (h *Handler) Counts(w http.ResponseWriter, r *http.Request) {
	counts, err := h.store.CountsByAccount(r.Context())
	if err != nil {
		h.logger.Error("payment counts", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make(map[string]int, len(counts))
	for k, v := range counts {
		out[strconv.Itoa(k)] = v
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handler) writeAccountError(w http.ResponseWriter, err error, id int, name string) {
	switch {
	case errors.Is(err, ErrAccountNotFound):
		writeDetailError(w, http.StatusNotFound,
			fmt.Sprintf("Account with id %d not found", id))
	case errors.Is(err, ErrAccountNotLiability):
		writeDetailError(w, http.StatusBadRequest,
			fmt.Sprintf("Account '%s' is not a liability account. "+
				"Only liability accounts can have debt payments.", name))
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

type createRequest struct {
	Amount      decimal.Decimal
	Date        time.Time
	OwnerUserID *int
}

func buildCreateRequest(raw map[string]json.RawMessage) (createRequest, *validationError) {
	var r createRequest
	amount, vErr := requireAmount(raw)
	if vErr != nil {
		return r, vErr
	}
	r.Amount = amount

	t, vErr := requireDateNotFuture(raw)
	if vErr != nil {
		return r, vErr
	}
	r.Date = t

	ownerID, vErr := requireOwnerUserID(raw)
	if vErr != nil {
		return r, vErr
	}
	r.OwnerUserID = ownerID
	return r, nil
}

// requireOwnerUserID reads the owner_user_id key: present and integer, or
// explicit null (the "Shared" owner).
func requireOwnerUserID(raw map[string]json.RawMessage) (*int, *validationError) {
	const key = "owner_user_id"
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

func requireDateNotFuture(raw map[string]json.RawMessage) (time.Time, *validationError) {
	const key = "date"
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

func requireAmount(raw map[string]json.RawMessage) (decimal.Decimal, *validationError) {
	const (
		key = "amount"
		msg = "Amount must be greater than 0"
	)
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
