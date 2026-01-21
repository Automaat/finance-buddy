"""Test data factories for reducing duplication across test files."""

from datetime import date
from decimal import Decimal
from typing import Any

from sqlalchemy.orm import Session

from app.models import (
    Account,
    Asset,
    Debt,
    DebtPayment,
    SalaryRecord,
    Snapshot,
    SnapshotValue,
    Transaction,
)


def create_test_account(
    db: Session,
    name: str = "Test Account",
    account_type: str = "asset",
    category: str = "bank",
    owner: str = "Marcin",
    currency: str = "PLN",
    purpose: str = "general",
    is_active: bool = True,
    **kwargs: Any,
) -> Account:
    """Factory for test accounts with sensible defaults.

    Args:
        db: Database session
        name: Account name
        account_type: Account type (asset, liability, income, expense)
        category: Account category (bank, stock, bond, fund, mortgage, etc.)
        owner: Account owner (Marcin, Ewa, Shared)
        currency: Currency code (PLN, USD, EUR)
        purpose: Account purpose (general, retirement, emergency)
        is_active: Whether account is active
        **kwargs: Additional fields (e.g., account_wrapper="IKE")

    Returns:
        Created Account instance
    """
    defaults = {
        "name": name,
        "type": account_type,
        "category": category,
        "owner": owner,
        "currency": currency,
        "purpose": purpose,
        "is_active": is_active,
    }
    defaults.update(kwargs)
    account = Account(**defaults)
    db.add(account)
    db.commit()
    db.refresh(account)
    return account


def create_test_snapshot(
    db: Session,
    snapshot_date: date = date(2024, 1, 31),
    notes: str = "",
    **kwargs: Any,
) -> Snapshot:
    """Factory for test snapshots with sensible defaults.

    Args:
        db: Database session
        snapshot_date: Snapshot date (defaults to end of January 2024)
        notes: Optional notes
        **kwargs: Additional fields

    Returns:
        Created Snapshot instance
    """
    defaults = {
        "date": snapshot_date,
        "notes": notes,
    }
    defaults.update(kwargs)
    snapshot = Snapshot(**defaults)
    db.add(snapshot)
    db.commit()
    db.refresh(snapshot)
    return snapshot


def create_test_snapshot_value(
    db: Session,
    snapshot_id: int,
    account_id: int | None = None,
    value: Decimal | float = Decimal("1000.00"),
    asset_id: int | None = None,
    **kwargs: Any,
) -> SnapshotValue:
    """Factory for test snapshot values.

    SnapshotValue can track either an account OR an asset (mutually exclusive).
    Provide either account_id or asset_id, not both.

    Args:
        db: Database session
        snapshot_id: Foreign key to snapshot
        account_id: Foreign key to account (for financial accounts)
        value: Value at snapshot date
        asset_id: Foreign key to asset (for physical assets like cars)
        **kwargs: Additional fields

    Returns:
        Created SnapshotValue instance
    """
    defaults: dict[str, Any] = {
        "snapshot_id": snapshot_id,
        "value": Decimal(str(value)) if isinstance(value, float) else value,
    }
    if asset_id is not None:
        defaults["asset_id"] = asset_id
    elif account_id is not None:
        defaults["account_id"] = account_id
    defaults.update(kwargs)
    snapshot_value = SnapshotValue(**defaults)
    db.add(snapshot_value)
    db.commit()
    db.refresh(snapshot_value)
    return snapshot_value


def create_test_debt(
    db: Session,
    account_id: int,
    name: str = "Test Debt",
    debt_type: str = "mortgage",
    start_date: date = date(2020, 1, 15),
    initial_amount: float = 500000.0,
    interest_rate: float = 3.5,
    currency: str = "PLN",
    is_active: bool = True,
    **kwargs: Any,
) -> Debt:
    """Factory for test debts with sensible defaults.

    Args:
        db: Database session
        account_id: Foreign key to liability account
        name: Debt name
        debt_type: Type (mortgage, installment_0percent, installment)
        start_date: Debt start date
        initial_amount: Initial debt amount
        interest_rate: Annual interest rate percentage
        currency: Currency code
        is_active: Whether debt is active
        **kwargs: Additional fields

    Returns:
        Created Debt instance
    """
    defaults = {
        "account_id": account_id,
        "name": name,
        "debt_type": debt_type,
        "start_date": start_date,
        "initial_amount": initial_amount,
        "interest_rate": interest_rate,
        "currency": currency,
        "is_active": is_active,
    }
    defaults.update(kwargs)
    debt = Debt(**defaults)
    db.add(debt)
    db.commit()
    db.refresh(debt)
    return debt


