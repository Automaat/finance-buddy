package accounts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/rules"
	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

// CPILoader is the subset of cpi.Store this handler needs. Decoupling lets
// unit tests stub the latest-known-year + YoY map without a pool.
type CPILoader interface {
	LatestKnownYear(ctx context.Context) (int, bool, error)
	LoadYoYMap(ctx context.Context) (map[int]decimal.Decimal, error)
}

var (
	validAccountTypes = map[string]struct{}{"asset": {}, "liability": {}}
	validCategories   = map[string]struct{}{
		"bank": {}, "saving_account": {}, "stock": {}, "bond": {}, "gold": {},
		"real_estate": {}, "ppk": {}, "fund": {}, "etf": {}, "vehicle": {},
		"mortgage": {}, "installment": {},
	}
	validPurposes = map[string]struct{}{
		"retirement": {}, "emergency_fund": {}, "general": {},
	}
	validWrappers = map[string]struct{}{
		"IKE": {}, "IKZE": {}, "PPK": {},
	}
)

type response struct {
	ID                    int           `json:"id"`
	Name                  string        `json:"name"`
	Type                  string        `json:"type"`
	Category              string        `json:"category"`
	OwnerUserID           *int          `json:"owner_user_id"`
	Currency              string        `json:"currency"`
	AccountWrapper        *string       `json:"account_wrapper"`
	Purpose               string        `json:"purpose"`
	SquareMeters          *wire.PyFloat `json:"square_meters"`
	IsActive              bool          `json:"is_active"`
	ReceivesContributions bool          `json:"receives_contributions"`
	ExcludedFromFire      bool          `json:"excluded_from_fire"`
	CreatedAt             wire.IsoNaive `json:"created_at"`
	CurrentValue          wire.PyFloat  `json:"current_value"`
	InterestRatePct       *wire.PyFloat `json:"interest_rate_pct"`
	RealYieldPct          *wire.PyFloat `json:"real_yield_pct"`
	CPIYoYPct             *wire.PyFloat `json:"cpi_yoy_pct"`
	CPIAsOfYear           *int          `json:"cpi_as_of_year"`
}

type listResponse struct {
	Assets      []response `json:"assets"`
	Liabilities []response `json:"liabilities"`
}

// Handler is the HTTP boundary for /api/accounts.
type Handler struct {
	store    *Store
	cpi      CPILoader
	holdings HoldingsValuator
	logger   *slog.Logger
}

// HoldingsValuator returns the live PLN market value per account derived
// from the holdings ledger. Snapshot/current_value pre-fill consumes this
// for stock/etf/bond/fund accounts so user-typed snapshots already match
// what brokers show. Implemented by holdings.Valuator.
type HoldingsValuator interface {
	AccountValuesPLN(ctx context.Context) (map[int]decimal.Decimal, error)
}

// investmentCategories names the account categories whose current_value
// gets overridden by the live holdings sum. Bank/cash categories keep the
// snapshot-derived value (they don't have lots).
var investmentCategories = map[string]bool{
	"stock": true,
	"etf":   true,
	"bond":  true,
	"fund":  true,
}

// NewHandler wires the store + CPI loader + holdings valuator + logger.
// cpiStore may be nil in tests (real-yield columns become null). holdings
// may be nil — then current_value falls back to the snapshot-derived value
// for every account.
func NewHandler(store *Store, cpiStore CPILoader, holdings HoldingsValuator, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, cpi: cpiStore, holdings: holdings, logger: logger}
}

// realYieldCtx carries the per-request CPI snapshot used to compute real
// yield. Zero value means CPI is unavailable; toResponse handles it by
// returning null for the yield columns.
type realYieldCtx struct {
	yoyPct    decimal.Decimal // latest YoY in percent (e.g. 4.0 for +4% YoY)
	asOfYear  int
	hasLatest bool
}

// belkaRatePct is the Polish capital-gains tax (Belka) applied to interest
// earned outside an IKE/IKZE wrapper, expressed as a percentage (e.g.
// 19.0). Sourced from the centralized rules table (issue #545) as a
// decimal — going through float64 here would inject rounding drift into a
// financial calculation, so we read the rule's Value directly.
var belkaRatePct = mustBelkaRatePct()

