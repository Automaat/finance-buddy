"""Black-box tests for /api/assets — CRUD coverage against the seeded backend."""

from __future__ import annotations

import httpx
import pytest

from _golden import assert_matches_golden
from fixtures.seed import ASSET_MARCIN_APARTMENT


def _find_asset_id(client: httpx.Client, name: str) -> int:
    response = client.get("/api/assets")
    assert response.status_code == 200, response.text
    for asset in response.json().get("assets", []):
        if asset["name"] == name:
            return int(asset["id"])
    raise AssertionError(f"Seeded asset {name!r} not found in /api/assets response")


@pytest.mark.golden
def test_get_assets_matches_golden(client: httpx.Client, update_golden: bool) -> None:
    response = client.get("/api/assets")
    assert response.status_code == 200, response.text
    assert_matches_golden("assets_list", response.json(), update=update_golden)


def test_get_assets_includes_seeded(client: httpx.Client) -> None:
    response = client.get("/api/assets")
    assert response.status_code == 200, response.text
    names = {a["name"] for a in response.json()["assets"]}
    assert ASSET_MARCIN_APARTMENT in names


def test_create_asset_happy_path(client: httpx.Client, request: pytest.FixtureRequest) -> None:
    unique_name = f"bb-test-{request.node.name}-asset"
    created_id: int | None = None
    try:
        response = client.post("/api/assets", json={"name": unique_name})
        assert response.status_code == 201, response.text
        body = response.json()
        created_id = int(body["id"])
        assert body["name"] == unique_name
        assert body["is_active"] is True
        assert body["current_value"] == 0.0
    finally:
        if created_id is not None:
            client.delete(f"/api/assets/{created_id}")


def test_create_asset_validation_error(client: httpx.Client) -> None:
    response = client.post("/api/assets", json={"name": "   "})
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


def test_update_asset_happy_path(client: httpx.Client, request: pytest.FixtureRequest) -> None:
    unique_name = f"bb-test-{request.node.name}-asset"
    renamed = f"{unique_name}-renamed"
    create_response = client.post("/api/assets", json={"name": unique_name})
    assert create_response.status_code == 201, create_response.text
    created_id = int(create_response.json()["id"])

    try:
        response = client.put(f"/api/assets/{created_id}", json={"name": renamed})
        assert response.status_code == 200, response.text
        body = response.json()
        assert body["id"] == created_id
        assert body["name"] == renamed
    finally:
        client.delete(f"/api/assets/{created_id}")


def test_update_asset_validation_error(client: httpx.Client) -> None:
    asset_id = _find_asset_id(client, ASSET_MARCIN_APARTMENT)
    response = client.put(f"/api/assets/{asset_id}", json={"name": "   "})
    assert response.status_code >= 400, response.text
    assert "detail" in response.json()


def test_delete_asset_happy_path(client: httpx.Client, request: pytest.FixtureRequest) -> None:
    unique_name = f"bb-test-{request.node.name}-asset"
    create_response = client.post("/api/assets", json={"name": unique_name})
    assert create_response.status_code == 201, create_response.text
    created_id = int(create_response.json()["id"])

    response = client.delete(f"/api/assets/{created_id}")
    assert response.status_code == 204, response.text

    listing = client.get("/api/assets")
    assert listing.status_code == 200, listing.text
    listed_ids = {a["id"] for a in listing.json()["assets"]}
    assert created_id not in listed_ids
