"""Deterministic seed for black-box tests.

Writes a small fixture set directly via SQL so the seed is language-independent
(both Python and Go backends read the same Postgres). Tests assume this seed
exists; mutation tests must create new rows with unique names and clean up
after themselves.
"""

from __future__ import annotations

import os
from contextlib import contextmanager
from datetime import UTC, date, datetime
from decimal import Decimal
from typing import TYPE_CHECKING
from urllib.parse import urlparse

import psycopg2

if TYPE_CHECKING:
    from collections.abc import Iterator


# Fixture identities — fixed values so tests can reference them.
PERSONA_MARCIN = "Marcin"
PERSONA_EWA = "Ewa"
PERSONA_SHARED = "Shared"
CONFIG_BIRTH_DATE = date(1990, 6, 15)
CONFIG_RETIREMENT_AGE = 65
CONFIG_RETIREMENT_MONTHLY_SALARY = Decimal("8000.00")
CONFIG_MONTHLY_EXPENSES = Decimal("5000.00")
CONFIG_MONTHLY_MORTGAGE_PAYMENT = Decimal("2000.00")

# Stable account names — tests look these up by name to resolve auto-assigned ids.
ACCOUNT_MARCIN_BANK = "Marcin Checking"
ACCOUNT_EWA_BANK = "Ewa Checking"
ACCOUNT_MARCIN_IKE = "Marcin IKE"
ACCOUNT_MARCIN_PPK = "Marcin PPK"
ACCOUNT_MARCIN_MORTGAGE = "Marcin Mortgage"
ACCOUNT_SHARED_REAL_ESTATE = "Shared Apartment Real Estate"

# Stable asset / aggregate identities.
ASSET_MARCIN_APARTMENT = "Marcin Apartment"

# Stable snapshot dates — month-end so backend aggregation buckets cleanly.
SNAPSHOT_DATES = (
    date(2025, 11, 30),
    date(2025, 12, 31),
    date(2026, 1, 31),
)

# Companies used across compensation fixtures.
COMPANY_MARCIN_EMPLOYER = "Acme Sp. z o.o."

# Created-at marker used everywhere so equality assertions are stable.
SEED_CREATED_AT = datetime(2026, 1, 1, 12, 0, 0)

# Allow-list of tables the seed manages. _truncate_seeded refuses to wipe anything
# outside this set — protects against a stray BB_DATABASE_URL pointed at a
# non-test database.
SEEDED_TABLES: frozenset[str] = frozenset(
    {
        "accounts",
        "app_config",
        "assets",
        "bonus_events",
        "company_valuations",
        "cpi_index",
        "debt_payments",
        "debts",
        "equity_grants",
        "fx_rates",
        "goals",
        "retirement_limits",
        "salary_records",
        "snapshot_aggregates",
        "snapshot_values",
        "snapshots",
        "transactions",
    }
)

LOCAL_HOSTS: frozenset[str] = frozenset({"localhost", "127.0.0.1", "::1", "postgres", "bb"})


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


def _assert_safe_to_truncate(dsn: str) -> None:
    """Refuse to truncate against anything that doesn't look like a throwaway DB.

    Trips when:
    - the DSN's host isn't in LOCAL_HOSTS, and
    - BB_ALLOW_DESTRUCTIVE_SEED isn't set truthy.

    Use the env var only when you know the target DB is disposable (e.g. CI).
    """
    host = (urlparse(dsn).hostname or "").lower()
    if host in LOCAL_HOSTS:
        return
    if os.environ.get("BB_ALLOW_DESTRUCTIVE_SEED", "").lower() in {"1", "true", "yes", "on"}:
        return
    raise RuntimeError(
        f"Refusing to truncate against host {host!r}. "
        "Point BB_DATABASE_URL at a local/throwaway DB, or set "
        "BB_ALLOW_DESTRUCTIVE_SEED=1 if you accept the risk."
    )


