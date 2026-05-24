package rules

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// ruleWire mirrors a Rule on the JSON wire. Money/% values are quoted
// strings (matches the rest of the codebase's decimal serialization
// convention); dates are YYYY-MM-DD without timezone suffix.
type ruleWire struct {
	Key             string    `json:"key"`
	Name            string    `json:"name"`
	Category        string    `json:"category"`
	Value           dimString `json:"value"`
	Unit            string    `json:"unit"`
	Year            int       `json:"year"`
	EffectiveDate   isoDate   `json:"effective_date"`
	SourceURL       string    `json:"source_url"`
	LastCheckedDate isoDate   `json:"last_checked_date"`
	Description     string    `json:"description"`
}

type listResponse struct {
	Rules []ruleWire `json:"rules"`
}

// Handler is the HTTP boundary for /api/rules.
type Handler struct {
	logger *slog.Logger
}

// NewHandler wires the logger. No DB dependency — rules are code-defined.
func NewHandler(logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{logger: logger}
}

// List serves GET /api/rules. Optional `?category=` filter restricts to
// a single bucket (e.g. "ike_limit", "pit"); unknown categories return an
// empty list rather than an error so the UI can probe without 422s.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	var src []Rule
	if category != "" {
		src = ByCategory(category)
	} else {
		src = All()
	}
	out := make([]ruleWire, 0, len(src))
	for i := range src {
		out = append(out, toWire(&src[i]))
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(listResponse{Rules: out}); err != nil {
		h.logger.Error("encode rules", "err", err)
	}
}

func toWire(r *Rule) ruleWire {
	return ruleWire{
		Key:             r.Key,
		Name:            r.Name,
		Category:        r.Category,
		Value:           dimString(r.Value),
		Unit:            r.Unit,
		Year:            r.Year,
		EffectiveDate:   isoDate(r.EffectiveDate),
		SourceURL:       r.SourceURL,
		LastCheckedDate: isoDate(r.LastCheckedDate),
		Description:     r.Description,
	}
}

// dimString marshals a decimal.Decimal as a JSON string preserving its
// stored precision (e.g. "28260", "0.12"). Distinct from the moneyJSON
// helper elsewhere because we don't want forced 2-decimal padding — these
// rules carry both monetary and percentage values.
type dimString decimal.Decimal

func (d dimString) MarshalJSON() ([]byte, error) {
	v := decimal.Decimal(d)
	return fmt.Appendf(nil, "%q", v.String()), nil
}

// isoDate marshals a time.Time as "YYYY-MM-DD" — naïve, no zone suffix,
// matches the wire convention used by /api/config etc.
type isoDate time.Time

func (d isoDate) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, "%q", time.Time(d).Format("2006-01-02")), nil
}
