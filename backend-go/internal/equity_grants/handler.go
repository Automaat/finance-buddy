package equitygrants

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	companyvaluations "github.com/Automaat/finance-buddy/backend-go/internal/company_valuations"
	"github.com/Automaat/finance-buddy/backend-go/internal/fx"
	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

var (
	validCurrencies = map[string]struct{}{
		"PLN": {}, "USD": {}, "EUR": {}, "GBP": {}, "CHF": {},
	}
	validGrantTypes = map[string]struct{}{
		"option": {}, "rsu": {},
	}
	validFrequencies = map[string]struct{}{
		"monthly": {}, "quarterly": {}, "yearly": {},
	}
	validTaxTreatments = map[string]struct{}{
		"capital_gains_19": {}, "employment_income": {},
	}
)

// response mirrors backend/app/schemas/equity_grants.EquityGrantResponse.
type response struct {
	ID                     int                   `json:"id"`
	GrantDate              wire.IsoDate          `json:"grant_date"`
	Type                   string                `json:"type"`
	Company                string                `json:"company"`
	OwnerUserID            *int                  `json:"owner_user_id"`
	TotalShares            int                   `json:"total_shares"`
	StrikePrice            *wire.PyFloat         `json:"strike_price"`
	Currency               string                `json:"currency"`
	VestStartDate          wire.IsoDate          `json:"vest_start_date"`
	VestCliffMonths        int                   `json:"vest_cliff_months"`
	VestTotalMonths        int                   `json:"vest_total_months"`
	VestFrequency          string                `json:"vest_frequency"`
	VestCustomSchedule     []CustomScheduleEntry `json:"vest_custom_schedule"`
	RequiresLiquidityEvent bool                  `json:"requires_liquidity_event"`
	LiquidityEventDate     *wire.IsoDate         `json:"liquidity_event_date"`
	TaxTreatment           string                `json:"tax_treatment"`
	Notes                  *string               `json:"notes"`
	IsActive               bool                  `json:"is_active"`
	CreatedAt              wire.IsoNaive         `json:"created_at"`

	// Computed
	VestedSharesToday  int           `json:"vested_shares_today"`
	VestingProgressPct wire.PyFloat  `json:"vesting_progress_pct"`
	PaperValueBase     *wire.PyFloat `json:"paper_value_base"`
	PaperValueLow      *wire.PyFloat `json:"paper_value_low"`
	PaperValueHigh     *wire.PyFloat `json:"paper_value_high"`
	PaperValueCurrency *string       `json:"paper_value_currency"`
	PaperValueBasePLN  *wire.PyFloat `json:"paper_value_base_pln"`
	PaperValueLowPLN   *wire.PyFloat `json:"paper_value_low_pln"`
	PaperValueHighPLN  *wire.PyFloat `json:"paper_value_high_pln"`
	FXRate             *wire.PyFloat `json:"fx_rate"`
	ValuationDate      *wire.IsoDate `json:"valuation_date"`
	ValuationSource    *string       `json:"valuation_source"`
}

type listResponse struct {
	EquityGrants       []response `json:"equity_grants"`
	TotalCount         int        `json:"total_count"`
	AvailableCompanies []string   `json:"available_companies"`
}

// Handler is the HTTP boundary for /api/equity-grants.
type Handler struct {
	store      *Store
	valuations *companyvaluations.Store
	fx         *fx.Service
	logger     *slog.Logger
	now        func() time.Time
}

// NewHandler wires the equity-grant store + valuations + FX dependencies.
func NewHandler(
	store *Store,
	valuations *companyvaluations.Store,
	fxSvc *fx.Service,
	logger *slog.Logger,
) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{
		store:      store,
		valuations: valuations,
		fx:         fxSvc,
		logger:     logger,
		now:        time.Now,
	}
}

func (h *Handler) toResponse(ctx context.Context, g *EquityGrant) (response, error) {
	today := h.now().UTC().Truncate(24 * time.Hour)
	sched := Schedule{
		TotalShares:            g.TotalShares,
		VestStartDate:          g.VestStartDate,
		VestCliffMonths:        g.VestCliffMonths,
		VestTotalMonths:        g.VestTotalMonths,
		VestFrequencyMonths:    FreqMonthsFromString(g.VestFrequency),
		CustomSchedule:         g.VestCustomSchedule,
		RequiresLiquidityEvent: g.RequiresLiquidityEvent,
		LiquidityEventDate:     g.LiquidityEventDate,
	}
	vested := VestedSharesAt(sched, today)
	// Python uses round(x, 2) which is banker's rounding (ties-to-even).
	// math.RoundToEven matches that semantics; math.Round would diverge on
	// .005 ties.
	progress := math.RoundToEven(VestingProgressPct(sched, today)*100) / 100

	paper, err := computePaperValues(ctx, h.valuations, g, vested)
	if err != nil {
		return response{}, err
	}

	rate, err := fxRateFor(ctx, h.fx, paper.Currency, paper.ValuationDate)
	if err != nil {
		return response{}, err
	}

	out := response{
		ID:                     g.ID,
		GrantDate:              wire.IsoDate(g.GrantDate),
		Type:                   g.Type,
		Company:                g.Company,
		OwnerUserID:            g.OwnerUserID,
		TotalShares:            g.TotalShares,
		Currency:               g.Currency,
		VestStartDate:          wire.IsoDate(g.VestStartDate),
		VestCliffMonths:        g.VestCliffMonths,
		VestTotalMonths:        g.VestTotalMonths,
		VestFrequency:          g.VestFrequency,
		VestCustomSchedule:     g.VestCustomSchedule,
		RequiresLiquidityEvent: g.RequiresLiquidityEvent,
		TaxTreatment:           g.TaxTreatment,
		Notes:                  g.Notes,
		IsActive:               g.IsActive,
		CreatedAt:              wire.IsoNaive(g.CreatedAt.UTC()),
		VestedSharesToday:      vested,
		VestingProgressPct:     wire.PyFloat(progress),
	}
	if g.StrikePrice != nil {
		f, _ := g.StrikePrice.Float64()
		pf := wire.PyFloat(f)
		out.StrikePrice = &pf
	}
	if g.LiquidityEventDate != nil {
		d := wire.IsoDate(*g.LiquidityEventDate)
		out.LiquidityEventDate = &d
	}
	attachPaperValues(&out, paper)
	attachFX(&out, rate, paper.Currency, paper)
	return out, nil
}

