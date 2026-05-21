package snapshots

import (
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
)

type valueResponse struct {
	ID          int     `json:"id"`
	AssetID     *int    `json:"asset_id"`
	AssetName   *string `json:"asset_name"`
	AccountID   *int    `json:"account_id"`
	AccountName *string `json:"account_name"`
	Value       pyFloat `json:"value"`
}

type response struct {
	ID     int             `json:"id"`
	Date   isoDate         `json:"date"`
	Notes  *string         `json:"notes"`
	Values []valueResponse `json:"values"`
}

type listItem struct {
	ID            int     `json:"id"`
	Date          isoDate `json:"date"`
	Notes         *string `json:"notes"`
	TotalNetWorth pyFloat `json:"total_net_worth"`
}

// Handler is the HTTP boundary for /api/snapshots.
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

func buildResponse(snap *Snapshot, values []Value) response {
	out := response{
		ID:    snap.ID,
		Date:  isoDate(snap.Date),
		Notes: snap.Notes,
	}
	out.Values = make([]valueResponse, 0, len(values))
	for _, v := range values {
		f, _ := v.Value.Float64()
		out.Values = append(out.Values, valueResponse{
			ID: v.ID, AssetID: v.AssetID, AssetName: v.AssetName,
			AccountID: v.AccountID, AccountName: v.AccountName,
			Value: pyFloat(f),
		})
	}
	return out
}

// List serves GET /api/snapshots.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("list snapshots", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	out := make([]listItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, listItem{
			ID: row.ID, Date: isoDate(row.Date), Notes: row.Notes,
			TotalNetWorth: pyFloat(row.TotalNetWorth),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// Get serves GET /api/snapshots/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	snap, values, err := h.store.Get(r.Context(), id)
	if err != nil {
		h.writeStoreError(w, err, id, nil)
		return
	}
	writeJSON(w, http.StatusOK, buildResponse(snap, values))
}

// Create serves POST /api/snapshots.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&raw); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	req, vErr := buildCreateRequest(raw)
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	snap, values, err := h.store.Create(r.Context(), req.Date, req.Notes, req.Values)
	if err != nil {
		h.writeStoreError(w, err, 0, &req.Date)
		return
	}
	writeJSON(w, http.StatusCreated, buildResponse(snap, values))
}

// Update serves PUT /api/snapshots/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r)
	if !ok {
		return
	}
	raw := map[string]json.RawMessage{}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&raw); err != nil {
		writeValidationError(w, "body", "Invalid JSON body", err.Error())
		return
	}
	patch, vErr := buildUpdatePatch(raw)
	if vErr != nil {
		writePydanticError(w, vErr)
		return
	}
	snap, values, err := h.store.Update(r.Context(), id, patch)
	if err != nil {
		h.writeStoreError(w, err, id, patch.Date)
		return
	}
	writeJSON(w, http.StatusOK, buildResponse(snap, values))
}

func (h *Handler) writeStoreError(w http.ResponseWriter, err error, id int, date *time.Time) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeDetailError(w, http.StatusNotFound,
			fmt.Sprintf("Snapshot %d not found", id))
	case errors.Is(err, ErrDuplicateDate):
		d := ""
		if date != nil {
			d = date.Format("2006-01-02")
		}
		writeDetailError(w, http.StatusBadRequest,
			fmt.Sprintf("Snapshot for date %s already exists", d))
	case errors.Is(err, ErrDuplicateAssetIDs):
		writeDetailError(w, http.StatusBadRequest,
			"Duplicate asset IDs in snapshot values")
	case errors.Is(err, ErrDuplicateAccountIDs):
		writeDetailError(w, http.StatusBadRequest,
			"Duplicate account IDs in snapshot values")
	default:
		var mre *MissingReferencesError
		if errors.As(err, &mre) {
			if len(mre.MissingAssets) > 0 {
				writeDetailError(w, http.StatusNotFound,
					fmt.Sprintf("Assets not found: %s", setLiteral(mre.MissingAssets)))
				return
			}
			writeDetailError(w, http.StatusNotFound,
				fmt.Sprintf("Accounts not found: %s", setLiteral(mre.MissingAccounts)))
			return
		}
		h.logger.Error("snapshot store", "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
	}
}

// setLiteral renders a Go []int as Python's `{1, 2, 3}` set-literal output
// so the error detail matches Python's f"Assets not found: {missing}".
func setLiteral(ids []int) string {
	if len(ids) == 0 {
		return "set()"
	}
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, strconv.Itoa(id))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func parseIDParam(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.Atoi(raw)
	if err != nil {
		writeValidationError(w, "snapshot_id", "must be an integer", raw)
		return 0, false
	}
	return id, true
}

// --- wire types ---

type isoDate time.Time

const isoDateLayout = "2006-01-02"

func (d isoDate) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(d).Format(isoDateLayout) + `"`), nil
}

type pyFloat float64

func (f pyFloat) MarshalJSON() ([]byte, error) {
	s := strconv.FormatFloat(float64(f), 'f', -1, 64)
	if !strings.ContainsRune(s, '.') {
		s += ".0"
	}
	return []byte(s), nil
}
