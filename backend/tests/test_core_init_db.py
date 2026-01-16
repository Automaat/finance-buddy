from unittest.mock import MagicMock, patch

from app.core.init_db import init_db


def test_init_db_creates_tables():
    """Test that init_db calls create_all on metadata."""
    with patch("app.core.init_db.Base") as mock_base, patch("app.core.init_db.engine") as mock_engine:
        mock_metadata = MagicMock()
        mock_base.metadata = mock_metadata

        init_db()

        mock_metadata.create_all.assert_called_once_with(bind=mock_engine)


def test_init_db_imports_models():
    """Test that init_db module imports all models."""
    # Import the module to ensure all models are registered
    import app.core.init_db  # noqa: F401

    from app.core.database import Base

    # Verify models are registered in Base metadata
    table_names = Base.metadata.tables.keys()
    assert "accounts" in table_names
    assert "goals" in table_names
    assert "snapshots" in table_names
    assert "snapshot_values" in table_names
