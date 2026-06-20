package retirement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Automaat/finance-buddy/backend-go/internal/httputil"
	"github.com/Automaat/finance-buddy/backend-go/internal/wire"
)

type yearlyStat struct {
	Year                int           `json:"year"`
	AccountWrapper      string        `json:"account_wrapper"`
	OwnerUserID         *int          `json:"owner_user_id"`
	LimitAmount         *wire.PyFloat `json:"limit_amount"`
	TotalContributed    wire.PyFloat  `json:"total_contributed"`
	EmployeeContributed wire.PyFloat  `json:"employee_contributed"`
	EmployerContributed wire.PyFloat  `json:"employer_contributed"`
	Remaining           *wire.PyFloat `json:"remaining"`
	PercentageUsed      *wire.PyFloat `json:"percentage_used"`
	IsWarning           bool          `json:"is_warning"`
	// IKZE-only: marginal PIT rate at the owner's annualized salary and
	// the estimated PIT savings from this year's contributions. Nil when
	// the wrapper is not IKZE or the owner has no salary record.
	MarginalTaxRate *wire.PyFloat `json:"marginal_tax_rate"`
	PITSavings      *wire.PyFloat `json:"pit_savings"`
}

type ppkStat struct {
	OwnerUserID           *int         `json:"owner_user_id"`
	TotalValue            wire.PyFloat `json:"total_value"`
	EmployeeContributed   wire.PyFloat `json:"employee_contributed"`
	EmployerContributed   wire.PyFloat `json:"employer_contributed"`
	GovernmentContributed wire.PyFloat `json:"government_contributed"`
	TotalContributed      wire.PyFloat `json:"total_contributed"`
	Returns               wire.PyFloat `json:"returns"`
	ROIPercentage         wire.PyFloat `json:"roi_percentage"`
}

type limitResponse struct {
	ID             int          `json:"id"`
	Year           int          `json:"year"`
	AccountWrapper string       `json:"account_wrapper"`
	OwnerUserID    *int         `json:"owner_user_id"`
	LimitAmount    wire.PyFloat `json:"limit_amount"`
	Notes          *string      `json:"notes"`
}

type ppkGenerateResponse struct {
	OwnerUserID         *int         `json:"owner_user_id"`
	Month               int          `json:"month"`
	Year                int          `json:"year"`
	GrossSalary         wire.PyFloat `json:"gross_salary"`
	EmployeeAmount      wire.PyFloat `json:"employee_amount"`
	EmployerAmount      wire.PyFloat `json:"employer_amount"`
	GovernmentAmount    wire.PyFloat `json:"government_amount"`
	WelcomeApplied      bool         `json:"welcome_applied"`
	AnnualApplied       bool         `json:"annual_applied"`
	TotalAmount         wire.PyFloat `json:"total_amount"`
	TransactionsCreated []int        `json:"transactions_created"`
}

// Handler is the HTTP boundary for /api/retirement.
type Handler struct {
	store  *Store
	logger *slog.Logger
	now    func() time.Time
}

// NewHandler wires the store + logger.
func NewHandler(store *Store, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, logger: logger, now: time.Now}
}

