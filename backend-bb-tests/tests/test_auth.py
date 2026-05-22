"""Auth endpoints: login, session gating, admin-only user management.

The shared `client` fixture is logged in as admin and session-scoped, so its
cookie jar must not be mutated here — login flows use fresh clients.
"""

from __future__ import annotations

import httpx

from conftest import ADMIN_PASSWORD, ADMIN_USERNAME


def test_login_success(base_url: str) -> None:
    with httpx.Client(base_url=base_url, timeout=10.0) as http:
        response = http.post(
            "/api/auth/login",
            json={"username": ADMIN_USERNAME, "password": ADMIN_PASSWORD},
        )
        assert response.status_code == 200
        body = response.json()
        assert body["token"]
        assert body["user"]["username"] == ADMIN_USERNAME
        assert body["user"]["is_admin"] is True
        assert "fb_token" in response.cookies


def test_login_wrong_password(base_url: str) -> None:
    with httpx.Client(base_url=base_url, timeout=10.0) as http:
        response = http.post(
            "/api/auth/login",
            json={"username": ADMIN_USERNAME, "password": "wrong"},
        )
        assert response.status_code == 401


def test_gated_route_rejects_unauthenticated(base_url: str) -> None:
    with httpx.Client(base_url=base_url, timeout=10.0) as http:
        assert http.get("/api/accounts").status_code == 401


def test_me_returns_current_user(client: httpx.Client) -> None:
    response = client.get("/api/auth/me")
    assert response.status_code == 200
    assert response.json()["username"] == ADMIN_USERNAME


def test_admin_creates_user_who_can_log_in(client: httpx.Client, base_url: str) -> None:
    created = client.post(
        "/api/auth/users",
        json={"username": "bb-member", "password": "member-pass-1"},
    )
    assert created.status_code == 201
    assert created.json()["is_admin"] is False

    with httpx.Client(base_url=base_url, timeout=10.0) as http:
        login = http.post(
            "/api/auth/login",
            json={"username": "bb-member", "password": "member-pass-1"},
        )
        assert login.status_code == 200


def test_non_admin_cannot_manage_users(client: httpx.Client, base_url: str) -> None:
    client.post(
        "/api/auth/users",
        json={"username": "bb-plain", "password": "plain-pass-1"},
    )
    with httpx.Client(base_url=base_url, timeout=10.0) as http:
        http.post(
            "/api/auth/login",
            json={"username": "bb-plain", "password": "plain-pass-1"},
        )
        assert http.get("/api/auth/users").status_code == 403
        assert (
            http.post(
                "/api/auth/users",
                json={"username": "nope", "password": "nope-pass-1"},
            ).status_code
            == 403
        )
        assert http.put("/api/auth/users/1", json={}).status_code == 403


def test_create_user_with_profile_and_default_ppk(client: httpx.Client) -> None:
    explicit = client.post(
        "/api/auth/users",
        json={
            "username": "bb-profile",
            "password": "profile-pass-1",
            "name": "Profile",
            "surname": "Tester",
            "ppk_employee_rate": 3,
            "ppk_employer_rate": 2.5,
        },
    )
    assert explicit.status_code == 201
    body = explicit.json()
    assert body["name"] == "Profile"
    assert body["surname"] == "Tester"
    assert body["ppk_employee_rate"] == "3.00"
    assert body["ppk_employer_rate"] == "2.50"

    # Omitted rates fall back to the defaults.
    defaulted = client.post(
        "/api/auth/users",
        json={"username": "bb-defaults", "password": "defaults-pass-1"},
    )
    assert defaulted.status_code == 201
    assert defaulted.json()["ppk_employee_rate"] == "2.00"
    assert defaulted.json()["ppk_employer_rate"] == "1.50"


def test_owner_picker_list(client: httpx.Client, base_url: str) -> None:
    created = client.post(
        "/api/auth/users",
        json={"username": "bb-owner", "password": "owner-pass-1", "name": "Owner One"},
    )
    assert created.status_code == 201

    response = client.get("/api/users")
    assert response.status_code == 200
    options = response.json()
    assert isinstance(options, list)
    entry = next(o for o in options if o["id"] == created.json()["id"])
    assert entry == {"id": created.json()["id"], "name": "Owner One"}
    # Every option exposes only id + display name.
    assert all(set(o.keys()) == {"id", "name"} for o in options)

    # Reachable by a non-admin user too.
    with httpx.Client(base_url=base_url, timeout=10.0) as http:
        http.post(
            "/api/auth/login",
            json={"username": "bb-owner", "password": "owner-pass-1"},
        )
        assert http.get("/api/users").status_code == 200


def test_admin_updates_user_profile(client: httpx.Client) -> None:
    created = client.post(
        "/api/auth/users",
        json={"username": "bb-editable", "password": "editable-pass-1"},
    )
    assert created.status_code == 201
    user_id = created.json()["id"]

    updated = client.put(
        f"/api/auth/users/{user_id}",
        json={
            "name": "Edited",
            "surname": "Name",
            "ppk_employee_rate": 1.5,
            "ppk_employer_rate": 1,
        },
    )
    assert updated.status_code == 200
    body = updated.json()
    assert body["name"] == "Edited"
    assert body["surname"] == "Name"
    assert body["ppk_employee_rate"] == "1.50"

    assert client.put("/api/auth/users/999999", json={}).status_code == 404
