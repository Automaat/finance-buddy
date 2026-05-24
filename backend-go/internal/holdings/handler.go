package holdings

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

// Handler is the HTTP boundary for /api/holdings.
type Handler struct {
	store  *Store
	logger *slog.Logger
}

// NewHandler wires store + logger.
func NewHandler(store *Store, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, logger: logger}
}

// --- security wire types ---

type securityResponse struct {
	ID        int     `json:"id"`
	Symbol    string  `json:"symbol"`
	ISIN      *string `json:"isin"`
	Name      string  `json:"name"`
	AssetType string  `json:"asset_type"`
	Currency  string  `json:"currency"`
	CreatedAt string  `json:"created_at"`
}

type listSecuritiesResponse struct {
	Securities []securityResponse `json:"securities"`
}

func toSecurity(s Security) securityResponse {
	return securityResponse{
		ID: s.ID, Symbol: s.Symbol, ISIN: s.ISIN, Name: s.Name,
		AssetType: s.AssetType, Currency: s.Currency,
		CreatedAt: s.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999"),
	}
}

// ListSecurities serves GET /api/holdings/securities.
func (h *Handler) ListSecurities(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.ListSecurities(r.Context())
	if err != nil {
		h.logger.Error("list securities", "err", err)
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]securityResponse, 0, len(rows))
	for _, s := range rows {
		out = append(out, toSecurity(s))
	}
	writeJSON(w, http.StatusOK, listSecuritiesResponse{Securities: out})
}

// CreateSecurity serves POST /api/holdings/securities.
func (h *Handler) CreateSecurity(w http.ResponseWriter, r *http.Request) {
	raw, err := readBody(r)
	if err != nil {
		writeValidation(w, "body", "Invalid JSON body")
		return
	}
	sec, vErr := buildSecurityInput(raw)
	if vErr != nil {
		writeValidation(w, vErr.Field, vErr.Msg)
		return
	}
	created, err := h.store.CreateSecurity(r.Context(), sec)
	if err != nil {
		h.logger.Error("create security", "err", err)
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toSecurity(created))
}

// DeleteSecurity serves DELETE /api/holdings/securities/{id}.
func (h *Handler) DeleteSecurity(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.store.DeleteSecurity(r.Context(), id); err != nil {
		if errors.Is(err, ErrSecurityNotFound) {
			writeError(w, http.StatusNotFound, "Security not found")
			return
		}
		// Foreign-key violation from referencing lots.
		writeError(w, http.StatusConflict, "Cannot delete security with lots; delete the lots first")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- lot wire types ---

type lotResponse struct {
	ID         int    `json:"id"`
	AccountID  int    `json:"account_id"`
	SecurityID int    `json:"security_id"`
	Side       string `json:"side"`
	Quantity   string `json:"quantity"`
	Price      string `json:"price"`
	Fee        string `json:"fee"`
	Date       string `json:"date"`
	CreatedAt  string `json:"created_at"`
}

type listLotsResponse struct {
	Lots []lotResponse `json:"lots"`
}

func toLot(l Lot) lotResponse {
	return lotResponse{
		ID: l.ID, AccountID: l.AccountID, SecurityID: l.SecurityID,
		Side:      string(l.Side),
		Quantity:  l.Quantity.String(),
		Price:     l.Price.String(),
		Fee:       l.Fee.StringFixed(2),
		Date:      l.Date.UTC().Format("2006-01-02"),
		CreatedAt: l.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999"),
	}
}

// ListLots serves GET /api/holdings/lots?account_id=&security_id=.
func (h *Handler) ListLots(w http.ResponseWriter, r *http.Request) {
	var accountID, securityID *int
	if v := r.URL.Query().Get("account_id"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			writeValidation(w, "account_id", "must be a positive integer")
			return
		}
		accountID = &n
	}
	if v := r.URL.Query().Get("security_id"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			writeValidation(w, "security_id", "must be a positive integer")
			return
		}
		securityID = &n
	}
	rows, err := h.store.ListLots(r.Context(), accountID, securityID)
	if err != nil {
		h.logger.Error("list lots", "err", err)
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]lotResponse, 0, len(rows))
	for i := range rows {
		out = append(out, toLot(rows[i]))
	}
	writeJSON(w, http.StatusOK, listLotsResponse{Lots: out})
}

