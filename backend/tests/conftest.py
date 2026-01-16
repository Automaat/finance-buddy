from collections.abc import Generator

import pytest
from fastapi.testclient import TestClient
from sqlalchemy import create_engine, event
from sqlalchemy.orm import Session, sessionmaker
from testcontainers.postgres import PostgresContainer

from app.core.database import Base, get_db
from app.main import app


@pytest.fixture(scope="function")
def test_db_engine():
    """Create in-memory SQLite engine for testing."""
    engine = create_engine(
        "sqlite:///:memory:", echo=False, connect_args={"check_same_thread": False}
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
def test_client(test_db_session) -> Generator[TestClient]:
    """Create test client with overridden database dependency."""

    def override_get_db():
        try:
            yield test_db_session
        finally:
            pass

    app.dependency_overrides[get_db] = override_get_db
    client = TestClient(app)
    yield client
    app.dependency_overrides.clear()
