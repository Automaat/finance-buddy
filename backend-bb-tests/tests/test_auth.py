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
