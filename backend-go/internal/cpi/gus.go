package cpi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
)

// GUS BDL configuration — matches backend/app/services/inflation.py.
const (
	GUSVariableID = 217230
	GUSBaseURL    = "https://bdl.stat.gov.pl/api/v1"
	GUSSourceTag  = "GUS-BDL-217230"
	gusTimeout    = 15 * time.Second
)

// GUSFetcher pulls annual yoy CPI from the GUS BDL API.
type GUSFetcher struct {
	client *http.Client
}

// NewGUSFetcher wires a fetcher with the standard 15s timeout.
func NewGUSFetcher() *GUSFetcher {
	return &GUSFetcher{
		client: &http.Client{Timeout: gusTimeout},
	}
}

type gusValue struct {
	Year string `json:"year"`
	Val  any    `json:"val"`
}

type gusResult struct {
	Values []gusValue `json:"values"`
}

type gusResponse struct {
	Results []gusResult `json:"results"`
}

// Fetch returns (year, yoy_rate) tuples sorted ascending by year. Empty slice
// when GUS responds with no data. Non-2xx responses surface as errors.
func (g *GUSFetcher) Fetch(ctx context.Context) ([]YearRate, error) {
	url := fmt.Sprintf(
		"%s/data/by-variable/%d?unit-level=0&format=json&page-size=100",
		GUSBaseURL, GUSVariableID,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gus fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil, fmt.Errorf("gus http %d", resp.StatusCode)
	}
	var body gusResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode gus payload: %w", err)
	}
	if len(body.Results) == 0 {
		return []YearRate{}, nil
	}
	values := body.Results[0].Values
	out := make([]YearRate, 0, len(values))
	for i, v := range values {
		year, err := strconv.Atoi(v.Year)
		if err != nil {
			return nil, fmt.Errorf("gus value[%d]: invalid year %q: %w", i, v.Year, err)
		}
		rate, err := decimal.NewFromString(fmt.Sprintf("%v", v.Val))
		if err != nil {
			return nil, fmt.Errorf("gus value[%d] year=%d: invalid yoy %v: %w", i, year, v.Val, err)
		}
		out = append(out, YearRate{Year: year, YoY: rate})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Year < out[j].Year })
	return out, nil
}
