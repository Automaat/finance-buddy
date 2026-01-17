from unittest.mock import MagicMock, patch

import app.core.init_db  # noqa: F401
from app.core.database import Base
from app.core.init_db import init_db


def test_init_db_creates_tables():
    """Test that init_db calls create_all on metadata."""
    with (
        patch("app.core.init_db.Base") as mock_base,
        patch("app.core.init_db.engine") as mock_engine,
        patch("app.core.init_db.SessionLocal") as mock_session_local,
    ):
        mock_metadata = MagicMock()
        mock_base.metadata = mock_metadata

        # Mock the session and query chain
        mock_session = MagicMock()
        mock_session_local.return_value = mock_session
        mock_session.query.return_value.count.return_value = 1  # Pretend limits exist

        init_db()

        mock_metadata.create_all.assert_called_once_with(bind=mock_engine)


def test_init_db_imports_models():
    """Test that init_db module imports all models."""

    # Verify models are registered in Base metadata
    table_names = Base.metadata.tables.keys()
    assert "accounts" in table_names
    assert "goals" in table_names
    assert "snapshots" in table_names
    assert "snapshot_values" in table_names
