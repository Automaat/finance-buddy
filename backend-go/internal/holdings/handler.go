package holdings

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
)

// pgErrorCode returns the SQLSTATE for a Postgres-driven error, or "" when
// the error didn't originate from pgx.
func pgErrorCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

// Handler is the HTTP boundary for /api/holdings.
type Handler struct {
	store  *Store
	rates  RateProvider
	logger *slog.Logger
}

// NewHandler wires store + rates + logger. rates may be nil — Holdings then
// returns native-currency fields only and skips PLN conversion.
func NewHandler(store *Store, rates RateProvider, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, rates: rates, logger: logger}
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
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]securityResponse, 0, len(rows))
	for _, s := range rows {
		out = append(out, toSecurity(s))
	}
	httputil.WriteJSON(w, http.StatusOK, listSecuritiesResponse{Securities: out})
}

// CreateSecurity serves POST /api/holdings/securities.
func (h *Handler) CreateSecurity(w http.ResponseWriter, r *http.Request) {
	raw, err := readBody(r)
	if err != nil {
		httputil.WriteBodyValidationError(w, "body", "Invalid JSON body", "")
		return
	}
	sec, vErr := buildSecurityInput(raw)
	if vErr != nil {
		httputil.WriteBodyValidationError(w, vErr.Field, vErr.Msg, "")
		return
	}
	created, err := h.store.CreateSecurity(r.Context(), sec)
	if err != nil {
		if pgErrorCode(err) == pgUniqueViolation {
			httputil.WriteDetailError(w, http.StatusConflict, "A security with the same symbol or ISIN already exists")
			return
		}
		h.logger.Error("create security", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toSecurity(created))
}

// DeleteSecurity serves DELETE /api/holdings/securities/{id}.
func (h *Handler) DeleteSecurity(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.store.DeleteSecurity(r.Context(), id); err != nil {
		if errors.Is(err, ErrSecurityNotFound) {
			httputil.WriteDetailError(w, http.StatusNotFound, "Security not found")
			return
		}
		if pgErrorCode(err) == pgForeignKeyViolation {
			httputil.WriteDetailError(w, http.StatusConflict, "Cannot delete security with lots; delete the lots first")
			return
		}
		h.logger.Error("delete security", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
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
			httputil.WriteBodyValidationError(w, "account_id", "must be a positive integer", "")
			return
		}
		accountID = &n
	}
	if v := r.URL.Query().Get("security_id"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			httputil.WriteBodyValidationError(w, "security_id", "must be a positive integer", "")
			return
		}
		securityID = &n
	}
	rows, err := h.store.ListLots(r.Context(), accountID, securityID)
	if err != nil {
		h.logger.Error("list lots", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]lotResponse, 0, len(rows))
	for i := range rows {
		out = append(out, toLot(rows[i]))
	}
	httputil.WriteJSON(w, http.StatusOK, listLotsResponse{Lots: out})
}

