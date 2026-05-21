"""Deterministic seed for black-box tests.

Writes a small fixture set directly via SQL so the seed is language-independent
(both Python and Go backends read the same Postgres). Tests assume this seed
exists; mutation tests must create new rows with unique names and clean up
after themselves.
"""

from __future__ import annotations

from contextlib import contextmanager
from datetime import date, datetime
from decimal import Decimal
from typing import TYPE_CHECKING

import psycopg2

if TYPE_CHECKING:
    from collections.abc import Iterator


# Fixture identities — fixed values so tests can reference them.
PERSONA_MARCIN = "Marcin"
PERSONA_EWA = "Ewa"
CONFIG_BIRTH_DATE = date(1990, 6, 15)
CONFIG_RETIREMENT_AGE = 65
CONFIG_RETIREMENT_MONTHLY_SALARY = Decimal("8000.00")
CONFIG_MONTHLY_EXPENSES = Decimal("5000.00")
CONFIG_MONTHLY_MORTGAGE_PAYMENT = Decimal("2000.00")


@contextmanager
def _connect(dsn: str) -> Iterator[psycopg2.extensions.connection]:
    conn = psycopg2.connect(dsn)
    try:
        yield conn
        conn.commit()
    except Exception:
        conn.rollback()
        raise
    finally:
        conn.close()


def _truncate_all(cur: psycopg2.extensions.cursor) -> None:
    """Reset every table the bb-tests touch. Keeps seed idempotent across runs."""
    cur.execute(
        """
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = 'public' AND tablename <> 'alembic_version'
        """
    )
    tables = [row[0] for row in cur.fetchall()]
    if not tables:
        return
    quoted = ", ".join(f'"{name}"' for name in tables)
    cur.execute(f"TRUNCATE {quoted} RESTART IDENTITY CASCADE")


def _seed_personas(cur: psycopg2.extensions.cursor) -> None:
    now = datetime(2026, 1, 1, 0, 0, 0)
    cur.executemany(
        """
        INSERT INTO personas (name, ppk_employee_rate, ppk_employer_rate, created_at)
        VALUES (%s, %s, %s, %s)
        """,
        [
            (PERSONA_MARCIN, Decimal("2.0"), Decimal("1.5"), now),
            (PERSONA_EWA, Decimal("2.0"), Decimal("1.5"), now),
        ],
    )


def _seed_config(cur: psycopg2.extensions.cursor) -> None:
    cur.execute(
        """
        INSERT INTO app_config (
            id, birth_date, retirement_age, retirement_monthly_salary,
            allocation_real_estate, allocation_stocks, allocation_bonds,
            allocation_gold, allocation_commodities,
            monthly_expenses, monthly_mortgage_payment
        )
        VALUES (1, %s, %s, %s, 40, 30, 20, 5, 5, %s, %s)
        """,
        (
            CONFIG_BIRTH_DATE,
            CONFIG_RETIREMENT_AGE,
            CONFIG_RETIREMENT_MONTHLY_SALARY,
            CONFIG_MONTHLY_EXPENSES,
            CONFIG_MONTHLY_MORTGAGE_PAYMENT,
        ),
    )


def seed(dsn: str) -> None:
    """Apply the deterministic seed. Idempotent — truncates first."""
    with _connect(dsn) as conn, conn.cursor() as cur:
        _truncate_all(cur)
        _seed_personas(cur)
        _seed_config(cur)
