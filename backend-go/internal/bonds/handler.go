package bonds

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"github.com/Automaat/finance-buddy/backend-go/internal/cpi"
	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

// CPILoader is the subset of cpi.Store the handler needs. Decoupling the
// import lets unit tests stub it without spinning up a pool.
type CPILoader interface {
	LoadYoYMap(ctx context.Context) (map[int]decimal.Decimal, error)
}

// Handler is the HTTP boundary for /api/bonds.
type Handler struct {
	store  *Store
	cpi    CPILoader
	logger *slog.Logger
	now    func() time.Time
}

// NewHandler wires the store + CPI loader + logger.
func NewHandler(store *Store, cpiStore CPILoader, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, cpi: cpiStore, logger: logger, now: time.Now}
}

// response is the wire shape for a single bond.
type response struct {
	ID            int           `json:"id"`
	Type          string        `json:"type"`
	Series        string        `json:"series"`
	FaceValue     float64       `json:"face_value"`
	PurchaseDate  wire.IsoDate  `json:"purchase_date"`
	MaturityDate  wire.IsoDate  `json:"maturity_date"`
	OwnerUserID   *int          `json:"owner_user_id"`
	FirstYearRate float64       `json:"first_year_rate"`
	Margin        float64       `json:"margin"`
	Capitalize    bool          `json:"capitalize"`
	CurrentValue  float64       `json:"current_value"`
	CurrentYield  float64       `json:"current_yield"`
	CreatedAt     wire.IsoNaive `json:"created_at"`
}

type listResponse struct {
	Bonds      []response `json:"bonds"`
	TotalValue float64    `json:"total_value"`
	TotalCount int        `json:"total_count"`
}

type ytmPointResponse struct {
	Year     int          `json:"year"`
	Date     wire.IsoDate `json:"date"`
	Value    float64      `json:"value"`
	YearRate float64      `json:"year_rate"`
}

type ytmResponse struct {
	BondID int                `json:"bond_id"`
	Points []ytmPointResponse `json:"points"`
}

type createRequest struct {
	Type          string       `json:"type"`
	Series        string       `json:"series"`
	FaceValue     float64      `json:"face_value"`
	PurchaseDate  wire.IsoDate `json:"purchase_date"`
	OwnerUserID   *int         `json:"owner_user_id"`
	FirstYearRate float64      `json:"first_year_rate"`
	Margin        float64      `json:"margin"`
	Capitalize    bool         `json:"capitalize"`
}

func (h *Handler) toResponse(b *TreasuryBond, yoy map[int]decimal.Decimal) response {
	face, _ := b.FaceValue.Float64()
	firstYear, _ := b.FirstYearRate.Float64()
	margin, _ := b.Margin.Float64()
	current := CurrentValue(b, yoy, h.now())
	currentFloat, _ := current.Float64()
	yieldRate, _ := YearRate(b, yoy, currentBondYear(b, h.now())).Float64()

	return response{
		ID:            b.ID,
		Type:          string(b.Type),
		Series:        b.Series,
		FaceValue:     face,
		PurchaseDate:  wire.IsoDate(b.PurchaseDate),
		MaturityDate:  wire.IsoDate(MaturityDate(b)),
		OwnerUserID:   b.OwnerUserID,
		FirstYearRate: firstYear,
		Margin:        margin,
		Capitalize:    b.Capitalize,
		CurrentValue:  currentFloat,
		CurrentYield:  yieldRate,
		CreatedAt:     wire.IsoNaive(b.CreatedAt),
	}
}

func currentBondYear(b *TreasuryBond, now time.Time) int {
	years, _ := elapsedYearsAndFraction(b.PurchaseDate, now)
	return years + 1
}

// List serves GET /api/bonds.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	bonds, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("list bonds", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	yoy, err := h.loadYoY(r.Context())
	if err != nil {
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := listResponse{Bonds: make([]response, 0, len(bonds))}
	total := decimal.Zero
	for i := range bonds {
		out.Bonds = append(out.Bonds, h.toResponse(&bonds[i], yoy))
		total = total.Add(CurrentValue(&bonds[i], yoy, h.now()))
	}
	t, _ := total.Float64()
	out.TotalValue = t
	out.TotalCount = len(bonds)
	httputil.WriteJSON(w, http.StatusOK, out)
}

// Get serves GET /api/bonds/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	b, err := h.store.Get(r.Context(), id)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	yoy, err := h.loadYoY(r.Context())
	if err != nil {
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, h.toResponse(b, yoy))
}

// YTM serves GET /api/bonds/{id}/ytm — the yield-to-maturity projection.
func (h *Handler) YTM(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	b, err := h.store.Get(r.Context(), id)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	yoy, err := h.loadYoY(r.Context())
	if err != nil {
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	pts := YieldToMaturity(b, yoy)
	out := ytmResponse{BondID: b.ID, Points: make([]ytmPointResponse, 0, len(pts))}
	for _, p := range pts {
		v, _ := p.Value.Float64()
		rate, _ := p.YearRate.Float64()
		out.Points = append(out.Points, ytmPointResponse{
			Year:     p.Year,
			Date:     wire.IsoDate(p.Date),
			Value:    v,
			YearRate: rate,
		})
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

// Create serves POST /api/bonds.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		httputil.WriteBodyValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	if vErr := validateCreate(&req); vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	b := &TreasuryBond{
		Type:          BondType(req.Type),
		Series:        req.Series,
		FaceValue:     decimal.NewFromFloat(req.FaceValue),
		PurchaseDate:  time.Time(req.PurchaseDate),
		OwnerUserID:   req.OwnerUserID,
		FirstYearRate: decimal.NewFromFloat(req.FirstYearRate),
		Margin:        decimal.NewFromFloat(req.Margin),
		Capitalize:    req.Capitalize,
	}
	created, err := h.store.Create(r.Context(), b)
	if err != nil {
		h.logger.Error("create bond", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	yoy, err := h.loadYoY(r.Context())
	if err != nil {
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, h.toResponse(created, yoy))
}

// Update serves PUT /api/bonds/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
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
	b, err := h.store.Update(r.Context(), id, patch)
	if err != nil {
		h.writeStoreError(w, err)
		return
	}
	yoy, err := h.loadYoY(r.Context())
	if err != nil {
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, h.toResponse(b, yoy))
}

// Delete serves DELETE /api/bonds/{id}.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		h.writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httputil.WriteDetailError(w, http.StatusNotFound, "Treasury bond not found")
	default:
		h.logger.Error("bonds store", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
	}
}

func (h *Handler) loadYoY(ctx context.Context) (map[int]decimal.Decimal, error) {
	yoy, err := h.cpi.LoadYoYMap(ctx)
	if err != nil {
		h.logger.Error("load cpi", "err", err)
		return nil, err
	}
	return yoy, nil
}

func parseIDParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.Atoi(raw)
	if err != nil {
		httputil.WriteBodyValidationError(w, "bond_id", "must be an integer", raw)
		return 0, false
	}
	return id, true
}

// Sanity: keep the package's cpi import alive even if the production wiring
// uses a different interface; the cpi.Store satisfies CPILoader exactly.
var _ CPILoader = (*cpi.Store)(nil)
