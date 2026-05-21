"""Black-box tests for /api/cpi — CPI series, inflation adjust, refresh."""

from __future__ import annotations

import httpx
import pytest


@pytest.mark.golden
def test_get_cpi_series_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    from _golden import assert_matches_golden

    response = client.get("/api/cpi/series")
    assert response.status_code == 200, response.text
    assert_matches_golden("cpi_series", response.json(), update=update_golden)


def test_get_cpi_series_shape(client: httpx.Client) -> None:
    response = client.get("/api/cpi/series")
    assert response.status_code == 200, response.text
    body = response.json()
    required = {"points", "base_year", "latest_year", "source"}
    assert required.issubset(body.keys())
    assert body["points"], "Seeded CPI series is empty"
    assert {"year", "yoy_rate", "cumulative_index"}.issubset(body["points"][0].keys())


def test_post_cpi_adjust_happy_path(client: httpx.Client) -> None:
    payload = {
        "amount": 1000.0,
        "from_date": "2023-01-01",
        "to_date": "2025-12-31",
    }
    response = client.post("/api/cpi/adjust", json=payload)
    assert response.status_code == 200, response.text
    body = response.json()
    required = {
        "original_amount",
        "adjusted_amount",
        "factor",
        "from_date",
        "to_date",
        "as_of_year",
    }
    assert required.issubset(body.keys())
    assert body["original_amount"] == 1000.0
    assert body["adjusted_amount"] > 0


def test_post_cpi_adjust_validation_error(client: httpx.Client) -> None:
    # Missing required fields → 422 with detail
    response = client.post("/api/cpi/adjust", json={"amount": 100.0})
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


def test_post_cpi_refresh_skipped_external_call() -> None:
    # POST /api/cpi/refresh hits GUS BDL (network + non-deterministic).
    # Skipping intentionally: no offline fixture for GUS, and a real HTTP
    # call would make the suite flaky.
    pytest.skip("POST /api/cpi/refresh hits external GUS BDL API; skipped for determinism")
