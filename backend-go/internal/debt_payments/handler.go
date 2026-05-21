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
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

type response struct {
	ID          int      `json:"id"`
	AccountID   int      `json:"account_id"`
	AccountName string   `json:"account_name"`
	Amount      pyFloat  `json:"amount"`
	Date        isoDate  `json:"date"`
	Owner       string   `json:"owner"`
	CreatedAt   isoNaive `json:"created_at"`
}

type listResponse struct {
	Payments     []response `json:"payments"`
	TotalPaid    pyFloat    `json:"total_paid"`
	PaymentCount int        `json:"payment_count"`
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
		Amount: pyFloat(amt), Date: isoDate(p.Date), Owner: p.Owner,
		CreatedAt: isoNaive(p.CreatedAt.UTC()),
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
	out.TotalPaid = pyFloat(f)
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
	if v := q.Get("owner"); v != "" {
		s := v
		f.Owner = &s
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
	out.TotalPaid = pyFloat(f64)
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
		AccountID: accountID, Amount: req.Amount, Date: req.Date, Owner: req.Owner,
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
	Amount decimal.Decimal
	Date   time.Time
	Owner  string
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

	owner, vErr := requireString(raw, "owner", "Owner cannot be empty")
	if vErr != nil {
		return r, vErr
	}
	r.Owner = owner
	return r, nil
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
