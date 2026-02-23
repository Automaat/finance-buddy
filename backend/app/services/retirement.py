import calendar
from datetime import date
from decimal import Decimal

from fastapi import HTTPException
from sqlalchemy import case, extract, func, or_
from sqlalchemy.orm import Session

from app.models import Account, RetirementLimit, Snapshot, SnapshotValue, Transaction
from app.models.persona import Persona
from app.schemas.retirement import (
    PPKContributionGenerateRequest,
    PPKContributionGenerateResponse,
    PPKStatsResponse,
    YearlyStatsResponse,
)
from app.services.salary_records import _get_current_salary


def get_yearly_stats(db: Session, year: int, owner: str | None = None) -> list[YearlyStatsResponse]:
    """Calculate yearly contribution stats for IKE/IKZE per owner"""
    stats = []

    owners = [owner] if owner else [p.name for p in db.query(Persona).order_by(Persona.name).all()]
    wrappers = ["IKE", "IKZE"]

    for wrapper in wrappers:
        for owner_name in owners:
            # Get all accounts with this wrapper and owner
            accounts = (
                db.query(Account)
                .filter(Account.account_wrapper == wrapper, Account.owner == owner_name)
                .all()
            )
            account_ids = [acc.id for acc in accounts]

            if not account_ids:
                continue

            # Aggregate transactions for this year
            contributions = (
                db.query(
                    func.coalesce(func.sum(Transaction.amount), 0).label("total"),
                    func.coalesce(
                        func.sum(
                            case(
                                (Transaction.transaction_type == "employee", Transaction.amount),
                                else_=0,
                            )
                        ),
                        0,
                    ).label("employee"),
                    func.coalesce(
                        func.sum(
                            case(
                                (Transaction.transaction_type == "employer", Transaction.amount),
                                else_=0,
                            )
                        ),
                        0,
                    ).label("employer"),
                )
                .filter(
                    Transaction.account_id.in_(account_ids),
                    extract("year", Transaction.date) == year,
                    Transaction.is_active.is_(True),
                )
                .first()
            )

            total = float(contributions.total if contributions else 0)
            employee = float(contributions.employee if contributions else 0)
            employer = float(contributions.employer if contributions else 0)

            # If some transactions have no type, assign the untyped amount to employee
            typed_sum = employee + employer
            if total > typed_sum:
                untyped_amount = total - typed_sum
                employee += untyped_amount

            # Get limit for this wrapper and owner
            limit = (
                db.query(RetirementLimit)
                .filter(
                    RetirementLimit.year == year,
                    RetirementLimit.account_wrapper == wrapper,
                    RetirementLimit.owner == owner_name,
                )
                .first()
            )

            if not limit:
                continue

            remaining = float(limit.limit_amount) - total
            percentage = (total / float(limit.limit_amount) * 100) if limit.limit_amount > 0 else 0

            stats.append(
                YearlyStatsResponse(
                    year=year,
                    account_wrapper=wrapper,
                    owner=owner_name,
                    limit_amount=float(limit.limit_amount),
                    total_contributed=total,
                    employee_contributed=employee,
                    employer_contributed=employer,
                    remaining=max(0.0, remaining),
                    percentage_used=round(percentage, 1),
                    is_warning=percentage >= 90,
                )
            )

    return stats


def get_or_create_limit(
    db: Session, year: int, wrapper: str, owner: str, amount: float, notes: str | None = None
) -> RetirementLimit:
    """Get existing limit or create new one"""
    limit = (
        db.query(RetirementLimit)
        .filter(
            RetirementLimit.year == year,
            RetirementLimit.account_wrapper == wrapper,
            RetirementLimit.owner == owner,
        )
        .first()
    )

    if limit:
        return limit

    limit = RetirementLimit(
        year=year,
        account_wrapper=wrapper,
        owner=owner,
        limit_amount=Decimal(str(amount)),
        notes=notes,
    )
    db.add(limit)
    db.commit()
    db.refresh(limit)
    return limit


def update_limit(
    db: Session, year: int, wrapper: str, owner: str, amount: float, notes: str | None = None
) -> RetirementLimit:
    """Update existing limit"""
    limit = (
        db.query(RetirementLimit)
        .filter(
            RetirementLimit.year == year,
            RetirementLimit.account_wrapper == wrapper,
            RetirementLimit.owner == owner,
        )
        .first()
    )

    if not limit:
        return get_or_create_limit(db, year, wrapper, owner, amount, notes)

    limit.limit_amount = Decimal(str(amount))
    if notes is not None:
        limit.notes = notes
    db.commit()
    db.refresh(limit)
    return limit