func attachPaperValues(out *response, paper paperValueResult) {
	if paper.Base != nil {
		f, _ := paper.Base.Float64()
		pf := wire.PyFloat(f)
		out.PaperValueBase = &pf
	}
	if paper.Low != nil {
		f, _ := paper.Low.Float64()
		pf := wire.PyFloat(f)
		out.PaperValueLow = &pf
	}
	if paper.High != nil {
		f, _ := paper.High.Float64()
		pf := wire.PyFloat(f)
		out.PaperValueHigh = &pf
	}
	if paper.Currency != "" {
		c := paper.Currency
		out.PaperValueCurrency = &c
	}
	if paper.ValuationDate != nil {
		d := wire.IsoDate(*paper.ValuationDate)
		out.ValuationDate = &d
	}
	if paper.ValuationSource != "" {
		s := paper.ValuationSource
		out.ValuationSource = &s
	}
}

func attachFX(out *response, rate fx.Result, currency string, paper paperValueResult) {
	if currency == "" {
		return
	}
	if rate.Found {
		f, _ := rate.Rate.Float64()
		pf := wire.PyFloat(f)
		out.FXRate = &pf
	}
	out.PaperValueBasePLN = pln(paper.Base, currency, rate)
	out.PaperValueLowPLN = pln(paper.Low, currency, rate)
	out.PaperValueHighPLN = pln(paper.High, currency, rate)
}

func pln(amount *decimal.Decimal, currency string, rate fx.Result) *wire.PyFloat {
	v, ok := fx.ToPLN(amount, currency, rate)
	if !ok {
		return nil
	}
	f, _ := v.Float64()
	pf := wire.PyFloat(f)
	return &pf
}

// --- HTTP handlers ---

// List serves GET /api/equity-grants.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter := ListFilter{}
	if v := strings.TrimSpace(r.URL.Query().Get("owner_user_id")); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			httputil.WriteBodyValidationError(w, "owner_user_id", "must be an integer", v)
			return
		}
		filter.OwnerUserID = &n
	}
	if v := strings.TrimSpace(r.URL.Query().Get("company")); v != "" {
		filter.Company = &v
	}
	rows, companies, err := h.store.List(r.Context(), filter)
	if err != nil {
		h.logger.Error("list equity grants", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := listResponse{
		EquityGrants:       make([]response, 0, len(rows)),
		TotalCount:         len(rows),
		AvailableCompanies: companies,
	}
	for i := range rows {
		resp, err := h.toResponse(r.Context(), &rows[i])
		if err != nil {
			h.logger.Error("equity grant response", "err", err)
			httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		out.EquityGrants = append(out.EquityGrants, resp)
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

// Get serves GET /api/equity-grants/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	g, err := h.store.Get(r.Context(), id)
	if err != nil {
		h.writeStoreError(w, err, id)
		return
	}
	h.writeGrant(w, r, http.StatusOK, g)
}

// Create serves POST /api/equity-grants.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	raw := map[string]json.RawMessage{}
	if !httputil.DecodeJSON(w, r, 1<<16, &raw) {
		return
	}
	req, vErr := buildCreateRequest(raw)
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	g := requestToGrant(req)
	created, err := h.store.Create(r.Context(), g)
	if err != nil {
		h.logger.Error("create equity grant", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	h.writeGrant(w, r, http.StatusCreated, created)
}

// Update serves PATCH /api/equity-grants/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	raw := map[string]json.RawMessage{}
	if !httputil.DecodeJSON(w, r, 1<<16, &raw) {
		return
	}
	patch, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	updated, err := h.store.Update(r.Context(), id, patch)
	if err != nil {
		h.writeStoreError(w, err, id)
		return
	}
	h.writeGrant(w, r, http.StatusOK, updated)
}

// Delete serves DELETE /api/equity-grants/{id}.
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

func (h *Handler) writeGrant(w http.ResponseWriter, r *http.Request, status int, g *EquityGrant) {
	resp, err := h.toResponse(r.Context(), g)
	if err != nil {
		h.logger.Error("equity grant response", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, status, resp)
}

func (h *Handler) writeStoreError(w http.ResponseWriter, err error, id int) {
	if errors.Is(err, ErrNotFound) {
		httputil.WriteDetailError(w, http.StatusNotFound,
			fmt.Sprintf("Equity grant with id %d not found", id))
		return
	}
	var inv *InvariantError
	if errors.As(err, &inv) {
		httputil.WriteBodyValidationError(w, inv.Field, inv.Msg, "")
		return
	}
	h.logger.Error("equity grant store", "err", err)
	httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
}

func parseIDParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	return httputil.PathIntField(w, r, "id", "grant_id")
}
