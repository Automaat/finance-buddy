from sqlalchemy import func
from sqlalchemy.orm import Session

from app.models import Account, Snapshot, SnapshotValue, Transaction
from app.schemas.investment import CategoryStatsResponse


def get_category_stats(db: Session, category: str) -> CategoryStatsResponse:
    """Calculate aggregate statistics for a given investment category (stock/bond)"""
    # Get all active accounts for this category
    accounts = (
        db.query(Account).filter(Account.category == category, Account.is_active.is_(True)).all()
    )

    if not accounts:
        return CategoryStatsResponse(
            category=category,
            total_value=0.0,
            total_contributed=0.0,
            returns=0.0,
            roi_percentage=0.0,
        )

    account_ids = [acc.id for acc in accounts]

    # Get latest snapshot date (subquery) - needed for both transactions and snapshot value
    max_date_subquery = (
        db.query(func.max(Snapshot.date))
        .join(SnapshotValue, Snapshot.id == SnapshotValue.snapshot_id)
        .filter(SnapshotValue.account_id.in_(account_ids))
        .scalar_subquery()
    )

    # Sum active transactions up to latest snapshot date (or all if no snapshots)
    transaction_query = db.query(func.coalesce(func.sum(Transaction.amount), 0)).filter(
        Transaction.account_id.in_(account_ids), Transaction.is_active.is_(True)
    )

    # Only filter by date if snapshots exist (max_date_subquery returns value, not NULL)
    has_snapshots = (
        db.query(Snapshot.id)
        .join(SnapshotValue, Snapshot.id == SnapshotValue.snapshot_id)
        .filter(SnapshotValue.account_id.in_(account_ids))
        .limit(1)
        .scalar()
    )

    if has_snapshots:
        transaction_query = transaction_query.filter(Transaction.date <= max_date_subquery)

    total_contributed = float(transaction_query.scalar() or 0)

    # Get sum of values for the latest snapshot
    latest_snapshot_value = (
        db.query(func.sum(SnapshotValue.value))
        .join(Snapshot, SnapshotValue.snapshot_id == Snapshot.id)
        .filter(SnapshotValue.account_id.in_(account_ids), Snapshot.date == max_date_subquery)
        .scalar()
    )
    total_value = float(latest_snapshot_value if latest_snapshot_value else 0)

    # Calculate returns and ROI
    returns = total_value - total_contributed
    roi_percentage = (returns / total_contributed * 100) if total_contributed > 0 else 0.0

    return CategoryStatsResponse(
        category=category,
        total_value=total_value,
        total_contributed=total_contributed,
        returns=returns,
        roi_percentage=round(roi_percentage, 2),
    )
