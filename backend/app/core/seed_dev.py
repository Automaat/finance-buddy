"""Demo data seeder for development.

Activated by setting ``SEED_DEV_DATA=true``. Idempotent: bails out if any
``Account`` rows already exist so a populated database survives restarts.
The seed builds a coherent ~24 months of household finance history so every
dashboard widget — net worth chart, allocation, goals, salaries, retirement,
inflation — has data to render.
"""

from __future__ import annotations

import logging
import math
import os
from datetime import UTC, date, datetime, timedelta
from decimal import Decimal

from sqlalchemy import func, select

from app.core.database import SessionLocal
from app.core.enums import (
    AccountType,
    Category,
    ContractType,
    DebtType,
    Purpose,
    TransactionType,
    Wrapper,
)
from app.models import (
    Account,
    AppConfig,
    Asset,
    CpiIndex,
    Debt,
    DebtPayment,
    Goal,
    SalaryRecord,
    Snapshot,
    SnapshotValue,
    Transaction,
)

logger = logging.getLogger(__name__)

SEED_ENV_VAR = "SEED_DEV_DATA"


def should_seed() -> bool:
    return os.getenv(SEED_ENV_VAR, "").strip().lower() in {"1", "true", "yes", "on"}


def _today() -> date:
    return datetime.now(UTC).date()


def _month_end(year: int, month: int) -> date:
    if month == 12:
        return date(year, 12, 31)
    return date(year, month + 1, 1) - timedelta(days=1)


def _months_back(reference: date, count: int) -> list[date]:
    """Last-day-of-month dates for the ``count`` months ending at ``reference``'s month."""
    months: list[date] = []
    year, month = reference.year, reference.month
    for _ in range(count):
        months.append(_month_end(year, month))
        month -= 1
        if month == 0:
            month = 12
            year -= 1
    months.reverse()
    return months


def _seed_app_config(db) -> None:
    if (db.scalar(select(func.count()).select_from(AppConfig)) or 0) > 0:
        return
    db.add(
        AppConfig(
            id=1,
            birth_date=date(1990, 6, 15),
            retirement_age=65,
            retirement_monthly_salary=Decimal("12000.00"),
            allocation_real_estate=30,
            allocation_stocks=40,
            allocation_bonds=15,
            allocation_gold=10,
            allocation_commodities=5,
            monthly_expenses=Decimal("9500.00"),
            monthly_mortgage_payment=Decimal("3200.00"),
        )
    )
    db.commit()


def _seed_cpi(db) -> None:
    if (db.scalar(select(func.count()).select_from(CpiIndex)) or 0) > 0:
        return
    # Polish CPI y/y rates (GUS published values, base = previous year = 100).
    rates = {
        2018: Decimal("101.6"),
        2019: Decimal("102.3"),
        2020: Decimal("103.4"),
        2021: Decimal("105.1"),
        2022: Decimal("114.4"),
        2023: Decimal("111.4"),
        2024: Decimal("103.6"),
        2025: Decimal("103.5"),
        2026: Decimal("103.2"),
    }
    db.add_all(
        CpiIndex(year=year, yoy_rate=rate, source="seed-dev", fetched_at=datetime.now(UTC))
        for year, rate in rates.items()
    )
    db.commit()


