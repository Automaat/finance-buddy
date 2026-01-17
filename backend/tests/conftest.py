from collections.abc import Generator
from unittest.mock import patch

import pytest
from fastapi.testclient import TestClient
from sqlalchemy import create_engine, event
from sqlalchemy.orm import Session, sessionmaker
from testcontainers.postgres import PostgresContainer

from app.core.database import Base, get_db
from app.main import app

# Import models BEFORE app to register them with SQLAlchemy Base.metadata
from app.models import Account, Asset, Goal, Snapshot, SnapshotValue, Transaction  # noqa: F401


@pytest.fixture(scope="function")
def test_db_engine():
    """Create in-memory SQLite engine for testing with shared cache."""
    # Use file:memdb?mode=memory&cache=shared to allow multiple connections
    # to share the same in-memory database
    engine = create_engine(
        "sqlite:///file:memdb?mode=memory&cache=shared&uri=true",
        echo=False,
        connect_args={"check_same_thread": False},
    )

    # Enable foreign key constraints in SQLite
    @event.listens_for(engine, "connect")
    def set_sqlite_pragma(dbapi_conn, _connection_record):
        cursor = dbapi_conn.cursor()
        cursor.execute("PRAGMA foreign_keys=ON")
        cursor.close()

    Base.metadata.create_all(engine)
    yield engine
    Base.metadata.drop_all(engine)
    engine.dispose()


@pytest.fixture(scope="function")
def test_db_session(test_db_engine) -> Generator[Session]:
    """Create test database session."""
    test_session_local = sessionmaker(autocommit=False, autoflush=False, bind=test_db_engine)
    session = test_session_local()
    try:
        yield session
    finally:
        session.rollback()
        session.close()


@pytest.fixture(scope="session")
def postgres_container():
    """Create PostgreSQL testcontainer for integration testing."""
    with PostgresContainer("postgres:18") as postgres:
        yield postgres


@pytest.fixture(scope="session")
def test_db_engine_postgres(postgres_container):
    """Create PostgreSQL engine for testing using testcontainers."""
    engine = create_engine(postgres_container.get_connection_url(), echo=False)
    Base.metadata.create_all(engine)
    yield engine
    engine.dispose()


@pytest.fixture(scope="function")
def test_db_session_postgres(test_db_engine_postgres) -> Generator[Session]:
    """Create test database session using PostgreSQL."""
    test_session_local = sessionmaker(
        autocommit=False, autoflush=False, bind=test_db_engine_postgres
    )
    session = test_session_local()
    try:
        yield session
    finally:
        session.rollback()
        session.close()


@pytest.fixture(scope="function")
def test_client(test_db_engine) -> Generator[TestClient]:
    """Create test client with overridden database dependency."""
    # Create a new session for this test
    test_session_local = sessionmaker(autocommit=False, autoflush=False, bind=test_db_engine)

    def override_get_db():
        session = test_session_local()
        try:
            yield session
        finally:
            session.close()

    # Mock init_db to use test engine instead of production engine
    def mock_init_db():
        Base.metadata.create_all(bind=test_db_engine)

    app.dependency_overrides[get_db] = override_get_db

    # Patch init_db to use test engine during lifespan
    with patch("app.core.init_db.init_db", side_effect=mock_init_db):
        client = TestClient(app, raise_server_exceptions=True)
        try:
            yield client
        finally:
            app.dependency_overrides.clear()
