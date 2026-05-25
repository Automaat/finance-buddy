"""Black-box tests for /api/accounts — CRUD coverage against the seeded backend."""

from __future__ import annotations

import httpx
import pytest

from _golden import assert_matches_golden
from fixtures.seed import (
    ACCOUNT_MARCIN_BANK,
    ACCOUNT_MARCIN_MORTGAGE,
    PERSONA_MARCIN,
)


def _find_account_id(client: httpx.Client, name: str) -> int:
    response = client.get("/api/accounts")
    assert response.status_code == 200, response.text
    body = response.json()
    for bucket in ("assets", "liabilities"):
        for account in body.get(bucket, []):
            if account["name"] == name:
                return int(account["id"])
    raise AssertionError(f"Seeded account {name!r} not found in /api/accounts response")


@pytest.mark.golden
def test_get_accounts_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    response = client.get("/api/accounts")
    assert response.status_code == 200, response.text
    assert_matches_golden("accounts_list", response.json(), update=update_golden)


def test_get_accounts_buckets_assets_and_liabilities(client: httpx.Client) -> None:
    response = client.get("/api/accounts")
    assert response.status_code == 200, response.text
    body = response.json()
    asset_names = {a["name"] for a in body["assets"]}
    liability_names = {a["name"] for a in body["liabilities"]}
    assert ACCOUNT_MARCIN_BANK in asset_names
    assert ACCOUNT_MARCIN_MORTGAGE in liability_names


def test_create_account_happy_path(
    client: httpx.Client, request: pytest.FixtureRequest, owner_ids: dict[str, int]
) -> None:
    unique_name = f"bb-test-{request.node.name}-account"
    created_id: int | None = None
    try:
        response = client.post(
            "/api/accounts",
            json={
                "name": unique_name,
                "type": "asset",
                "category": "bank",
                "owner_user_id": owner_ids[PERSONA_MARCIN],
                "currency": "PLN",
                "purpose": "general",
                "receives_contributions": True,
            },
        )
        assert response.status_code == 201, response.text
        body = response.json()
        created_id = int(body["id"])
        assert body["name"] == unique_name
        assert body["type"] == "asset"
        assert body["category"] == "bank"
        assert body["owner_user_id"] == owner_ids[PERSONA_MARCIN]
        assert body["is_active"] is True
        assert body["current_value"] == 0.0
    finally:
        if created_id is not None:
            client.delete(f"/api/accounts/{created_id}")


def test_create_account_validation_error(client: httpx.Client) -> None:
    response = client.post(
        "/api/accounts",
        json={
            "name": "   ",
            "type": "asset",
            "category": "bank",
            "owner_user_id": None,
            "purpose": "general",
        },
    )
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


def test_update_account_happy_path(
    client: httpx.Client, request: pytest.FixtureRequest, owner_ids: dict[str, int]
) -> None:
    unique_name = f"bb-test-{request.node.name}-account"
    renamed = f"{unique_name}-renamed"
    create_response = client.post(
        "/api/accounts",
        json={
            "name": unique_name,
            "type": "asset",
            "category": "bank",
            "owner_user_id": owner_ids[PERSONA_MARCIN],
            "currency": "PLN",
            "purpose": "general",
        },
    )
    assert create_response.status_code == 201, create_response.text
    created_id = int(create_response.json()["id"])

    try:
        response = client.put(
            f"/api/accounts/{created_id}",
            json={"name": renamed, "category": "saving_account"},
        )
        assert response.status_code == 200, response.text
        body = response.json()
        assert body["id"] == created_id
        assert body["name"] == renamed
        assert body["category"] == "saving_account"
    finally:
        client.delete(f"/api/accounts/{created_id}")


def test_update_account_validation_error(client: httpx.Client) -> None:
    account_id = _find_account_id(client, ACCOUNT_MARCIN_BANK)
    response = client.put(f"/api/accounts/{account_id}", json={"name": "   "})
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


def test_excluded_from_fire_round_trip(
    client: httpx.Client, request: pytest.FixtureRequest, owner_ids: dict[str, int]
) -> None:
    unique_name = f"bb-test-{request.node.name}-account"
    create_response = client.post(
        "/api/accounts",
        json={
            "name": unique_name,
            "type": "asset",
            "category": "real_estate",
            "owner_user_id": owner_ids[PERSONA_MARCIN],
            "currency": "PLN",
            "purpose": "general",
            "excluded_from_fire": True,
        },
    )
    assert create_response.status_code == 201, create_response.text
    created = create_response.json()
    created_id = int(created["id"])
    assert created["excluded_from_fire"] is True

    try:
        toggled = client.put(
            f"/api/accounts/{created_id}", json={"excluded_from_fire": False}
        )
        assert toggled.status_code == 200, toggled.text
        assert toggled.json()["excluded_from_fire"] is False

        listing = client.get("/api/accounts")
        assert listing.status_code == 200, listing.text
        match = next(
            a for a in listing.json()["assets"] if a["id"] == created_id
        )
        assert match["excluded_from_fire"] is False
    finally:
        client.delete(f"/api/accounts/{created_id}")


def test_excluded_from_fire_defaults_to_false(
    client: httpx.Client, request: pytest.FixtureRequest, owner_ids: dict[str, int]
) -> None:
    unique_name = f"bb-test-{request.node.name}-account"
    response = client.post(
        "/api/accounts",
        json={
            "name": unique_name,
            "type": "asset",
            "category": "bank",
            "owner_user_id": owner_ids[PERSONA_MARCIN],
            "currency": "PLN",
            "purpose": "general",
        },
    )
    assert response.status_code == 201, response.text
    created_id = int(response.json()["id"])
    try:
        assert response.json()["excluded_from_fire"] is False
    finally:
        client.delete(f"/api/accounts/{created_id}")


def test_delete_account_happy_path(
    client: httpx.Client, request: pytest.FixtureRequest, owner_ids: dict[str, int]
) -> None:
    unique_name = f"bb-test-{request.node.name}-account"
    create_response = client.post(
        "/api/accounts",
        json={
            "name": unique_name,
            "type": "asset",
            "category": "bank",
            "owner_user_id": owner_ids[PERSONA_MARCIN],
            "currency": "PLN",
            "purpose": "general",
        },
    )
    assert create_response.status_code == 201, create_response.text
    created_id = int(create_response.json()["id"])

    response = client.delete(f"/api/accounts/{created_id}")
    assert response.status_code == 204, response.text

    listing = client.get("/api/accounts")
    assert listing.status_code == 200, listing.text
    listed_ids = {a["id"] for a in listing.json()["assets"]} | {
        a["id"] for a in listing.json()["liabilities"]
    }
    assert created_id not in listed_ids