def _seed_accounts_and_assets(db) -> tuple[dict[str, Account], dict[str, Asset]]:
    accounts_spec = [
        # Banking
        ("Konto Marcin", AccountType.ASSET, Category.BANK, "Marcin", None, Purpose.GENERAL),
        ("Konto Ewa", AccountType.ASSET, Category.BANK, "Ewa", None, Purpose.GENERAL),
        (
            "Oszczędności Shared",
            AccountType.ASSET,
            Category.SAVING_ACCOUNT,
            "Shared",
            None,
            Purpose.EMERGENCY_FUND,
        ),
        # Retirement wrappers
        (
            "IKE Marcin",
            AccountType.ASSET,
            Category.STOCK,
            "Marcin",
            Wrapper.IKE,
            Purpose.RETIREMENT,
        ),
        ("IKE Ewa", AccountType.ASSET, Category.STOCK, "Ewa", Wrapper.IKE, Purpose.RETIREMENT),
        (
            "IKZE Marcin",
            AccountType.ASSET,
            Category.BOND,
            "Marcin",
            Wrapper.IKZE,
            Purpose.RETIREMENT,
        ),
        ("IKZE Ewa", AccountType.ASSET, Category.BOND, "Ewa", Wrapper.IKZE, Purpose.RETIREMENT),
        ("PPK Marcin", AccountType.ASSET, Category.PPK, "Marcin", Wrapper.PPK, Purpose.RETIREMENT),
        ("PPK Ewa", AccountType.ASSET, Category.PPK, "Ewa", Wrapper.PPK, Purpose.RETIREMENT),
        # Brokerage
        ("Akcje Shared", AccountType.ASSET, Category.STOCK, "Shared", None, Purpose.GENERAL),
        ("Obligacje Shared", AccountType.ASSET, Category.BOND, "Shared", None, Purpose.GENERAL),
        # Tangibles
        ("Mieszkanie", AccountType.ASSET, Category.REAL_ESTATE, "Shared", None, Purpose.GENERAL),
        ("Samochód", AccountType.ASSET, Category.VEHICLE, "Shared", None, Purpose.GENERAL),
        # Liabilities
        ("Hipoteka", AccountType.LIABILITY, Category.MORTGAGE, "Shared", None, Purpose.GENERAL),
        (
            "Raty 0% RTV",
            AccountType.LIABILITY,
            Category.INSTALLMENT,
            "Marcin",
            None,
            Purpose.GENERAL,
        ),
    ]

    accounts: dict[str, Account] = {}
    for name, type_, category, owner, wrapper, purpose in accounts_spec:
        square_meters = Decimal("62.50") if category == Category.REAL_ESTATE else None
        receives_contributions = type_ == AccountType.ASSET and category != Category.VEHICLE
        account = Account(
            name=name,
            type=type_.value,
            category=category.value,
            owner=owner,
            currency="PLN",
            account_wrapper=wrapper.value if wrapper else None,
            purpose=purpose.value,
            square_meters=square_meters,
            is_active=True,
            receives_contributions=receives_contributions,
        )
        db.add(account)
        accounts[name] = account

    assets = {
        "Złoto inwestycyjne": Asset(name="Złoto inwestycyjne", is_active=True),
        "Kryptowaluty": Asset(name="Kryptowaluty", is_active=True),
    }
    db.add_all(assets.values())

    db.commit()
    for account in accounts.values():
        db.refresh(account)
    for asset in assets.values():
        db.refresh(asset)
    return accounts, assets


def _seed_debts_and_payments(db, accounts: dict[str, Account]) -> None:
    mortgage = Debt(
        account_id=accounts["Hipoteka"].id,
        name="Hipoteka mieszkanie",
        debt_type=DebtType.MORTGAGE.value,
        start_date=date(2021, 9, 1),
        initial_amount=Decimal("420000.00"),
        interest_rate=Decimal("7.25"),
        currency="PLN",
        notes="Mortgage on the apartment",
        is_active=True,
    )
    installments = Debt(
        account_id=accounts["Raty 0% RTV"].id,
        name="Raty 0% pralka + lodówka",
        debt_type=DebtType.INSTALLMENT_0PERCENT.value,
        start_date=date(2025, 8, 15),
        initial_amount=Decimal("8400.00"),
        interest_rate=Decimal("0.00"),
        currency="PLN",
        notes="24-month 0% installment plan",
        is_active=True,
    )
    db.add_all([mortgage, installments])
    db.commit()

    today = _today()
    payment_months = _months_back(today, 24)
    payments: list[DebtPayment] = []
    for snapshot_date in payment_months:
        payments.append(
            DebtPayment(
                account_id=accounts["Hipoteka"].id,
                amount=Decimal("3200.00"),
                date=snapshot_date,
                owner="Shared",
                is_active=True,
            )
        )
    # Installments only since the start_date
    for snapshot_date in payment_months:
        if snapshot_date >= date(2025, 8, 1):
            payments.append(
                DebtPayment(
                    account_id=accounts["Raty 0% RTV"].id,
                    amount=Decimal("350.00"),
                    date=snapshot_date,
                    owner="Marcin",
                    is_active=True,
                )
            )
    db.add_all(payments)
    db.commit()