// CreateLot serves POST /api/holdings/lots.
func (h *Handler) CreateLot(w http.ResponseWriter, r *http.Request) {
	raw, err := readBody(r)
	if err != nil {
		httputil.WriteBodyValidationError(w, "body", "Invalid JSON body", "")
		return
	}
	lot, vErr := buildLotInput(raw)
	if vErr != nil {
		httputil.WriteBodyValidationError(w, vErr.Field, vErr.Msg, "")
		return
	}
	created, err := h.store.CreateLot(r.Context(), lot)
	if err != nil {
		if pgErrorCode(err) == pgForeignKeyViolation {
			httputil.WriteDetailError(w, http.StatusNotFound, "Referenced account or security not found")
			return
		}
		h.logger.Error("create lot", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toLot(created))
}

// DeleteLot serves DELETE /api/holdings/lots/{id}.
func (h *Handler) DeleteLot(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.store.DeleteLot(r.Context(), id); err != nil {
		if errors.Is(err, ErrLotNotFound) {
			httputil.WriteDetailError(w, http.StatusNotFound, "Lot not found")
			return
		}
		h.logger.Error("delete lot", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
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
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]quoteResponse, 0, len(rows))
	for _, q := range rows {
		out = append(out, toQuote(q))
	}
	httputil.WriteJSON(w, http.StatusOK, listQuotesResponse{Quotes: out})
}

// UpsertQuote serves POST /api/holdings/securities/{id}/quotes.
func (h *Handler) UpsertQuote(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	raw, err := readBody(r)
	if err != nil {
		httputil.WriteBodyValidationError(w, "body", "Invalid JSON body", "")
		return
	}
	q, vErr := buildQuoteInput(raw, id)
	if vErr != nil {
		httputil.WriteBodyValidationError(w, vErr.Field, vErr.Msg, "")
		return
	}
	saved, err := h.store.UpsertQuote(r.Context(), q)
	if err != nil {
		if pgErrorCode(err) == pgForeignKeyViolation {
			httputil.WriteDetailError(w, http.StatusNotFound, "Security not found")
			return
		}
		h.logger.Error("upsert quote", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toQuote(saved))
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

	// PLN-converted fields. Null when no FX rate was available for any lot
	// date or the latest quote date; the UI then renders native currency only.
	AverageCostPLN     *string `json:"average_cost_pln"`
	CostBasisPLN       *string `json:"cost_basis_pln"`
	MarketValuePLN     *string `json:"market_value_pln"`
	UnrealizedGainPLN  *string `json:"unrealized_gain_pln"`
	RealizedGainPLN    *string `json:"realized_gain_pln"`
	LatestQuoteRatePLN *string `json:"latest_quote_rate_pln"`

	// Accounts is the per-account breakdown of this security position.
	// Always present (possibly empty if quantities aggregate to zero per
	// account). Snapshot pre-fill consumes these per-account totals.
	Accounts []accountPositionResponse `json:"accounts"`
}

type accountPositionResponse struct {
	AccountID         int     `json:"account_id"`
	AccountName       string  `json:"account_name"`
	OwnerUserID       int     `json:"owner_user_id"`
	Quantity          string  `json:"quantity"`
	AverageCost       string  `json:"average_cost"`
	CostBasis         string  `json:"cost_basis"`
	MarketValue       string  `json:"market_value"`
	UnrealizedGain    string  `json:"unrealized_gain"`
	RealizedGain      string  `json:"realized_gain"`
	AverageCostPLN    *string `json:"average_cost_pln"`
	CostBasisPLN      *string `json:"cost_basis_pln"`
	MarketValuePLN    *string `json:"market_value_pln"`
	UnrealizedGainPLN *string `json:"unrealized_gain_pln"`
	RealizedGainPLN   *string `json:"realized_gain_pln"`
}

type holdingsResponse struct {
	Holdings []holdingRowResponse `json:"holdings"`
}

// Holdings serves GET /api/holdings.
func (h *Handler) Holdings(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.Holdings(r.Context(), h.rates)
	if err != nil {
		h.logger.Error("holdings", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
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
		if hr.Running.HasPLN {
			avg := hr.Running.AverageCostPLN.StringFixed(4)
			cb := hr.Running.CostBasisPLN.StringFixed(2)
			rg := hr.Running.RealizedGainPLN.StringFixed(2)
			row.AverageCostPLN = &avg
			row.CostBasisPLN = &cb
			row.RealizedGainPLN = &rg
		}
		if hr.HasPLN {
			mv := hr.MarketValuePLN.StringFixed(2)
			ug := hr.UnrealizedGainPLN.StringFixed(2)
			rate := hr.LatestQuoteRatePLN.StringFixed(4)
			row.MarketValuePLN = &mv
			row.UnrealizedGainPLN = &ug
			row.LatestQuoteRatePLN = &rate
		}
		row.Accounts = make([]accountPositionResponse, 0, len(hr.Accounts))
		for j := range hr.Accounts {
			row.Accounts = append(row.Accounts, toAccountPosition(&hr.Accounts[j]))
		}
		out = append(out, row)
	}
	httputil.WriteJSON(w, http.StatusOK, holdingsResponse{Holdings: out})
}

func toAccountPosition(p *AccountPosition) accountPositionResponse {
	resp := accountPositionResponse{
		AccountID:      p.AccountID,
		AccountName:    p.AccountName,
		OwnerUserID:    p.OwnerUserID,
		Quantity:       p.Running.Quantity.String(),
		AverageCost:    p.Running.AverageCost.String(),
		CostBasis:      p.Running.CostBasis.StringFixed(2),
		MarketValue:    p.MarketValue.StringFixed(2),
		UnrealizedGain: p.UnrealizedGain.StringFixed(2),
		RealizedGain:   p.Running.RealizedGain.StringFixed(2),
	}
	if p.Running.HasPLN {
		avg := p.Running.AverageCostPLN.StringFixed(4)
		cb := p.Running.CostBasisPLN.StringFixed(2)
		rg := p.Running.RealizedGainPLN.StringFixed(2)
		resp.AverageCostPLN = &avg
		resp.CostBasisPLN = &cb
		resp.RealizedGainPLN = &rg
	}
	if p.HasPLN {
		mv := p.MarketValuePLN.StringFixed(2)
		ug := p.UnrealizedGainPLN.StringFixed(2)
		resp.MarketValuePLN = &mv
		resp.UnrealizedGainPLN = &ug
	}
	return resp
}

// --- dividend wire types ---

type dividendResponse struct {
	ID             int    `json:"id"`
	AccountID      int    `json:"account_id"`
	SecurityID     int    `json:"security_id"`
	PayDate        string `json:"pay_date"`
	GrossAmount    string `json:"gross_amount"`
	WithholdingTax string `json:"withholding_tax"`
	NetAmount      string `json:"net_amount"`
	Currency       string `json:"currency"`
	CreatedAt      string `json:"created_at"`
}

type listDividendsResponse struct {
	Dividends []dividendResponse `json:"dividends"`
}

// dividendCreateRequest documents the POST /api/holdings/dividends body for
// OpenAPI generation. Decimal money fields ride as JSON strings to preserve
// precision; withholding_tax and currency are optional (default 0 / "PLN").
type dividendCreateRequest struct {
	AccountID      int    `json:"account_id"`
	SecurityID     int    `json:"security_id"`
	PayDate        string `json:"pay_date"`
	GrossAmount    string `json:"gross_amount"`
	WithholdingTax string `json:"withholding_tax"`
	Currency       string `json:"currency"`
}

func toDividend(d Dividend) dividendResponse {
	return dividendResponse{
		ID: d.ID, AccountID: d.AccountID, SecurityID: d.SecurityID,
		PayDate:        d.PayDate.UTC().Format("2006-01-02"),
		GrossAmount:    d.GrossAmount.StringFixed(2),
		WithholdingTax: d.WithholdingTax.StringFixed(2),
		NetAmount:      d.Net().StringFixed(2),
		Currency:       d.Currency,
		CreatedAt:      d.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999"),
	}
}

// ListDividends serves GET /api/holdings/dividends?account_id=&security_id=.
func (h *Handler) ListDividends(w http.ResponseWriter, r *http.Request) {
	var accountID, securityID *int
	if v := r.URL.Query().Get("account_id"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			httputil.WriteBodyValidationError(w, "account_id", "must be a positive integer", "")
			return
		}
		accountID = &n
	}
	if v := r.URL.Query().Get("security_id"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			httputil.WriteBodyValidationError(w, "security_id", "must be a positive integer", "")
			return
		}
		securityID = &n
	}
	rows, err := h.store.ListDividends(r.Context(), accountID, securityID)
	if err != nil {
		h.logger.Error("list dividends", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]dividendResponse, 0, len(rows))
	for _, d := range rows {
		out = append(out, toDividend(d))
	}
	httputil.WriteJSON(w, http.StatusOK, listDividendsResponse{Dividends: out})
}

// CreateDividend serves POST /api/holdings/dividends.
func (h *Handler) CreateDividend(w http.ResponseWriter, r *http.Request) {
	raw, err := readBody(r)
	if err != nil {
		httputil.WriteBodyValidationError(w, "body", "Invalid JSON body", "")
		return
	}
	div, vErr := buildDividendInput(raw)
	if vErr != nil {
		httputil.WriteBodyValidationError(w, vErr.Field, vErr.Msg, "")
		return
	}
	created, err := h.store.CreateDividend(r.Context(), div)
	if err != nil {
		if pgErrorCode(err) == pgForeignKeyViolation {
			httputil.WriteDetailError(w, http.StatusNotFound, "Referenced account or security not found")
			return
		}
		h.logger.Error("create dividend", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toDividend(created))
}

// DeleteDividend serves DELETE /api/holdings/dividends/{id}.
func (h *Handler) DeleteDividend(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.store.DeleteDividend(r.Context(), id); err != nil {
		if errors.Is(err, ErrDividendNotFound) {
			httputil.WriteDetailError(w, http.StatusNotFound, "Dividend not found")
			return
		}
		h.logger.Error("delete dividend", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- shared helpers ---

func parseID(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		httputil.WriteBodyValidationError(w, "id", "must be a positive integer", "")
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