def _truncate_seeded(cur: psycopg2.extensions.cursor) -> None:
    """Truncate only tables on the seed's allow-list. Keeps seed idempotent."""
    cur.execute(
        """
        SELECT tablename
        FROM pg_tables
        WHERE schemaname = 'public'
        """
    )
    existing = {row[0] for row in cur.fetchall()}
    to_truncate = sorted(SEEDED_TABLES & existing)
    if not to_truncate:
        return
    quoted = ", ".join(f'"{name}"' for name in to_truncate)
    cur.execute(f"TRUNCATE {quoted} RESTART IDENTITY CASCADE")


# Sub-select resolving an owner name to a users.id. A name with no matching
# user (notably "Shared") yields NULL — the jointly-owned bucket.
_OWNER_ID_SQL = "(SELECT id FROM users WHERE name = %s)"


def _seed_users(cur: psycopg2.extensions.cursor) -> None:
    """Insert Marcin/Ewa as users so owner_user_id can reference them.

    PPK rates live on the users table now that owner-aware code resolves
    them via owner_user_id instead of the personas table.

    The users table is not truncated (it holds the backend-seeded admin),
    so the insert is idempotent on username.
    """
    cur.executemany(
        """
        INSERT INTO users (
            username, password_hash, name, is_admin,
            ppk_employee_rate, ppk_employer_rate, created_at
        )
        VALUES (%s, %s, %s, FALSE, %s, %s, %s)
        ON CONFLICT (username) DO UPDATE SET
            name = EXCLUDED.name,
            ppk_employee_rate = EXCLUDED.ppk_employee_rate,
            ppk_employer_rate = EXCLUDED.ppk_employer_rate
        """,
        [
            (
                "marcin",
                "!seed-no-login",
                PERSONA_MARCIN,
                Decimal("2.0"),
                Decimal("1.5"),
                SEED_CREATED_AT,
            ),
            (
                "ewa",
                "!seed-no-login",
                PERSONA_EWA,
                Decimal("2.0"),
                Decimal("1.5"),
                SEED_CREATED_AT,
            ),
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


def _seed_accounts(cur: psycopg2.extensions.cursor) -> dict[str, int]:
    """Insert seeded accounts; return a name → id map for downstream FKs."""
    rows = [
        # name, type, category, owner name, currency, wrapper, purpose, sqm, receives_contrib
        (
            ACCOUNT_MARCIN_BANK,
            "asset",
            "bank",
            PERSONA_MARCIN,
            "PLN",
            None,
            "general",
            None,
            True,
        ),
        (
            ACCOUNT_EWA_BANK,
            "asset",
            "bank",
            PERSONA_EWA,
            "PLN",
            None,
            "general",
            None,
            True,
        ),
        (
            ACCOUNT_MARCIN_IKE,
            "asset",
            "stock",
            PERSONA_MARCIN,
            "PLN",
            "IKE",
            "retirement",
            None,
            True,
        ),
        (
            ACCOUNT_MARCIN_PPK,
            "asset",
            "ppk",
            PERSONA_MARCIN,
            "PLN",
            "PPK",
            "retirement",
            None,
            True,
        ),
        (
            ACCOUNT_MARCIN_MORTGAGE,
            "liability",
            "mortgage",
            PERSONA_MARCIN,
            "PLN",
            None,
            "general",
            None,
            False,
        ),
        (
            ACCOUNT_SHARED_REAL_ESTATE,
            "asset",
            "real_estate",
            PERSONA_SHARED,
            "PLN",
            None,
            "general",
            Decimal("65.50"),
            False,
        ),
    ]
    ids: dict[str, int] = {}
    for row in rows:
        cur.execute(
            f"""
            INSERT INTO accounts (
                name, type, category, owner_user_id, currency, account_wrapper, purpose,
                square_meters, is_active, receives_contributions, created_at
            )
            VALUES (%s, %s, %s, {_OWNER_ID_SQL}, %s, %s, %s, %s, TRUE, %s, %s)
            RETURNING id
            """,
            (*row, SEED_CREATED_AT),
        )
        ids[row[0]] = cur.fetchone()[0]
    return ids


def _seed_assets(cur: psycopg2.extensions.cursor) -> dict[str, int]:
    cur.execute(
        """
        INSERT INTO assets (name, is_active, created_at)
        VALUES (%s, TRUE, %s)
        RETURNING id
        """,
        (ASSET_MARCIN_APARTMENT, SEED_CREATED_AT),
    )
    return {ASSET_MARCIN_APARTMENT: cur.fetchone()[0]}


def _seed_snapshots(cur: psycopg2.extensions.cursor) -> dict[date, int]:
    ids: dict[date, int] = {}
    for snap_date in SNAPSHOT_DATES:
        cur.execute(
            """
            INSERT INTO snapshots (date, notes, created_at)
            VALUES (%s, %s, %s)
            RETURNING id
            """,
            (snap_date, f"Seed snapshot {snap_date.isoformat()}", SEED_CREATED_AT),
        )
        ids[snap_date] = cur.fetchone()[0]
    return ids


def _seed_snapshot_values(
    cur: psycopg2.extensions.cursor,
    snapshot_ids: dict[date, int],
    account_ids: dict[str, int],
    asset_ids: dict[str, int],
) -> int:
    """Three months of values per account + asset. Trend rises gently for assets,
    mortgage principal shrinks. Returns row count for logging.
    """
    # Per-snapshot values (PLN). Ordering: [snap0, snap1, snap2] = Nov/Dec/Jan.
    by_account: dict[str, tuple[Decimal, Decimal, Decimal]] = {
        ACCOUNT_MARCIN_BANK: (Decimal("25000.00"), Decimal("27500.00"), Decimal("30000.00")),
        ACCOUNT_EWA_BANK: (Decimal("18000.00"), Decimal("19200.00"), Decimal("20500.00")),
        ACCOUNT_MARCIN_IKE: (Decimal("42000.00"), Decimal("44500.00"), Decimal("46100.00")),
        ACCOUNT_MARCIN_PPK: (Decimal("12000.00"), Decimal("12800.00"), Decimal("13600.00")),
        ACCOUNT_MARCIN_MORTGAGE: (
            Decimal("280000.00"),
            Decimal("278500.00"),
            Decimal("277000.00"),
        ),
        ACCOUNT_SHARED_REAL_ESTATE: (
            Decimal("650000.00"),
            Decimal("655000.00"),
            Decimal("660000.00"),
        ),
    }
    by_asset: dict[str, tuple[Decimal, Decimal, Decimal]] = {
        ASSET_MARCIN_APARTMENT: (Decimal("420000.00"), Decimal("422500.00"), Decimal("425000.00")),
    }

    count = 0
    snap_dates = list(SNAPSHOT_DATES)
    for name, values in by_account.items():
        account_id = account_ids[name]
        for snap_date, value in zip(snap_dates, values, strict=True):
            cur.execute(
                """
                INSERT INTO snapshot_values (snapshot_id, account_id, asset_id, value)
                VALUES (%s, %s, NULL, %s)
                """,
                (snapshot_ids[snap_date], account_id, value),
            )
            count += 1
    for name, values in by_asset.items():
        asset_id = asset_ids[name]
        for snap_date, value in zip(snap_dates, values, strict=True):
            cur.execute(
                """
                INSERT INTO snapshot_values (snapshot_id, account_id, asset_id, value)
                VALUES (%s, NULL, %s, %s)
                """,
                (snapshot_ids[snap_date], asset_id, value),
            )
            count += 1
    return count


def _seed_transactions(
    cur: psycopg2.extensions.cursor,
    account_ids: dict[str, int],
) -> None:
    rows = [
        (
            account_ids[ACCOUNT_MARCIN_IKE],
            Decimal("1500.00"),
            date(2025, 12, 5),
            PERSONA_MARCIN,
            "employee",
        ),
        (
            account_ids[ACCOUNT_MARCIN_PPK],
            Decimal("400.00"),
            date(2025, 12, 28),
            PERSONA_MARCIN,
            "employer",
        ),
        (
            account_ids[ACCOUNT_MARCIN_PPK],
            Decimal("250.00"),
            date(2026, 1, 15),
            PERSONA_MARCIN,
            "government",
        ),
    ]
    cur.executemany(
        f"""
        INSERT INTO transactions (
            account_id, amount, date, owner_user_id, transaction_type, is_active, created_at
        )
        VALUES (%s, %s, %s, {_OWNER_ID_SQL}, %s, TRUE, %s)
        """,
        [(*r, SEED_CREATED_AT) for r in rows],
    )


def _seed_bonus_events(cur: psycopg2.extensions.cursor) -> None:
    rows = [
        # PLN annual bonus
        (
            date(2025, 12, 20),
            Decimal("15000.00"),
            "PLN",
            "annual",
            COMPANY_MARCIN_EMPLOYER,
            PERSONA_MARCIN,
            "UOP",
            "2025 annual performance bonus",
        ),
        # USD sign-on bonus
        (
            date(2025, 7, 1),
            Decimal("5000.00"),
            "USD",
            "signon",
            COMPANY_MARCIN_EMPLOYER,
            PERSONA_MARCIN,
            "B2B",
            "Joining USD signon",
        ),
    ]
    cur.executemany(
        f"""
        INSERT INTO bonus_events (
            date, amount, currency, type, company, owner_user_id, contract_type,
            notes, is_active, created_at
        )
        VALUES (%s, %s, %s, %s, %s, {_OWNER_ID_SQL}, %s, %s, TRUE, %s)
        """,
        [(*r, SEED_CREATED_AT) for r in rows],
    )


def _seed_company_valuations(cur: psycopg2.extensions.cursor) -> None:
    cur.execute(
        """
        INSERT INTO company_valuations (
            company, date, currency, fmv_per_share, fmv_low, fmv_high,
            source, common_stock_discount_pct, notes, is_active, created_at
        )
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, TRUE, %s)
        """,
        (
            COMPANY_MARCIN_EMPLOYER,
            date(2025, 6, 30),
            "USD",
            Decimal("12.5000"),
            Decimal("11.0000"),
            Decimal("14.0000"),
            "409a",
            Decimal("25.00"),
            "Mid-2025 409A appraisal",
            SEED_CREATED_AT,
        ),
    )


def _seed_equity_grants(cur: psycopg2.extensions.cursor) -> None:
    cur.execute(
        f"""
        INSERT INTO equity_grants (
            grant_date, type, company, owner_user_id, total_shares, strike_price, currency,
            vest_start_date, vest_cliff_months, vest_total_months, vest_frequency,
            vest_custom_schedule, requires_liquidity_event, liquidity_event_date,
            tax_treatment, notes, is_active, created_at
        )
        VALUES (%s, %s, %s, {_OWNER_ID_SQL}, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, TRUE, %s)
        """,
        (
            date(2024, 1, 15),
            "rsu",
            COMPANY_MARCIN_EMPLOYER,
            PERSONA_MARCIN,
            4800,
            None,
            "USD",
            date(2024, 1, 15),
            12,
            48,
            "monthly",
            None,
            False,
            None,
            "capital_gains_19",
            "4-year RSU grant, monthly vest after 1-yr cliff",
            SEED_CREATED_AT,
        ),
    )


def _seed_fx_rates(cur: psycopg2.extensions.cursor) -> None:
    # USD rows on the company-valuation date (2025-06-30) and the USD sign-on
    # bonus date (2025-07-01) keep the equity-grant + bonus FX lookups as pure
    # cache hits — without them every list request would miss the cache and
    # block on a synchronous NBP fetch.
    rows = [
        (date(2025, 6, 30), "USD", Decimal("3.720000")),
        (date(2025, 7, 1), "USD", Decimal("3.730000")),
        (date(2026, 1, 31), "USD", Decimal("4.150000")),
        (date(2026, 1, 31), "EUR", Decimal("4.350000")),
    ]
    cur.executemany(
        """
        INSERT INTO fx_rates (date, currency, rate_pln, created_at)
        VALUES (%s, %s, %s, %s)
        """,
        [(*r, SEED_CREATED_AT) for r in rows],
    )


def _seed_cpi_index(cur: psycopg2.extensions.cursor) -> None:
    # Full GUS-BDL Polish CPI history (variable 217230). Real values so the
    # /api/cpi/series endpoint returns the same numbers the production
    # backend would, fully offline. fetched_at must be "fresh" enough that
    # the backend's startup-staleness check (services/inflation.needs_refresh,
    # 7-day threshold) doesn't trigger an external GUS BDL refresh.
    fetched_at = datetime.now(UTC).replace(tzinfo=None)
    rows = [
        (2003, Decimal("100.8"), "GUS-BDL-217230"),
        (2004, Decimal("103.5"), "GUS-BDL-217230"),
        (2005, Decimal("102.1"), "GUS-BDL-217230"),
        (2006, Decimal("101.0"), "GUS-BDL-217230"),
        (2007, Decimal("102.5"), "GUS-BDL-217230"),
        (2008, Decimal("104.2"), "GUS-BDL-217230"),
        (2009, Decimal("103.5"), "GUS-BDL-217230"),
        (2010, Decimal("102.6"), "GUS-BDL-217230"),
        (2011, Decimal("104.3"), "GUS-BDL-217230"),
        (2012, Decimal("103.7"), "GUS-BDL-217230"),
        (2013, Decimal("100.9"), "GUS-BDL-217230"),
        (2014, Decimal("100.0"), "GUS-BDL-217230"),
        (2015, Decimal("99.1"), "GUS-BDL-217230"),
        (2016, Decimal("99.4"), "GUS-BDL-217230"),
        (2017, Decimal("102.0"), "GUS-BDL-217230"),
        (2018, Decimal("101.6"), "GUS-BDL-217230"),
        (2019, Decimal("102.3"), "GUS-BDL-217230"),
        (2020, Decimal("103.4"), "GUS-BDL-217230"),
        (2021, Decimal("105.1"), "GUS-BDL-217230"),
        (2022, Decimal("114.4"), "GUS-BDL-217230"),
        (2023, Decimal("111.4"), "GUS-BDL-217230"),
        (2024, Decimal("103.6"), "GUS-BDL-217230"),
        (2025, Decimal("103.6"), "GUS-BDL-217230"),
    ]
    cur.executemany(
        """
        INSERT INTO cpi_index (year, yoy_rate, source, fetched_at)
        VALUES (%s, %s, %s, %s)
        """,
        [(*r, fetched_at) for r in rows],
    )


def _seed_debts(
    cur: psycopg2.extensions.cursor,
    account_ids: dict[str, int],
) -> int:
    cur.execute(
        """
        INSERT INTO debts (
            account_id, name, debt_type, start_date, initial_amount,
            interest_rate, currency, notes, is_active, created_at
        )
        VALUES (%s, %s, %s, %s, %s, %s, %s, %s, TRUE, %s)
        RETURNING id
        """,
        (
            account_ids[ACCOUNT_MARCIN_MORTGAGE],
            "Apartment Mortgage",
            "mortgage",
            date(2022, 6, 1),
            Decimal("320000.00"),
            Decimal("7.25"),
            "PLN",
            "Bank Pekao 30-year mortgage",
            SEED_CREATED_AT,
        ),
    )
    return cur.fetchone()[0]


def _seed_debt_payments(
    cur: psycopg2.extensions.cursor,
    account_ids: dict[str, int],
) -> None:
    rows = [
        (
            account_ids[ACCOUNT_MARCIN_MORTGAGE],
            Decimal("2000.00"),
            date(2025, 12, 1),
            PERSONA_MARCIN,
        ),
        (
            account_ids[ACCOUNT_MARCIN_MORTGAGE],
            Decimal("2000.00"),
            date(2026, 1, 1),
            PERSONA_MARCIN,
        ),
    ]
    cur.executemany(
        f"""
        INSERT INTO debt_payments (account_id, amount, date, owner_user_id, is_active, created_at)
        VALUES (%s, %s, %s, {_OWNER_ID_SQL}, TRUE, %s)
        """,
        [(*r, SEED_CREATED_AT) for r in rows],
    )


def _seed_salary_records(cur: psycopg2.extensions.cursor) -> None:
    rows = [
        (date(2025, 1, 31), Decimal("18000.00"), "UOP", COMPANY_MARCIN_EMPLOYER, PERSONA_MARCIN),
        (date(2025, 6, 30), Decimal("19000.00"), "UOP", COMPANY_MARCIN_EMPLOYER, PERSONA_MARCIN),
        (date(2026, 1, 31), Decimal("21000.00"), "UOP", COMPANY_MARCIN_EMPLOYER, PERSONA_MARCIN),
    ]
    cur.executemany(
        f"""
        INSERT INTO salary_records (
            date, gross_amount, contract_type, company, owner_user_id, is_active, created_at
        )
        VALUES (%s, %s, %s, %s, {_OWNER_ID_SQL}, TRUE, %s)
        """,
        [(*r, SEED_CREATED_AT) for r in rows],
    )


def _seed_goals(
    cur: psycopg2.extensions.cursor,
    account_ids: dict[str, int],
) -> None:
    cur.execute(
        """
        INSERT INTO goals (
            name, target_amount, target_date, current_amount, monthly_contribution,
            is_completed, account_id, category, created_at
        )
        VALUES (%s, %s, %s, %s, %s, FALSE, %s, %s, %s)
        """,
        (
            "Emergency fund",
            Decimal("60000.00"),
            date(2026, 12, 31),
            Decimal("30000.00"),
            Decimal("2500.00"),
            account_ids[ACCOUNT_MARCIN_BANK],
            # NOTE: Goal.category is a String(100) at the DB layer but the
            # GoalResponse schema reads it back as `Category | None`. Pydantic
            # raises on read if it's a non-Category value (e.g. a Purpose like
            # "emergency_fund"). Leave NULL here — a seeded category isn't
            # load-bearing for any current test.
            None,
            SEED_CREATED_AT,
        ),
    )


def _seed_retirement_limits(cur: psycopg2.extensions.cursor) -> None:
    rows = [
        (2025, "IKE", PERSONA_MARCIN, Decimal("23472.00"), "2025 IKE limit"),
        (2025, "IKZE", PERSONA_MARCIN, Decimal("9388.80"), "2025 IKZE limit"),
    ]
    cur.executemany(
        f"""
        INSERT INTO retirement_limits (
            year, account_wrapper, owner_user_id, limit_amount, notes
        )
        VALUES (%s, %s, {_OWNER_ID_SQL}, %s, %s)
        """,
        rows,
    )


def seed(dsn: str) -> None:
    """Apply the deterministic seed. Idempotent — truncates first."""
    _assert_safe_to_truncate(dsn)
    with _connect(dsn) as conn, conn.cursor() as cur:
        _truncate_seeded(cur)
        _seed_users(cur)
        _seed_config(cur)
        account_ids = _seed_accounts(cur)
        asset_ids = _seed_assets(cur)
        snapshot_ids = _seed_snapshots(cur)
        _seed_snapshot_values(cur, snapshot_ids, account_ids, asset_ids)
        _seed_transactions(cur, account_ids)
        _seed_bonus_events(cur)
        _seed_company_valuations(cur)
        _seed_equity_grants(cur)
        _seed_fx_rates(cur)
        _seed_cpi_index(cur)
        _seed_debts(cur, account_ids)
        _seed_debt_payments(cur, account_ids)
        _seed_salary_records(cur)
        _seed_goals(cur, account_ids)
        _seed_retirement_limits(cur)