def _seed_goals(db) -> None:
    today = _today()
    db.add_all(
        [
            Goal(
                name="Fundusz awaryjny",
                target_amount=Decimal("60000.00"),
                target_date=date(today.year, 12, 31),
                current_amount=Decimal("42000.00"),
                monthly_contribution=Decimal("2000.00"),
                is_completed=False,
            ),
            Goal(
                name="Wakacje Japonia",
                target_amount=Decimal("25000.00"),
                target_date=date(today.year + 1, 5, 1),
                current_amount=Decimal("8000.00"),
                monthly_contribution=Decimal("1500.00"),
                is_completed=False,
            ),
            Goal(
                name="Wkład własny na działkę",
                target_amount=Decimal("120000.00"),
                target_date=date(today.year + 3, 6, 1),
                current_amount=Decimal("18000.00"),
                monthly_contribution=Decimal("2500.00"),
                is_completed=False,
            ),
            Goal(
                name="Nowy laptop",
                target_amount=Decimal("9000.00"),
                target_date=date(today.year - 1, 6, 1),
                current_amount=Decimal("9000.00"),
                monthly_contribution=Decimal("0.00"),
                is_completed=True,
            ),
        ]
    )
    db.commit()


def _seed_salaries(db) -> None:
    today = _today()
    months = _months_back(today, 24)
    records: list[SalaryRecord] = []
    for i, month_end in enumerate(months):
        # Marcin: B2B with annual raise
        marcin_gross = Decimal("18000.00") + Decimal("250.00") * (i // 12)
        records.append(
            SalaryRecord(
                date=month_end,
                gross_amount=marcin_gross,
                contract_type=ContractType.B2B.value,
                company="Acme Software sp. z o.o.",
                owner="Marcin",
                is_active=True,
            )
        )
        # Ewa: UOP
        ewa_gross = Decimal("9500.00") + Decimal("200.00") * (i // 12)
        records.append(
            SalaryRecord(
                date=month_end,
                gross_amount=ewa_gross,
                contract_type=ContractType.UOP.value,
                company="Beta Healthcare S.A.",
                owner="Ewa",
                is_active=True,
            )
        )
    db.add_all(records)
    db.commit()


def _seed_transactions(db, accounts: dict[str, Account]) -> None:
    today = _today()
    months = _months_back(today, 24)
    txs: list[Transaction] = []

    # IKE: lump-sum employee contribution every January
    for owner, account_name in (("Marcin", "IKE Marcin"), ("Ewa", "IKE Ewa")):
        account = accounts[account_name]
        for month_end in months:
            if month_end.month == 1:
                txs.append(
                    Transaction(
                        account_id=account.id,
                        amount=Decimal("23472.00"),
                        date=month_end,
                        owner=owner,
                        transaction_type=TransactionType.EMPLOYEE.value,
                        is_active=True,
                    )
                )

    # IKZE: monthly employee contributions
    for owner, account_name in (("Marcin", "IKZE Marcin"), ("Ewa", "IKZE Ewa")):
        account = accounts[account_name]
        for month_end in months:
            txs.append(
                Transaction(
                    account_id=account.id,
                    amount=Decimal("780.00"),
                    date=month_end,
                    owner=owner,
                    transaction_type=TransactionType.EMPLOYEE.value,
                    is_active=True,
                )
            )

    # PPK: employee + employer + government contributions
    for owner, account_name, monthly_employee in (
        ("Marcin", "PPK Marcin", Decimal("360.00")),
        ("Ewa", "PPK Ewa", Decimal("190.00")),
    ):
        account = accounts[account_name]
        for month_end in months:
            txs.append(
                Transaction(
                    account_id=account.id,
                    amount=monthly_employee,
                    date=month_end,
                    owner=owner,
                    transaction_type=TransactionType.EMPLOYEE.value,
                    is_active=True,
                )
            )
            txs.append(
                Transaction(
                    account_id=account.id,
                    amount=monthly_employee * Decimal("0.75"),
                    date=month_end,
                    owner=owner,
                    transaction_type=TransactionType.EMPLOYER.value,
                    is_active=True,
                )
            )
            # Government welcome bonus once per year (April)
            if month_end.month == 4:
                txs.append(
                    Transaction(
                        account_id=account.id,
                        amount=Decimal("240.00"),
                        date=month_end,
                        owner=owner,
                        transaction_type=TransactionType.GOVERNMENT.value,
                        is_active=True,
                    )
                )

    db.add_all(txs)
    db.commit()


def _seed_snapshots(db, accounts: dict[str, Account], assets: dict[str, Asset]) -> None:
    today = _today()
    months = _months_back(today, 24)

    # Per-account starting balances and monthly growth (PLN/month).
    profiles: dict[str, tuple[Decimal, Decimal]] = {
        "Konto Marcin": (Decimal("8000"), Decimal("150")),
        "Konto Ewa": (Decimal("5000"), Decimal("80")),
        "Oszczędności Shared": (Decimal("28000"), Decimal("1200")),
        "IKE Marcin": (Decimal("32000"), Decimal("450")),
        "IKE Ewa": (Decimal("18000"), Decimal("420")),
        "IKZE Marcin": (Decimal("12000"), Decimal("780")),
        "IKZE Ewa": (Decimal("8000"), Decimal("780")),
        "PPK Marcin": (Decimal("4500"), Decimal("640")),
        "PPK Ewa": (Decimal("2200"), Decimal("330")),
        "Akcje Shared": (Decimal("45000"), Decimal("900")),
        "Obligacje Shared": (Decimal("22000"), Decimal("400")),
        "Mieszkanie": (Decimal("680000"), Decimal("1500")),
        "Samochód": (Decimal("78000"), Decimal("-450")),
        "Hipoteka": (Decimal("395000"), Decimal("-1100")),
        "Raty 0% RTV": (Decimal("0"), Decimal("0")),
    }

    asset_profiles: dict[str, tuple[Decimal, Decimal]] = {
        "Złoto inwestycyjne": (Decimal("12000"), Decimal("260")),
        "Kryptowaluty": (Decimal("6500"), Decimal("180")),
    }

    for index, month_end in enumerate(months):
        snapshot = Snapshot(
            date=month_end,
            notes=f"Seeded snapshot for {month_end.isoformat()}",
        )
        db.add(snapshot)
        db.flush()

        values: list[SnapshotValue] = []
        # Small deterministic seasonality so charts aren't a perfect line.
        wobble = Decimal(str(round(math.sin(index / 2.0) * 250, 2)))

        for account_name, (start, monthly_delta) in profiles.items():
            # Installments debt only exists from Aug 2025 onward.
            if account_name == "Raty 0% RTV":
                if month_end < date(2025, 8, 1):
                    continue
                months_since = (month_end.year - 2025) * 12 + month_end.month - 8
                installment_value = max(
                    Decimal("0"),
                    Decimal("8400") - Decimal("350") * Decimal(str(months_since)),
                )
                value = installment_value
            else:
                value = start + monthly_delta * Decimal(str(index)) + wobble
                if value < 0:
                    value = Decimal("0")
            values.append(
                SnapshotValue(
                    snapshot_id=snapshot.id,
                    account_id=accounts[account_name].id,
                    value=value.quantize(Decimal("0.01")),
                )
            )

        for asset_name, (start, monthly_delta) in asset_profiles.items():
            value = start + monthly_delta * Decimal(str(index)) + wobble / 2
            if value < 0:
                value = Decimal("0")
            values.append(
                SnapshotValue(
                    snapshot_id=snapshot.id,
                    asset_id=assets[asset_name].id,
                    value=value.quantize(Decimal("0.01")),
                )
            )

        db.add_all(values)
    db.commit()


def seed_dev_data() -> None:
    """Idempotent demo seed. Bails out if any account already exists."""
    db = SessionLocal()
    try:
        if (db.scalar(select(func.count()).select_from(Account)) or 0) > 0:
            logger.info("Dev seed skipped: accounts already present")
            return
        logger.info("Seeding development demo data")
        _seed_app_config(db)
        _seed_cpi(db)
        accounts, assets = _seed_accounts_and_assets(db)
        _seed_debts_and_payments(db, accounts)
        _seed_goals(db)
        _seed_salaries(db)
        _seed_transactions(db, accounts)
        _seed_snapshots(db, accounts, assets)
        logger.info("Development demo data seeded")
    finally:
        db.close()
