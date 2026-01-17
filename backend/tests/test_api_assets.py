def test_create_asset_missing_name(test_client):
    """Test POST /api/assets with missing name field"""
    payload = {}

    response = test_client.post("/api/assets", json=payload)

    assert response.status_code == 422  # Unprocessable Entity
