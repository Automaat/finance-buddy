"""Tests for the equity grants API endpoints."""

from tests.factories import create_test_equity_grant


def _valid_payload(**overrides):
    base = {
        "grant_date": "2024-01-01",
        "type": "rsu",
        "company": "Co X",
        "owner": "Marcin",
        "total_shares": 4800,
        "currency": "USD",
        "vest_start_date": "2024-01-01",
        "vest_cliff_months": 12,
        "vest_total_months": 48,
        "vest_frequency": "monthly",
    }
    base.update(overrides)
    return base


def test_list_equity_grants_empty(test_client):
    response = test_client.get("/api/equity-grants")
    assert response.status_code == 200
    data = response.json()
    assert data["equity_grants"] == []
    assert data["total_count"] == 0


def test_create_equity_grant_rsu(test_client):
    response = test_client.post("/api/equity-grants", json=_valid_payload())
    assert response.status_code == 201
    body = response.json()
    assert body["type"] == "rsu"
    assert body["total_shares"] == 4800
    assert body["vested_shares_today"] >= 0


def test_create_option_requires_strike(test_client):
    payload = _valid_payload(type="option", strike_price=None)
    response = test_client.post("/api/equity-grants", json=payload)
    assert response.status_code == 422


def test_create_with_custom_schedule(test_client):
    payload = _valid_payload(
        vest_custom_schedule=[
            {"month": 12, "pct": 10},
            {"month": 24, "pct": 20},
            {"month": 36, "pct": 30},
            {"month": 48, "pct": 40},
        ]
    )
    response = test_client.post("/api/equity-grants", json=payload)
    assert response.status_code == 201
    assert len(response.json()["vest_custom_schedule"]) == 4


def test_filter_by_owner(test_client, test_db_session):
    create_test_equity_grant(test_db_session, owner="Marcin")
    create_test_equity_grant(test_db_session, owner="Ewa")

    response = test_client.get("/api/equity-grants?owner=Marcin")
    assert response.status_code == 200
    assert response.json()["total_count"] == 1


def test_update_equity_grant(test_client, test_db_session):
    grant = create_test_equity_grant(test_db_session)

    response = test_client.patch(f"/api/equity-grants/{grant.id}", json={"total_shares": 5000})
    assert response.status_code == 200
    assert response.json()["total_shares"] == 5000


def test_delete_equity_grant(test_client, test_db_session):
    grant = create_test_equity_grant(test_db_session)

    response = test_client.delete(f"/api/equity-grants/{grant.id}")
    assert response.status_code == 204

    follow = test_client.get(f"/api/equity-grants/{grant.id}")
    assert follow.status_code == 404