func mustBelkaRatePct() decimal.Decimal {
	r, ok := rules.Get("capital_gains_tax_2026")
	if !ok {
		panic("accounts: rules table missing capital_gains_tax_2026")
	}
	return r.Value.Mul(decimal.NewFromInt(100))
}

// isShieldedFromBelka reports whether interest in this wrapper is exempt
// from the 19% Belka withholding. IKE income is tax-free at withdrawal; IKZE
// is taxed at a flat 10% on withdrawal but exempt from Belka year-on-year,
// so we treat both as shielded for the real-yield indicator.
func isShieldedFromBelka(wrapper *string) bool {
	if wrapper == nil {
		return false
	}
	return *wrapper == "IKE" || *wrapper == "IKZE"
}

// computeRealYieldPct returns the after-tax-after-inflation yield for a
// nominal rate, given the current CPI YoY (as percent) and whether the
// account is held in an IKE/IKZE wrapper.
//
// Formula matches the issue example: net = nominal * (1 - belka) when not
// shielded, then real = net - cpi. The approximation matches what a Polish
// saver would compute on the back of an envelope; the small (real ≈ net -
// cpi vs. Fisher's (1+net)/(1+cpi)-1) error is in the basis-point range
// for the sub-10% rates this widget targets.
func computeRealYieldPct(nominalPct, cpiYoYPct decimal.Decimal, shielded bool) decimal.Decimal {
	net := nominalPct
	if !shielded {
		net = nominalPct.Mul(decimal.NewFromInt(100).Sub(belkaRatePct)).Div(decimal.NewFromInt(100))
	}
	return net.Sub(cpiYoYPct)
}

func toResponse(a *Account, currentValue decimal.Decimal, ry realYieldCtx) response {
	cv, _ := currentValue.Float64()
	out := response{
		ID:                    a.ID,
		Name:                  a.Name,
		Type:                  a.Type,
		Category:              a.Category,
		OwnerUserID:           a.OwnerUserID,
		Currency:              a.Currency,
		AccountWrapper:        a.AccountWrapper,
		Purpose:               a.Purpose,
		IsActive:              a.IsActive,
		ReceivesContributions: a.ReceivesContributions,
		ExcludedFromFire:      a.ExcludedFromFire,
		CreatedAt:             wire.IsoNaive(a.CreatedAt.UTC()),
		CurrentValue:          wire.PyFloat(cv),
	}
	if a.SquareMeters != nil {
		f, _ := a.SquareMeters.Float64()
		pf := wire.PyFloat(f)
		out.SquareMeters = &pf
	}
	if a.InterestRatePct != nil {
		nominalF, _ := a.InterestRatePct.Float64()
		nominal := wire.PyFloat(nominalF)
		out.InterestRatePct = &nominal
		if ry.hasLatest {
			realDec := computeRealYieldPct(*a.InterestRatePct, ry.yoyPct, isShieldedFromBelka(a.AccountWrapper))
			realF, _ := realDec.Float64()
			realPF := wire.PyFloat(realF)
			out.RealYieldPct = &realPF
			yoyF, _ := ry.yoyPct.Float64()
			yoyPF := wire.PyFloat(yoyF)
			out.CPIYoYPct = &yoyPF
			year := ry.asOfYear
			out.CPIAsOfYear = &year
		}
	}
	return out
}

// loadHoldingsValues fetches live PLN market value per account from the
// holdings ledger. Returns nil on failure (logged) so the request still
// succeeds with snapshot-derived values. Empty when no valuator wired.
func (h *Handler) loadHoldingsValues(ctx context.Context) map[int]decimal.Decimal {
	if h.holdings == nil {
		return nil
	}
	vals, err := h.holdings.AccountValuesPLN(ctx)
	if err != nil {
		h.logger.Warn("accounts: holdings valuation failed", "err", err)
		return nil
	}
	return vals
}

// realYieldCtxFor is the single-account variant of loadRealYieldCtx: when
// the account has no nominal rate, the real-yield columns are null anyway,
// so we skip the CPI round-trip entirely. Used by Create/Update where the
// response carries one row and the typical case (account without a rate)
// shouldn't pay for two extra DB reads.
func (h *Handler) realYieldCtxFor(ctx context.Context, a *Account) realYieldCtx {
	if a == nil || a.InterestRatePct == nil {
		return realYieldCtx{}
	}
	return h.loadRealYieldCtx(ctx)
}

