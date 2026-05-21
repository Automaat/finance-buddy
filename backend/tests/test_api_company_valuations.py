"""Tests for company valuation API endpoints."""

from tests.factories import create_test_company_valuation


def _valid_payload(**overrides):
    base = {
        "company": "Co X",
        "date": "2024-06-01",
        "currency": "USD",
        "fmv_per_share": 10.0,
        "source": "409a",
    }
    base.update(overrides)
    return base


def test_list_empty(test_client):
    response = test_client.get("/api/company-valuations")
    assert response.status_code == 200
    assert response.json()["total_count"] == 0


def test_create_valuation(test_client):
    response = test_client.post("/api/company-valuations", json=_valid_payload())
    assert response.status_code == 201
    body = response.json()
    assert body["fmv_per_share"] == 10.0


def test_create_with_range(test_client):
    response = test_client.post(
        "/api/company-valuations",
        json=_valid_payload(fmv_low=8.0, fmv_high=14.0),
    )
    assert response.status_code == 201
    body = response.json()
    assert body["fmv_low"] == 8.0
    assert body["fmv_high"] == 14.0


def test_invalid_range_rejected(test_client):
    response = test_client.post(
        "/api/company-valuations",
        json=_valid_payload(fmv_low=20.0),
    )
    assert response.status_code == 422


def test_filter_by_company(test_client, test_db_session):
    create_test_company_valuation(test_db_session, company="A")
    create_test_company_valuation(test_db_session, company="B")

    response = test_client.get("/api/company-valuations?company=A")
    assert response.json()["total_count"] == 1


def test_update_valuation(test_client, test_db_session):
    valuation = create_test_company_valuation(test_db_session)
    response = test_client.patch(
        f"/api/company-valuations/{valuation.id}", json={"fmv_per_share": 15.0}
    )
    assert response.status_code == 200
    assert response.json()["fmv_per_share"] == 15.0


def test_delete_valuation(test_client, test_db_session):
    valuation = create_test_company_valuation(test_db_session)
    response = test_client.delete(f"/api/company-valuations/{valuation.id}")
    assert response.status_code == 204

    follow = test_client.get(f"/api/company-valuations/{valuation.id}")
    assert follow.status_code == 404