// CreateLot serves POST /api/holdings/lots.
func (h *Handler) CreateLot(w http.ResponseWriter, r *http.Request) {
	raw, err := readBody(r)
	if err != nil {
		writeValidation(w, "body", "Invalid JSON body")
		return
	}
	lot, vErr := buildLotInput(raw)
	if vErr != nil {
		writeValidation(w, vErr.Field, vErr.Msg)
		return
	}
	created, err := h.store.CreateLot(r.Context(), lot)
	if err != nil {
		h.logger.Error("create lot", "err", err)
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusCreated, toLot(created))
}

// DeleteLot serves DELETE /api/holdings/lots/{id}.
func (h *Handler) DeleteLot(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.store.DeleteLot(r.Context(), id); err != nil {
		if errors.Is(err, ErrLotNotFound) {
			writeError(w, http.StatusNotFound, "Lot not found")
			return
		}
		h.logger.Error("delete lot", "err", err)
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- quote wire types ---

type quoteResponse struct {
	ID         int    `json:"id"`
	SecurityID int    `json:"security_id"`
	Date       string `json:"date"`
	Price      string `json:"price"`
	Source     string `json:"source"`
	CreatedAt  string `json:"created_at"`
}

type listQuotesResponse struct {
	Quotes []quoteResponse `json:"quotes"`
}

func toQuote(q PriceQuote) quoteResponse {
	return quoteResponse{
		ID: q.ID, SecurityID: q.SecurityID,
		Date:      q.Date.UTC().Format("2006-01-02"),
		Price:     q.Price.String(),
		Source:    q.Source,
		CreatedAt: q.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999"),
	}
}

// ListQuotes serves GET /api/holdings/securities/{id}/quotes.
func (h *Handler) ListQuotes(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	rows, err := h.store.ListQuotes(r.Context(), id)
	if err != nil {
		h.logger.Error("list quotes", "err", err)
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]quoteResponse, 0, len(rows))
	for _, q := range rows {
		out = append(out, toQuote(q))
	}
	writeJSON(w, http.StatusOK, listQuotesResponse{Quotes: out})
}

// UpsertQuote serves POST /api/holdings/securities/{id}/quotes.
func (h *Handler) UpsertQuote(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	raw, err := readBody(r)
	if err != nil {
		writeValidation(w, "body", "Invalid JSON body")
		return
	}
	q, vErr := buildQuoteInput(raw, id)
	if vErr != nil {
		writeValidation(w, vErr.Field, vErr.Msg)
		return
	}
	saved, err := h.store.UpsertQuote(r.Context(), q)
	if err != nil {
		h.logger.Error("upsert quote", "err", err)
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	writeJSON(w, http.StatusOK, toQuote(saved))
}

// --- holdings aggregate ---

type holdingRowResponse struct {
	Security        securityResponse `json:"security"`
	Quantity        string           `json:"quantity"`
	AverageCost     string           `json:"average_cost"`
	CostBasis       string           `json:"cost_basis"`
	LatestQuote     *string          `json:"latest_quote"`
	LatestQuoteDate *string          `json:"latest_quote_date"`
	MarketValue     string           `json:"market_value"`
	UnrealizedGain  string           `json:"unrealized_gain"`
	RealizedGain    string           `json:"realized_gain"`
}

type holdingsResponse struct {
	Holdings []holdingRowResponse `json:"holdings"`
}

// Holdings serves GET /api/holdings.
func (h *Handler) Holdings(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.Holdings(r.Context())
	if err != nil {
		h.logger.Error("holdings", "err", err)
		writeError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]holdingRowResponse, 0, len(rows))
	for i := range rows {
		hr := &rows[i]
		row := holdingRowResponse{
			Security:       toSecurity(hr.Security),
			Quantity:       hr.Running.Quantity.String(),
			AverageCost:    hr.Running.AverageCost.String(),
			CostBasis:      hr.Running.CostBasis.StringFixed(2),
			MarketValue:    hr.MarketValue.StringFixed(2),
			UnrealizedGain: hr.UnrealizedGain.StringFixed(2),
			RealizedGain:   hr.Running.RealizedGain.StringFixed(2),
		}
		if !hr.LatestQuote.IsZero() {
			q := hr.LatestQuote.String()
			row.LatestQuote = &q
		}
		if hr.LatestQuoteDate != nil {
			d := hr.LatestQuoteDate.UTC().Format("2006-01-02")
			row.LatestQuoteDate = &d
		}
		out = append(out, row)
	}
	writeJSON(w, http.StatusOK, holdingsResponse{Holdings: out})
}

// --- validation ---

type validationError struct {
	Field string
	Msg   string
}

func buildSecurityInput(raw map[string]json.RawMessage) (Security, *validationError) {
	var s Security
	s.Currency = "PLN"
	if v, ok := raw["symbol"]; ok && string(v) != "null" {
		if err := json.Unmarshal(v, &s.Symbol); err != nil {
			return s, &validationError{Field: "symbol", Msg: "must be a string"}
		}
	}
	if s.Symbol = strings.TrimSpace(s.Symbol); s.Symbol == "" {
		return s, &validationError{Field: "symbol", Msg: "required"}
	}
	if len(s.Symbol) > 32 {
		return s, &validationError{Field: "symbol", Msg: "max 32 chars"}
	}
	if v, ok := raw["isin"]; ok && string(v) != "null" {
		var isin string
		if err := json.Unmarshal(v, &isin); err != nil {
			return s, &validationError{Field: "isin", Msg: "must be a string"}
		}
		if isin = strings.TrimSpace(isin); isin != "" {
			if len(isin) != 12 {
				return s, &validationError{Field: "isin", Msg: "must be 12 chars"}
			}
			s.ISIN = &isin
		}
	}
	if v, ok := raw["name"]; ok && string(v) != "null" {
		if err := json.Unmarshal(v, &s.Name); err != nil {
			return s, &validationError{Field: "name", Msg: "must be a string"}
		}
	}
	if s.Name = strings.TrimSpace(s.Name); s.Name == "" {
		return s, &validationError{Field: "name", Msg: "required"}
	}
	if len(s.Name) > 200 {
		return s, &validationError{Field: "name", Msg: "max 200 chars"}
	}
	if v, ok := raw["asset_type"]; ok && string(v) != "null" {
		if err := json.Unmarshal(v, &s.AssetType); err != nil {
			return s, &validationError{Field: "asset_type", Msg: "must be a string"}
		}
	}
	if !validAssetType(s.AssetType) {
		return s, &validationError{Field: "asset_type", Msg: "must be one of stock|etf|bond|fund"}
	}
	if v, ok := raw["currency"]; ok && string(v) != "null" {
		var c string
		if err := json.Unmarshal(v, &c); err != nil {
			return s, &validationError{Field: "currency", Msg: "must be a string"}
		}
		if c = strings.TrimSpace(c); c != "" {
			if len(c) != 3 {
				return s, &validationError{Field: "currency", Msg: "must be 3 chars"}
			}
			s.Currency = strings.ToUpper(c)
		}
	}
	return s, nil
}

func validAssetType(t string) bool {
	switch t {
	case "stock", "etf", "bond", "fund":
		return true
	}
	return false
}

func buildLotInput(raw map[string]json.RawMessage) (Lot, *validationError) {
	var l Lot
	if vErr := requireInt(raw, "account_id", &l.AccountID); vErr != nil {
		return l, vErr
	}
	if l.AccountID <= 0 {
		return l, &validationError{Field: "account_id", Msg: "must be positive"}
	}
	if vErr := requireInt(raw, "security_id", &l.SecurityID); vErr != nil {
		return l, vErr
	}
	if l.SecurityID <= 0 {
		return l, &validationError{Field: "security_id", Msg: "must be positive"}
	}
	var sideStr string
	if vErr := requireString(raw, "side", &sideStr); vErr != nil {
		return l, vErr
	}
	if !IsValidSide(sideStr) {
		return l, &validationError{Field: "side", Msg: "must be buy or sell"}
	}
	l.Side = Side(sideStr)
	qty, vErr := requireDecimal(raw, "quantity")
	if vErr != nil {
		return l, vErr
	}
	if qty.Sign() <= 0 {
		return l, &validationError{Field: "quantity", Msg: "must be positive"}
	}
	l.Quantity = qty
	price, vErr := requireDecimal(raw, "price")
	if vErr != nil {
		return l, vErr
	}
	if price.Sign() < 0 {
		return l, &validationError{Field: "price", Msg: "must not be negative"}
	}
	l.Price = price
	if v, ok := raw["fee"]; ok && string(v) != "null" {
		fee, ferr := decimal.NewFromString(strings.Trim(string(v), `"`))
		if ferr != nil {
			return l, &validationError{Field: "fee", Msg: "must be a number"}
		}
		if fee.Sign() < 0 {
			return l, &validationError{Field: "fee", Msg: "must not be negative"}
		}
		l.Fee = fee
	}
	dateStr, vErr := requireString2(raw, "date")
	if vErr != nil {
		return l, vErr
	}
	parsed, derr := time.Parse("2006-01-02", dateStr)
	if derr != nil {
		return l, &validationError{Field: "date", Msg: "must be YYYY-MM-DD"}
	}
	l.Date = parsed
	return l, nil
}

func buildQuoteInput(raw map[string]json.RawMessage, securityID int) (PriceQuote, *validationError) {
	q := PriceQuote{SecurityID: securityID, Source: "manual"}
	dateStr, vErr := requireString2(raw, "date")
	if vErr != nil {
		return q, vErr
	}
	parsed, derr := time.Parse("2006-01-02", dateStr)
	if derr != nil {
		return q, &validationError{Field: "date", Msg: "must be YYYY-MM-DD"}
	}
	q.Date = parsed
	price, vErr := requireDecimal(raw, "price")
	if vErr != nil {
		return q, vErr
	}
	if price.Sign() < 0 {
		return q, &validationError{Field: "price", Msg: "must not be negative"}
	}
	q.Price = price
	if v, ok := raw["source"]; ok && string(v) != "null" {
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			return q, &validationError{Field: "source", Msg: "must be a string"}
		}
		if s = strings.TrimSpace(s); s != "" {
			if len(s) > 40 {
				return q, &validationError{Field: "source", Msg: "max 40 chars"}
			}
			q.Source = s
		}
	}
	return q, nil
}

