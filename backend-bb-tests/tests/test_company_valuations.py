"""Black-box tests for /api/company-valuations — company valuation CRUD."""

from __future__ import annotations

import httpx
import pytest

from fixtures.seed import COMPANY_MARCIN_EMPLOYER


def test_list_company_valuations_includes_seeded(client: httpx.Client) -> None:
    response = client.get("/api/company-valuations")
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["total_count"] >= 1
    companies = {r["company"] for r in body["company_valuations"]}
    assert COMPANY_MARCIN_EMPLOYER in companies
    assert COMPANY_MARCIN_EMPLOYER in body["available_companies"]


@pytest.mark.golden
def test_list_company_valuations_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    from _golden import assert_matches_golden

    response = client.get("/api/company-valuations")
    assert response.status_code == 200, response.text
    assert_matches_golden("company_valuations_list", response.json(), update=update_golden)


def test_get_company_valuation_by_id_returns_seeded_record(client: httpx.Client) -> None:
    listing = client.get("/api/company-valuations").json()
    sample = listing["company_valuations"][0]
    response = client.get(f"/api/company-valuations/{sample['id']}")
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["id"] == sample["id"]
    assert body["company"] == sample["company"]
    assert body["fmv_per_share"] == sample["fmv_per_share"]


def test_get_company_valuation_not_found(client: httpx.Client) -> None:
    response = client.get("/api/company-valuations/999999")
    assert response.status_code == 404, response.text
    assert "detail" in response.json()


def test_create_company_valuation_happy_path(
    client: httpx.Client, request: pytest.FixtureRequest
) -> None:
    created_id: int | None = None
    try:
        payload = {
            "company": f"bb-test-{request.node.name}",
            "date": "2025-12-31",
            "currency": "USD",
            "fmv_per_share": 15.0,
            "fmv_low": 13.0,
            "fmv_high": 17.0,
            "source": "409a",
            "common_stock_discount_pct": 20.0,
            "notes": "test valuation",
        }
        response = client.post("/api/company-valuations", json=payload)
        assert response.status_code == 201, response.text
        body = response.json()
        created_id = body["id"]
        assert body["fmv_per_share"] == 15.0
        assert body["company"] == payload["company"]
        assert body["is_active"] is True
    finally:
        if created_id is not None:
            client.delete(f"/api/company-valuations/{created_id}")


def test_create_company_valuation_validation_error(client: httpx.Client) -> None:
    payload = {
        "company": "bb-test-bad",
        "date": "2025-12-31",
        "currency": "USD",
        "fmv_per_share": 10.0,
        "fmv_low": 20.0,
        "fmv_high": 15.0,
        "source": "409a",
    }
    response = client.post("/api/company-valuations", json=payload)
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


def test_update_company_valuation_happy_path(
    client: httpx.Client, request: pytest.FixtureRequest
) -> None:
    created_id: int | None = None
    try:
        create_payload = {
            "company": f"bb-test-{request.node.name}",
            "date": "2025-11-30",
            "currency": "USD",
            "fmv_per_share": 10.0,
            "source": "estimate",
        }
        created = client.post("/api/company-valuations", json=create_payload)
        assert created.status_code == 201, created.text
        created_id = created.json()["id"]

        response = client.patch(
            f"/api/company-valuations/{created_id}",
            json={"fmv_per_share": 11.0, "notes": "updated"},
        )
        assert response.status_code == 200, response.text
        body = response.json()
        assert body["fmv_per_share"] == 11.0
        assert body["notes"] == "updated"
    finally:
        if created_id is not None:
            client.delete(f"/api/company-valuations/{created_id}")


def test_update_company_valuation_validation_error(
    client: httpx.Client, request: pytest.FixtureRequest
) -> None:
    created_id: int | None = None
    try:
        create_payload = {
            "company": f"bb-test-{request.node.name}",
            "date": "2025-11-30",
            "currency": "USD",
            "fmv_per_share": 10.0,
            "source": "estimate",
        }
        created = client.post("/api/company-valuations", json=create_payload)
        assert created.status_code == 201, created.text
        created_id = created.json()["id"]

        response = client.patch(
            f"/api/company-valuations/{created_id}",
            json={"fmv_per_share": -1.0},
        )
        assert response.status_code >= 400, response.text
        assert "detail" in response.json()
    finally:
        if created_id is not None:
            client.delete(f"/api/company-valuations/{created_id}")


def test_delete_company_valuation_happy_path(
    client: httpx.Client, request: pytest.FixtureRequest
) -> None:
    create_payload = {
        "company": f"bb-test-{request.node.name}",
        "date": "2025-10-31",
        "currency": "USD",
        "fmv_per_share": 9.5,
        "source": "estimate",
    }
    created = client.post("/api/company-valuations", json=create_payload)
    assert created.status_code == 201, created.text
    created_id = created.json()["id"]

    response = client.delete(f"/api/company-valuations/{created_id}")
    assert response.status_code == 204, response.text

    follow = client.get(f"/api/company-valuations/{created_id}")
    assert follow.status_code == 404, follow.text
