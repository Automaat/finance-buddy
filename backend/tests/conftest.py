from collections.abc import Generator

import pytest
from sqlalchemy import create_engine, event
from sqlalchemy.orm import Session, sessionmaker
from testcontainers.postgres import PostgresContainer

from app.core.database import Base


@pytest.fixture(scope="function")
def test_db_engine():
    """Create in-memory SQLite engine for testing."""
    engine = create_engine("sqlite:///:memory:", echo=False)

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


@pytest.fixture(scope="function")
def test_db_engine_postgres(postgres_container):
    """Create PostgreSQL engine for testing using testcontainers."""
    engine = create_engine(postgres_container.get_connection_url(), echo=False)
    Base.metadata.create_all(engine)
    yield engine
    Base.metadata.drop_all(engine)
    engine.dispose()


@pytest.fixture(scope="function")
def test_db_session_postgres(test_db_engine_postgres) -> Generator[Session]:
    """Create test database session using PostgreSQL."""
    test_session_local = sessionmaker(autocommit=False, autoflush=False, bind=test_db_engine_postgres)
    session = test_session_local()
    try:
        yield session
    finally:
        session.rollback()
        session.close()
