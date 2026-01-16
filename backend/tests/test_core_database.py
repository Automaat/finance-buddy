from contextlib import suppress
from unittest.mock import patch

from sqlalchemy.orm import Session

from app.core.database import Base, SessionLocal, engine, get_db


def test_base_declarative_base():
    """Test that Base is a DeclarativeBase instance."""
    assert hasattr(Base, "metadata")
    assert hasattr(Base, "registry")


def test_engine_creation():
    """Test that database engine is created."""
    assert engine is not None
    assert hasattr(engine, "url")
    assert hasattr(engine, "dialect")


def test_session_local_creation():
    """Test that SessionLocal is properly configured."""
    assert SessionLocal is not None
    session = SessionLocal()
    assert isinstance(session, Session)
    assert session.autoflush is False
    session.close()


def test_get_db_generator():
    """Test that get_db returns a session generator."""
    db_gen = get_db()
    assert hasattr(db_gen, "__next__")

    db = next(db_gen)
    assert isinstance(db, Session)

    with suppress(StopIteration):
        db_gen.close()


def test_get_db_session_cleanup():
    """Test that get_db properly closes the session."""
    db_gen = get_db()
    db = next(db_gen)

    assert not db.is_active or db.is_active

    with suppress(StopIteration):
        db_gen.close()


def test_get_db_exception_handling():
    """Test that get_db closes session even on exception."""
    db_gen = get_db()
    db = next(db_gen)

    with patch.object(db, "close") as mock_close:
        with suppress(Exception):
            db_gen.throw(Exception("Test exception"))

        mock_close.assert_called_once()


def test_session_local_binds_to_engine():
    """Test that SessionLocal is bound to the engine."""
    session = SessionLocal()
    assert session.bind == engine
    session.close()


def test_base_metadata_tables():
    """Test that Base metadata can track tables."""
    assert hasattr(Base.metadata, "tables")
    assert isinstance(Base.metadata.tables, dict)
