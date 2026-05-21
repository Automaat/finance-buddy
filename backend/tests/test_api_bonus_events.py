"""Tests for bonus event API endpoints."""

from datetime import date

from tests.factories import create_test_bonus_event


def test_get_all_bonus_events_empty(test_client):
    response = test_client.get("/api/bonuses")

    assert response.status_code == 200
    data = response.json()
    assert data["bonus_events"] == []
    assert data["total_count"] == 0
    assert data["available_companies"] == []


def test_get_all_bonus_events_success(test_client, test_db_session):
    create_test_bonus_event(test_db_session, bonus_date=date(2024, 12, 15))
    create_test_bonus_event(
        test_db_session, bonus_date=date(2024, 6, 1), amount=5000.0, owner="Ewa"
    )

    response = test_client.get("/api/bonuses")

    assert response.status_code == 200
    data = response.json()
    assert data["total_count"] == 2
    assert data["bonus_events"][0]["amount"] == 20000.0
    assert data["bonus_events"][1]["amount"] == 5000.0


def test_get_all_bonus_events_filter_by_owner(test_client, test_db_session):
    create_test_bonus_event(test_db_session, owner="Marcin")
    create_test_bonus_event(test_db_session, bonus_date=date(2024, 6, 1), owner="Ewa")

    response = test_client.get("/api/bonuses?owner=Marcin")

    assert response.status_code == 200
    assert response.json()["total_count"] == 1
    assert response.json()["bonus_events"][0]["owner"] == "Marcin"


def test_create_bonus_event_success(test_client):
    payload = {
        "date": "2024-12-15",
        "amount": 18000.0,
        "currency": "PLN",
        "type": "annual",
        "company": "Test Co",
        "owner": "Marcin",
        "contract_type": "UOP",
        "notes": "Year-end",
    }

    response = test_client.post("/api/bonuses", json=payload)

    assert response.status_code == 201
    body = response.json()
    assert body["amount"] == 18000.0
    assert body["type"] == "annual"
    assert body["notes"] == "Year-end"
    assert body["is_active"] is True


def test_create_bonus_event_invalid_currency(test_client):
    response = test_client.post(
        "/api/bonuses",
        json={
            "date": "2024-12-15",
            "amount": 100.0,
            "currency": "XYZ",
            "type": "annual",
            "company": "Co",
            "owner": "Marcin",
            "contract_type": "UOP",
        },
    )
    assert response.status_code == 422


def test_get_bonus_event_by_id(test_client, test_db_session):
    bonus = create_test_bonus_event(test_db_session)

    response = test_client.get(f"/api/bonuses/{bonus.id}")

    assert response.status_code == 200
    assert response.json()["id"] == bonus.id


def test_get_bonus_event_not_found(test_client):
    response = test_client.get("/api/bonuses/99999")
    assert response.status_code == 404


def test_update_bonus_event(test_client, test_db_session):
    bonus = create_test_bonus_event(test_db_session)

    response = test_client.patch(
        f"/api/bonuses/{bonus.id}",
        json={"amount": 25000.0, "notes": "revised"},
    )

    assert response.status_code == 200
    body = response.json()
    assert body["amount"] == 25000.0
    assert body["notes"] == "revised"


def test_delete_bonus_event(test_client, test_db_session):
    bonus = create_test_bonus_event(test_db_session)

    response = test_client.delete(f"/api/bonuses/{bonus.id}")
    assert response.status_code == 204

    follow_up = test_client.get(f"/api/bonuses/{bonus.id}")
    assert follow_up.status_code == 404