def create_test_transaction(
    db: Session,
    account_id: int,
    amount: float = 5000.0,
    transaction_date: date = date(2024, 1, 15),
    owner: str = "Marcin",
    is_active: bool = True,
    **kwargs: Any,
) -> Transaction:
    """Factory for test transactions with sensible defaults.

    Args:
        db: Database session
        account_id: Foreign key to investment account
        amount: Transaction amount
        transaction_date: Transaction date
        owner: Transaction owner (Marcin, Ewa, Shared)
        is_active: Whether transaction is active
        **kwargs: Additional fields

    Returns:
        Created Transaction instance
    """
    defaults = {
        "account_id": account_id,
        "amount": amount,
        "date": transaction_date,
        "owner": owner,
        "is_active": is_active,
    }
    defaults.update(kwargs)
    transaction = Transaction(**defaults)
    db.add(transaction)
    db.commit()
    db.refresh(transaction)
    return transaction


def create_test_asset(
    db: Session,
    name: str = "Test Asset",
    is_active: bool = True,
    **kwargs: Any,
) -> Asset:
    """Factory for test assets with sensible defaults.

    Args:
        db: Database session
        name: Asset name
        is_active: Whether asset is active
        **kwargs: Additional fields

    Returns:
        Created Asset instance
    """
    defaults = {
        "name": name,
        "is_active": is_active,
    }
    defaults.update(kwargs)
    asset = Asset(**defaults)
    db.add(asset)
    db.commit()
    db.refresh(asset)
    return asset


def create_test_salary_record(
    db: Session,
    salary_date: date = date(2024, 1, 1),
    gross_amount: float = 10000.0,
    contract_type: str = "UOP",
    company: str = "Company A",
    owner: str = "Marcin",
    is_active: bool = True,
    **kwargs: Any,
) -> SalaryRecord:
    """Factory for test salary records with sensible defaults.

    Args:
        db: Database session
        salary_date: Salary record date
        gross_amount: Gross salary amount
        contract_type: Contract type (UOP, UZ, B2B)
        company: Company name
        owner: Salary owner (Marcin, Ewa)
        is_active: Whether record is active
        **kwargs: Additional fields

    Returns:
        Created SalaryRecord instance
    """
    defaults = {
        "date": salary_date,
        "gross_amount": gross_amount,
        "contract_type": contract_type,
        "company": company,
        "owner": owner,
        "is_active": is_active,
    }
    defaults.update(kwargs)
    salary_record = SalaryRecord(**defaults)
    db.add(salary_record)
    db.commit()
    db.refresh(salary_record)
    return salary_record


def create_test_debt_payment(
    db: Session,
    account_id: int,
    amount: float = 50000.0,
    payment_date: date = date(2024, 6, 1),
    owner: str = "Shared",
    is_active: bool = True,
    **kwargs: Any,
) -> DebtPayment:
    """Factory for test debt payments with sensible defaults.

    Args:
        db: Database session
        account_id: Foreign key to liability account
        amount: Payment amount
        payment_date: Payment date
        owner: Payment owner (Marcin, Ewa, Shared)
        is_active: Whether payment is active
        **kwargs: Additional fields

    Returns:
        Created DebtPayment instance
    """
    defaults = {
        "account_id": account_id,
        "amount": amount,
        "date": payment_date,
        "owner": owner,
        "is_active": is_active,
    }
    defaults.update(kwargs)
    debt_payment = DebtPayment(**defaults)
    db.add(debt_payment)
    db.commit()
    db.refresh(debt_payment)
    return debt_payment
