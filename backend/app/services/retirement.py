from decimal import Decimal

from sqlalchemy import case, extract, func
from sqlalchemy.orm import Session

from app.models import Account, RetirementLimit, Transaction
from app.schemas.retirement import YearlyStatsResponse


def get_yearly_stats(db: Session, year: int, owner: str | None = None) -> list[YearlyStatsResponse]:
    """Calculate yearly contribution stats for IKE/IKZE per owner"""
    stats = []

    owners = [owner] if owner else ["Marcin", "Ewa"]
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

            # If no transaction_type specified, count as employee contribution
            if total > 0 and employee == 0 and employer == 0:
                employee = total

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