// --- shared helpers ---

func parseID(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		writeValidation(w, "id", "must be a positive integer")
		return 0, false
	}
	return id, true
}

func readBody(r *http.Request) (map[string]json.RawMessage, error) {
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func requireInt(raw map[string]json.RawMessage, key string, dest *int) *validationError {
	v, ok := raw[key]
	if !ok || string(v) == "null" {
		return &validationError{Field: key, Msg: "required"}
	}
	if err := json.Unmarshal(v, dest); err != nil {
		return &validationError{Field: key, Msg: "must be integer"}
	}
	return nil
}

func requireString(raw map[string]json.RawMessage, key string, dest *string) *validationError {
	v, ok := raw[key]
	if !ok || string(v) == "null" {
		return &validationError{Field: key, Msg: "required"}
	}
	if err := json.Unmarshal(v, dest); err != nil {
		return &validationError{Field: key, Msg: "must be a string"}
	}
	if *dest == "" {
		return &validationError{Field: key, Msg: "cannot be empty"}
	}
	return nil
}

func requireString2(raw map[string]json.RawMessage, key string) (string, *validationError) {
	var s string
	if vErr := requireString(raw, key, &s); vErr != nil {
		return "", vErr
	}
	return s, nil
}

func requireDecimal(raw map[string]json.RawMessage, key string) (decimal.Decimal, *validationError) {
	v, ok := raw[key]
	if !ok || string(v) == "null" {
		return decimal.Zero, &validationError{Field: key, Msg: "required"}
	}
	d, err := decimal.NewFromString(strings.Trim(string(v), `"`))
	if err != nil {
		return decimal.Zero, &validationError{Field: key, Msg: "must be a number"}
	}
	return d, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Default().Error("encode response", "err", err, "status", status)
	}
}

func writeError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]string{"detail": detail})
}

func writeValidation(w http.ResponseWriter, field, msg string) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
		"detail": []map[string]any{
			{"type": "value_error", "loc": []string{"body", field}, "msg": msg},
		},
	})
}