def get_ppk_stats(db: Session, owner: str | None = None) -> list[PPKStatsResponse]:
    """Calculate PPK contribution breakdown and returns per owner"""
    stats = []
    owners = [owner] if owner else [p.name for p in db.query(Persona).order_by(Persona.name).all()]

    for owner_name in owners:
        # Get all PPK accounts for this owner
        accounts = (
            db.query(Account)
            .filter(Account.account_wrapper == "PPK", Account.owner == owner_name)
            .all()
        )

        if not accounts:
            continue

        account_ids = [acc.id for acc in accounts]

        # Aggregate all-time contributions by type
        contributions = (
            db.query(
                func.coalesce(
                    func.sum(
                        case(
                            (
                                or_(
                                    Transaction.transaction_type.in_(
                                        ["employee", "employer", "government"]
                                    ),
                                    Transaction.transaction_type.is_(None),
                                ),
                                Transaction.amount,
                            ),
                            else_=0,
                        )
                    ),
                    0,
                ).label("total"),
                func.coalesce(
                    func.sum(
                        case(
                            (Transaction.transaction_type == "employee", Transaction.amount),
                            else_=0,
                        )
                    ),
                    0,
                ).label("employee"),
                func.coalesce(
                    func.sum(
                        case(
                            (Transaction.transaction_type == "employer", Transaction.amount),
                            else_=0,
                        )
                    ),
                    0,
                ).label("employer"),
                func.coalesce(
                    func.sum(
                        case(
                            (Transaction.transaction_type == "government", Transaction.amount),
                            else_=0,
                        )
                    ),
                    0,
                ).label("government"),
            )
            .filter(Transaction.account_id.in_(account_ids), Transaction.is_active.is_(True))
            .first()
        )

        employee_contrib = float(contributions.employee if contributions else 0)
        employer_contrib = float(contributions.employer if contributions else 0)
        government_contrib = float(contributions.government if contributions else 0)
        total_contrib = float(contributions.total if contributions else 0)

        # If some transactions have no type, assign the untyped amount to employee
        typed_sum = employee_contrib + employer_contrib + government_contrib
        if total_contrib > typed_sum:
            untyped_amount = total_contrib - typed_sum
            employee_contrib += untyped_amount

        # Get latest snapshot date first (subquery for performance)
        max_date_subquery = (
            db.query(func.max(Snapshot.date))
            .join(SnapshotValue, Snapshot.id == SnapshotValue.snapshot_id)
            .filter(SnapshotValue.account_id.in_(account_ids))
            .scalar_subquery()
        )

        # Get sum of values for the latest snapshot
        latest_snapshot_value = (
            db.query(func.sum(SnapshotValue.value))
            .join(Snapshot, SnapshotValue.snapshot_id == Snapshot.id)
            .filter(SnapshotValue.account_id.in_(account_ids), Snapshot.date == max_date_subquery)
            .scalar()
        )

        total_value = float(latest_snapshot_value if latest_snapshot_value else 0)

        # Calculate returns and ROI
        returns = total_value - total_contrib
        roi_percentage = (returns / total_contrib * 100) if total_contrib > 0 else 0.0

        stats.append(
            PPKStatsResponse(
                owner=owner_name,
                total_value=total_value,
                employee_contributed=employee_contrib,
                employer_contributed=employer_contrib,
                government_contributed=government_contrib,
                total_contributed=total_contrib,
                returns=returns,
                roi_percentage=round(roi_percentage, 2),
            )
        )

    return stats


def generate_ppk_contributions(
    db: Session, data: PPKContributionGenerateRequest
) -> PPKContributionGenerateResponse:
    """Generate PPK contributions for a specific month based on salary and configured rates"""
    # Get salary for first day of specified month
    first_day = date(data.year, data.month, 1)
    gross_salary = _get_current_salary(db, data.owner, first_day)

    if gross_salary is None:
        raise HTTPException(
            status_code=400,
            detail=f"No salary record found for {data.owner} in {data.month}/{data.year}",
        )

    # Get PPK rates from Persona table
    persona = db.query(Persona).filter(Persona.name == data.owner).first()
    if not persona:
        raise HTTPException(status_code=404, detail=f"Persona '{data.owner}' not found")

    employee_rate = float(persona.ppk_employee_rate)
    employer_rate = float(persona.ppk_employer_rate)

    # Calculate contribution amounts
    employee_amount = gross_salary * (employee_rate / 100)
    employer_amount = gross_salary * (employer_rate / 100)

    # Get active PPK account for this owner (receives contributions)
    ppk_account = (
        db.query(Account)
        .filter(
            Account.account_wrapper == "PPK",
            Account.owner == data.owner,
            Account.is_active.is_(True),
            Account.receives_contributions.is_(True),
        )
        .first()
    )

    if not ppk_account:
        raise HTTPException(
            status_code=404,
            detail=f"No active PPK account found for {data.owner}. "
            "Mark one PPK account as receiving contributions.",
        )

    # Check if contributions already exist for this month
    # DB unique constraint provides ultimate safeguard against race conditions
    last_day = date(data.year, data.month, calendar.monthrange(data.year, data.month)[1])
    existing_count = (
        db.query(Transaction)
        .filter(
            Transaction.account_id == ppk_account.id,
            Transaction.date == last_day,
            Transaction.transaction_type.in_(["employee", "employer"]),
            Transaction.is_active.is_(True),
        )
        .count()
    )

    if existing_count > 0:
        raise HTTPException(
            status_code=409,
            detail=f"Contributions already exist for {data.month}/{data.year}",
        )

    # Create transactions
    employee_transaction = Transaction(
        account_id=ppk_account.id,
        amount=Decimal(str(employee_amount)),
        date=last_day,
        owner=data.owner,
        transaction_type="employee",
    )
    employer_transaction = Transaction(
        account_id=ppk_account.id,
        amount=Decimal(str(employer_amount)),
        date=last_day,
        owner=data.owner,
        transaction_type="employer",
    )

    db.add(employee_transaction)
    db.add(employer_transaction)
    db.commit()
    db.refresh(employee_transaction)
    db.refresh(employer_transaction)

    return PPKContributionGenerateResponse(
        owner=data.owner,
        month=data.month,
        year=data.year,
        gross_salary=gross_salary,
        employee_amount=employee_amount,
        employer_amount=employer_amount,
        total_amount=employee_amount + employer_amount,
        transactions_created=[employee_transaction.id, employer_transaction.id],
    )
