from unittest.mock import patch

from fastapi.testclient import TestClient

from app.main import app

client = TestClient(app)


def test_health_check():
    response = client.get("/health")
    assert response.status_code == 200
    assert response.json() == {"status": "ok"}


def test_lifespan_initializes_db():
    with patch("app.main.init_db") as mock_init:
        with TestClient(app):
            pass  # Triggers lifespan startup
        mock_init.assert_called_once()
