package investment

import (
	"encoding/json"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
)

type response struct {
	Category         string  `json:"category"`
	TotalValue       pyFloat `json:"total_value"`
	TotalContributed pyFloat `json:"total_contributed"`
	Returns          pyFloat `json:"returns"`
	ROIPercentage    pyFloat `json:"roi_percentage"`
}

// Handler is the HTTP boundary for /api/investment.
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

// StockStats serves GET /api/investment/stock-stats.
func (h *Handler) StockStats(w http.ResponseWriter, r *http.Request) {
	h.categoryStats(w, r, "stock")
}

// BondStats serves GET /api/investment/bond-stats.
func (h *Handler) BondStats(w http.ResponseWriter, r *http.Request) {
	h.categoryStats(w, r, "bond")
}

func (h *Handler) categoryStats(w http.ResponseWriter, r *http.Request, category string) {
	stats, err := h.store.StatsForCategory(r.Context(), category)
	if err != nil {
		h.logger.Error("category stats", "category", category, "err", err)
		writeDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	if !stats.HasAccounts {
		writeJSON(w, http.StatusOK, response{Category: category})
		return
	}
	value, _ := stats.TotalValue.Float64()
	contributed, _ := stats.TotalContributed.Float64()
	returns := value - contributed
	roi := 0.0
	if contributed > 0 {
		roi = returns / contributed * 100
	}
	writeJSON(w, http.StatusOK, response{
		Category:         category,
		TotalValue:       pyFloat(value),
		TotalContributed: pyFloat(contributed),
		Returns:          pyFloat(returns),
		ROIPercentage:    pyFloat(math.Round(roi*100) / 100),
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Default().Error("encode response", "err", err, "status", status)
	}
}

func writeDetailError(w http.ResponseWriter, status int, detail string) {
	writeJSON(w, status, map[string]string{"detail": detail})
}

type pyFloat float64

func (f pyFloat) MarshalJSON() ([]byte, error) {
	s := strconv.FormatFloat(float64(f), 'f', -1, 64)
	if !strings.ContainsRune(s, '.') {
		s += ".0"
	}
	return []byte(s), nil
}