// loadRealYieldCtx pulls the latest known CPI YoY from the index table and
// converts it from the GUS form (e.g. 104.0 = +4% YoY) to percent points.
// Returns a zero-value context when CPI is unavailable, in which case
// toResponse skips the real-yield columns.
func (h *Handler) loadRealYieldCtx(ctx context.Context) realYieldCtx {
	if h.cpi == nil {
		return realYieldCtx{}
	}
	latestYear, ok, err := h.cpi.LatestKnownYear(ctx)
	if err != nil {
		h.logger.Error("real yield: latest cpi year", "err", err)
		return realYieldCtx{}
	}
	if !ok {
		return realYieldCtx{}
	}
	yoyMap, err := h.cpi.LoadYoYMap(ctx)
	if err != nil {
		h.logger.Error("real yield: load cpi", "err", err)
		return realYieldCtx{}
	}
	raw, ok := yoyMap[latestYear]
	if !ok {
		return realYieldCtx{}
	}
	return realYieldCtx{
		yoyPct:    raw.Sub(decimal.NewFromInt(100)),
		asOfYear:  latestYear,
		hasLatest: true,
	}
}

// List serves GET /api/accounts.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("list accounts", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	ids := make([]int, 0, len(accounts))
	for i := range accounts {
		ids = append(ids, accounts[i].ID)
	}
	values, err := h.store.LatestValuesByAccount(r.Context(), ids)
	if err != nil {
		h.logger.Error("latest values", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	holdingsValues := h.loadHoldingsValues(r.Context())
	ry := h.loadRealYieldCtx(r.Context())
	out := listResponse{Assets: []response{}, Liabilities: []response{}}
	for i := range accounts {
		a := &accounts[i]
		cv := values[a.ID]
		if investmentCategories[a.Category] {
			if live, ok := holdingsValues[a.ID]; ok {
				cv = live
			}
		}
		resp := toResponse(a, cv, ry)
		if a.Type == "asset" {
			out.Assets = append(out.Assets, resp)
		} else {
			out.Liabilities = append(out.Liabilities, resp)
		}
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

// Create serves POST /api/accounts.
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
	created, err := h.store.Create(r.Context(), &Account{
		Name:                  req.Name,
		Type:                  req.Type,
		Category:              req.Category,
		OwnerUserID:           req.OwnerUserID,
		Currency:              req.Currency,
		AccountWrapper:        req.AccountWrapper,
		Purpose:               req.Purpose,
		SquareMeters:          req.SquareMeters,
		ReceivesContributions: req.ReceivesContributions,
		ExcludedFromFire:      req.ExcludedFromFire,
		InterestRatePct:       req.InterestRatePct,
	})
	if err != nil {
		if errors.Is(err, ErrDuplicateName) {
			httputil.WriteDetailError(w, http.StatusConflict,
				fmt.Sprintf("Account with name '%s' already exists", req.Name))
			return
		}
		h.logger.Error("create account", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, toResponse(created, decimal.Zero, h.realYieldCtxFor(r.Context(), created)))
}

// Update serves PUT /api/accounts/{id}.
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
		h.writeStoreError(w, err, id, patch.Name)
		return
	}
	cv, err := h.store.LatestValueForAccount(r.Context(), id)
	if err != nil {
		h.logger.Error("latest value", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, toResponse(updated, cv, h.realYieldCtxFor(r.Context(), updated)))
}

// Delete serves DELETE /api/accounts/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		h.writeStoreError(w, err, id, nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeStoreError(w http.ResponseWriter, err error, id int, name *string) {
	switch {
	case errors.Is(err, ErrNotFound):
		httputil.WriteDetailError(w, http.StatusNotFound,
			fmt.Sprintf("Account with id %d not found", id))
	case errors.Is(err, ErrDuplicateName):
		n := ""
		if name != nil {
			n = *name
		}
		httputil.WriteDetailError(w, http.StatusConflict,
			fmt.Sprintf("Account with name '%s' already exists", n))
	default:
		h.logger.Error("account store", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
	}
}

func parseIDParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	return httputil.PathIntField(w, r, "id", "account_id")
}