// Stats serves GET /api/retirement/stats.
func (h *Handler) Stats(w http.ResponseWriter, r *http.Request) {
	year := h.now().UTC().Year()
	if v := r.URL.Query().Get("year"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			httputil.WriteBodyValidationError(w, "year", "must be an integer", v)
			return
		}
		year = n
	}
	owners, err := h.resolveOwners(r.Context(), r.URL.Query().Get("owner_user_id"))
	if err != nil {
		if errors.Is(err, errBadOwnerParam) {
			httputil.WriteBodyValidationError(w, "owner_user_id", "must be an integer",
				r.URL.Query().Get("owner_user_id"))
			return
		}
		h.logger.Error("resolve owners", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := []yearlyStat{}
	for _, wrapper := range []string{"IKE", "IKZE"} {
		for _, owner := range owners {
			stat, included, err := h.buildYearlyStat(r.Context(), year, wrapper, owner)
			if err != nil {
				h.logger.Error("yearly stat", "err", err)
				httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
				return
			}
			if included {
				out = append(out, stat)
			}
		}
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) buildYearlyStat(ctx context.Context, year int, wrapper string, ownerUserID *int) (yearlyStat, bool, error) {
	accountIDs, err := h.store.AccountIDsForWrapper(ctx, wrapper, ownerUserID)
	if err != nil {
		return yearlyStat{}, false, err
	}
	if len(accountIDs) == 0 {
		return yearlyStat{}, false, nil
	}
	totals, err := h.store.YearlyContributions(ctx, year, accountIDs)
	if err != nil {
		return yearlyStat{}, false, err
	}
	limit, configured, err := h.store.LimitConfigured(ctx, year, wrapper, ownerUserID)
	if err != nil {
		return yearlyStat{}, false, err
	}
	if !configured {
		return yearlyStat{}, false, nil
	}
	stat := computeYearlyStat(year, wrapper, ownerUserID, totals, limit.LimitAmount)
	totalF, _ := totals.Total.Float64()
	if wrapper == "IKZE" && totalF > 0 {
		if rate, savings, ok := h.estimateIKZEPITSavings(ctx, ownerUserID, year, totalF); ok {
			r := wire.PyFloat(rate)
			s := wire.PyFloat(savings)
			stat.MarginalTaxRate = &r
			stat.PITSavings = &s
		}
	}
	return stat, true, nil
}

// estimateIKZEPITSavings derives the owner's marginal PIT rate from their
// latest salary record on or before year-end, then approximates the tax
// saved by deducting the year's IKZE contributions. Returns ok=false when
// the owner has no salary on record.
func (h *Handler) estimateIKZEPITSavings(ctx context.Context, ownerUserID *int, year int, contribution float64) (float64, float64, bool) {
	asOf := time.Date(year, 12, 31, 0, 0, 0, 0, time.UTC)
	monthlyGross, err := h.store.CurrentSalaryFor(ctx, ownerUserID, asOf)
	if err != nil {
		// Skip the PIT card for owners with no salary. Anything else is an
		// infra error that shouldn't silently degrade the UI — log it.
		if !errors.Is(err, ErrNoSalary) {
			h.logger.Warn("ikze pit salary lookup", "owner", ownerUserID, "err", err)
		}
		return 0, 0, false
	}
	monthly, _ := monthlyGross.Float64()
	if monthly <= 0 {
		return 0, 0, false
	}
	annualGross := monthly * 12
	return MarginalPITRate(annualGross), EstimatePITSavings(contribution, annualGross), true
}

// PPKStats serves GET /api/retirement/ppk-stats.
func (h *Handler) PPKStats(w http.ResponseWriter, r *http.Request) {
	owners, err := h.resolveOwners(r.Context(), r.URL.Query().Get("owner_user_id"))
	if err != nil {
		if errors.Is(err, errBadOwnerParam) {
			httputil.WriteBodyValidationError(w, "owner_user_id", "must be an integer",
				r.URL.Query().Get("owner_user_id"))
			return
		}
		h.logger.Error("resolve owners", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := []ppkStat{}
	for _, owner := range owners {
		stat, included, err := h.buildPPKStat(r.Context(), owner)
		if err != nil {
			h.logger.Error("ppk stat", "err", err)
			httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
			return
		}
		if included {
			out = append(out, stat)
		}
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) buildPPKStat(ctx context.Context, ownerUserID *int) (ppkStat, bool, error) {
	accountIDs, err := h.store.AccountIDsForWrapper(ctx, "PPK", ownerUserID)
	if err != nil {
		return ppkStat{}, false, err
	}
	if len(accountIDs) == 0 {
		return ppkStat{}, false, nil
	}
	totals, err := h.store.PPKContributions(ctx, accountIDs)
	if err != nil {
		return ppkStat{}, false, err
	}
	latest, err := h.store.LatestSnapshotValueSum(ctx, accountIDs)
	if err != nil {
		return ppkStat{}, false, err
	}
	return computePPKStat(ownerUserID, totals, latest), true, nil
}

// LimitsForYear serves GET /api/retirement/limits/{year}.
func (h *Handler) LimitsForYear(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		httputil.WriteBodyValidationError(w, "year", "must be an integer", chi.URLParam(r, "year"))
		return
	}
	limits, err := h.store.LimitsForYear(r.Context(), year)
	if err != nil {
		h.logger.Error("limits", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]limitResponse, 0, len(limits))
	for _, l := range limits {
		amt, _ := l.LimitAmount.Float64()
		out = append(out, limitResponse{
			ID: l.ID, Year: l.Year, AccountWrapper: l.AccountWrapper,
			OwnerUserID: l.OwnerUserID, LimitAmount: wire.PyFloat(amt), Notes: l.Notes,
		})
	}
	httputil.WriteJSON(w, http.StatusOK, out)
}

// UpsertLimit serves PUT /api/retirement/limits/{year}/{wrapper}/{owner_user_id}.
func (h *Handler) UpsertLimit(w http.ResponseWriter, r *http.Request) {
	year, err := strconv.Atoi(chi.URLParam(r, "year"))
	if err != nil {
		httputil.WriteBodyValidationError(w, "year", "must be an integer", chi.URLParam(r, "year"))
		return
	}
	wrapper := chi.URLParam(r, "wrapper")
	ownerParam := chi.URLParam(r, "owner_user_id")
	// The literal "null" addresses the jointly-owned (Shared) limit row.
	var ownerUserID *int
	if ownerParam != "null" {
		n, err := strconv.Atoi(ownerParam)
		if err != nil {
			httputil.WriteBodyValidationError(w, "owner_user_id", "must be an integer", ownerParam)
			return
		}
		ownerUserID = &n
	}
	raw := map[string]json.RawMessage{}
	if !httputil.DecodeJSON(w, r, 1<<16, &raw) {
		return
	}
	req, vErr := buildLimitRequest(raw, h.now)
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	l, err := h.store.UpsertLimit(r.Context(), year, wrapper, ownerUserID, req.LimitAmount, req.Notes)
	if err != nil {
		h.logger.Error("upsert limit", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	amt, _ := l.LimitAmount.Float64()
	httputil.WriteJSON(w, http.StatusOK, limitResponse{
		ID: l.ID, Year: l.Year, AccountWrapper: l.AccountWrapper,
		OwnerUserID: l.OwnerUserID, LimitAmount: wire.PyFloat(amt), Notes: l.Notes,
	})
}

// GeneratePPKContributions serves POST /api/retirement/ppk-contributions/generate.
func (h *Handler) GeneratePPKContributions(w http.ResponseWriter, r *http.Request) {
	raw := map[string]json.RawMessage{}
	if !httputil.DecodeJSON(w, r, 1<<16, &raw) {
		return
	}
	req, vErr := buildGenerateRequest(raw, h.now)
	if vErr != nil {
		httputil.WritePydanticError(w, vErr)
		return
	}
	outcome, err := GeneratePPK(r.Context(), h.store, req.OwnerUserID, req.Month, req.Year,
		GenerateOptions{IncludeWelcome: req.IncludeWelcome, IncludeAnnual: req.IncludeAnnual})
	if err != nil {
		h.writeGenerateError(w, req, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, computePPKGenerateResponse(
		req, outcome.Gross, outcome.EmployeeAmt, outcome.EmployerAmt, outcome.Subsidy, outcome.Result))
}

// writeGenerateError maps GeneratePPK sentinel errors to HTTP responses.
func (h *Handler) writeGenerateError(w http.ResponseWriter, req generateRequest, err error) {
	switch {
	case errors.Is(err, ErrNoSalary):
		httputil.WriteDetailError(w, http.StatusBadRequest,
			fmt.Sprintf("No salary record found for user %s in %d/%d",
				ownerLabel(req.OwnerUserID), req.Month, req.Year))
	case errors.Is(err, ErrNotUOP):
		httputil.WriteDetailError(w, http.StatusBadRequest,
			fmt.Sprintf("PPK contributions require a UOP (employment) salary for user %s",
				ownerLabel(req.OwnerUserID)))
	case errors.Is(err, ErrUserNotFound):
		httputil.WriteDetailError(w, http.StatusNotFound,
			fmt.Sprintf("User %s not found", ownerLabel(req.OwnerUserID)))
	case errors.Is(err, ErrNoPPKAccount):
		httputil.WriteDetailError(w, http.StatusNotFound,
			fmt.Sprintf("No active PPK account found for user %s. "+
				"Mark one PPK account as receiving contributions.",
				ownerLabel(req.OwnerUserID)))
	case errors.Is(err, ErrContributionsExist):
		httputil.WriteDetailError(w, http.StatusConflict,
			fmt.Sprintf("Contributions already exist for %d/%d", req.Month, req.Year))
	default:
		h.logger.Error("generate ppk contributions", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
	}
}

// errBadOwnerParam marks a non-integer owner_user_id query value.
var errBadOwnerParam = errors.New("owner_user_id must be an integer")

func (h *Handler) resolveOwners(ctx context.Context, queryOwner string) ([]*int, error) {
	if queryOwner != "" {
		n, err := strconv.Atoi(queryOwner)
		if err != nil {
			return nil, errBadOwnerParam
		}
		return []*int{&n}, nil
	}
	ids, err := h.store.UserIDs(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*int, 0, len(ids))
	for i := range ids {
		id := ids[i]
		out = append(out, &id)
	}
	return out, nil
}

// ownerLabel renders an owner_user_id for error messages.
func ownerLabel(id *int) string {
	if id == nil {
		return "Shared"
	}
	return strconv.Itoa(*id)
}
