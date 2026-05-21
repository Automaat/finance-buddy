"""Black-box tests for /api/snapshots — list, get, create, update."""

from __future__ import annotations

import httpx
import psycopg2
import pytest

from fixtures.seed import ACCOUNT_MARCIN_BANK, SNAPSHOT_DATES


def _account_id_by_name(client: httpx.Client, name: str) -> int:
    response = client.get("/api/accounts")
    assert response.status_code == 200, response.text
    body = response.json()
    for account in (*body["assets"], *body["liabilities"]):
        if account["name"] == name:
            return int(account["id"])
    raise AssertionError(f"Account {name!r} not found in /api/accounts response")


def _hard_delete_snapshot(dsn: str, snapshot_id: int) -> None:
    """No DELETE endpoint on /api/snapshots — clean up directly via Postgres.

    Removes snapshot_values, snapshot_aggregates (populated by recompute_for_snapshot
    on every create/update), and the snapshot row itself.
    """
    with psycopg2.connect(dsn) as conn, conn.cursor() as cur:
        cur.execute("DELETE FROM snapshot_values WHERE snapshot_id = %s", (snapshot_id,))
        cur.execute("DELETE FROM snapshot_aggregates WHERE snapshot_id = %s", (snapshot_id,))
        cur.execute("DELETE FROM snapshots WHERE id = %s", (snapshot_id,))


@pytest.mark.golden
def test_list_snapshots_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    from _golden import assert_matches_golden

    response = client.get("/api/snapshots")
    assert response.status_code == 200, response.text
    assert_matches_golden("snapshots_list", response.json(), update=update_golden)


def test_list_snapshots_returns_seeded_dates(client: httpx.Client) -> None:
    response = client.get("/api/snapshots")
    assert response.status_code == 200, response.text
    dates = {item["date"] for item in response.json()}
    for snap_date in SNAPSHOT_DATES:
        assert snap_date.isoformat() in dates


def test_get_snapshot_by_id(client: httpx.Client) -> None:
    listing = client.get("/api/snapshots")
    assert listing.status_code == 200, listing.text
    snapshot_id = listing.json()[0]["id"]

    response = client.get(f"/api/snapshots/{snapshot_id}")
    assert response.status_code == 200, response.text
    body = response.json()
    assert body["id"] == snapshot_id
    assert "values" in body
    assert isinstance(body["values"], list)
    assert len(body["values"]) > 0


def test_get_snapshot_not_found(client: httpx.Client) -> None:
    response = client.get("/api/snapshots/999999")
    assert response.status_code == 404, response.text
    assert "detail" in response.json()


@pytest.mark.mutates
def test_create_snapshot_happy_path(client: httpx.Client, database_url: str) -> None:
    account_id = _account_id_by_name(client, ACCOUNT_MARCIN_BANK)
    payload = {
        "date": "2030-03-31",
        "notes": "bb-create-happy",
        "values": [{"account_id": account_id, "value": 12345.67}],
    }
    created_id: int | None = None
    try:
        response = client.post("/api/snapshots", json=payload)
        assert response.status_code == 201, response.text
        body = response.json()
        created_id = body["id"]
        assert body["date"] == "2030-03-31"
        assert body["notes"] == "bb-create-happy"
        assert len(body["values"]) == 1
        assert body["values"][0]["account_id"] == account_id
        assert body["values"][0]["value"] == pytest.approx(12345.67)
    finally:
        if created_id is not None:
            _hard_delete_snapshot(database_url, created_id)


def test_create_snapshot_validation_error(client: httpx.Client) -> None:
    payload = {"date": "2030-04-30", "notes": "bb-invalid", "values": []}
    response = client.post("/api/snapshots", json=payload)
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


@pytest.mark.mutates
def test_update_snapshot_happy_path(client: httpx.Client, database_url: str) -> None:
    account_id = _account_id_by_name(client, ACCOUNT_MARCIN_BANK)

    create_payload = {
        "date": "2030-05-31",
        "notes": "bb-update-initial",
        "values": [{"account_id": account_id, "value": 1000.0}],
    }
    create_resp = client.post("/api/snapshots", json=create_payload)
    assert create_resp.status_code == 201, create_resp.text
    snapshot_id = create_resp.json()["id"]

    try:
        update_payload = {
            "notes": "bb-update-final",
            "values": [{"account_id": account_id, "value": 2500.0}],
        }
        response = client.put(f"/api/snapshots/{snapshot_id}", json=update_payload)
        assert response.status_code == 200, response.text
        body = response.json()
        assert body["id"] == snapshot_id
        assert body["notes"] == "bb-update-final"
        assert len(body["values"]) == 1
        assert body["values"][0]["value"] == pytest.approx(2500.0)
    finally:
        _hard_delete_snapshot(database_url, snapshot_id)


def test_update_snapshot_validation_error(client: httpx.Client) -> None:
    listing = client.get("/api/snapshots")
    assert listing.status_code == 200, listing.text
    snapshot_id = listing.json()[0]["id"]

    response = client.put(f"/api/snapshots/{snapshot_id}", json={"values": []})
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()
