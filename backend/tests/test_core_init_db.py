from unittest.mock import MagicMock, patch

from app.core.database import Base
from app.core.init_db import BASELINE_REVISION, _run_migrations, _seed_defaults, init_db
from app.models import Persona, RetirementLimit


def test_init_db_runs_migrations_then_seeds():
    """init_db brings the schema to head, then seeds defaults."""
    with (
        patch("app.core.init_db._run_migrations") as mock_migrations,
        patch("app.core.init_db._seed_defaults") as mock_seed,
    ):
        init_db()

        mock_migrations.assert_called_once_with()
        mock_seed.assert_called_once_with()


def test_run_migrations_upgrades_fresh_database():
    """A fresh database is upgraded straight to head without stamping."""
    with (
        patch("app.core.init_db.inspect") as mock_inspect,
        patch("app.core.init_db.command") as mock_command,
    ):
        mock_inspect.return_value.get_table_names.return_value = []

        _run_migrations()

        mock_command.stamp.assert_not_called()
        mock_command.upgrade.assert_called_once()
        assert mock_command.upgrade.call_args[0][1] == "head"


def test_run_migrations_stamps_pre_alembic_database():
    """A pre-Alembic database is stamped at the baseline before upgrading."""
    with (
        patch("app.core.init_db.inspect") as mock_inspect,
        patch("app.core.init_db.command") as mock_command,
    ):
        mock_inspect.return_value.get_table_names.return_value = ["accounts", "snapshots"]

        _run_migrations()

        mock_command.stamp.assert_called_once()
        assert mock_command.stamp.call_args[0][1] == BASELINE_REVISION
        mock_command.upgrade.assert_called_once()


def test_run_migrations_skips_stamp_when_alembic_version_present():
    """An Alembic-managed database is upgraded without re-stamping."""
    with (
        patch("app.core.init_db.inspect") as mock_inspect,
        patch("app.core.init_db.command") as mock_command,
    ):
        mock_inspect.return_value.get_table_names.return_value = ["accounts", "alembic_version"]

        _run_migrations()

        mock_command.stamp.assert_not_called()
        mock_command.upgrade.assert_called_once()


def test_models_register_on_metadata():
    """All model tables are registered on Base.metadata."""
    table_names = Base.metadata.tables.keys()
    assert "accounts" in table_names
    assert "goals" in table_names
    assert "snapshots" in table_names
    assert "snapshot_values" in table_names


def test_seed_defaults_inserts_personas_and_limits(test_db_session):
    """_seed_defaults populates personas and retirement limits on an empty DB."""
    test_db_session.close = MagicMock()  # keep the shared session open across the call
    with patch("app.core.init_db.SessionLocal", return_value=test_db_session):
        _seed_defaults()

    assert test_db_session.query(Persona).count() == 2
    assert test_db_session.query(RetirementLimit).count() == 4
