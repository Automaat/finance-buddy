"""Database helper utilities for common query patterns."""

from typing import TypeVar

from fastapi import HTTPException
from sqlalchemy import select
from sqlalchemy.orm import Session

from app.models.snapshot import SnapshotValue

ModelT = TypeVar("ModelT")


def get_or_404(db: Session, model: type[ModelT], entity_id: int) -> ModelT:
    """
    Fetch entity by ID or raise 404.

    Args:
        db: Database session
        model: SQLAlchemy model class
        entity_id: Primary key value

    Returns:
        Model instance

    Raises:
        HTTPException: 404 if not found
    """
    entity = db.get(model, entity_id)
    if not entity:
        model_name = model.__name__
        raise HTTPException(status_code=404, detail=f"{model_name} with id {entity_id} not found")
    return entity


def get_latest_snapshot_value(db: Session, account_id: int) -> float:
    """
    Get most recent snapshot value for single account.

    Args:
        db: Database session
        account_id: Account ID to query

    Returns:
        Latest value or 0.0 if no snapshots exist
    """
    result = db.execute(
        select(SnapshotValue.value)
        .where(SnapshotValue.account_id == account_id)
        .order_by(SnapshotValue.snapshot_id.desc())
        .limit(1)
    ).scalar()
    return float(result) if result is not None else 0.0


def get_latest_snapshot_values_batch(db: Session, account_ids: list[int]) -> dict[int, float]:
    """
    Batch fetch latest snapshot values for multiple accounts.

    Args:
        db: Database session
        account_ids: List of account IDs

    Returns:
        Dict mapping account_id -> latest_value (0.0 if no snapshots)
    """
    if not account_ids:
        return {}

    # Subquery to get latest snapshot_id per account
    latest_snapshots = (
        select(
            SnapshotValue.account_id,
            SnapshotValue.snapshot_id,
            SnapshotValue.value,
        )
        .where(SnapshotValue.account_id.in_(account_ids))
        .order_by(SnapshotValue.account_id, SnapshotValue.snapshot_id.desc())
    )

    results = db.execute(latest_snapshots).all()

    # Group by account_id, take first (latest) value
    values_map: dict[int, float] = {}
    for account_id, _, value in results:
        if account_id not in values_map:
            values_map[account_id] = value

    # Fill missing accounts with 0.0
    for account_id in account_ids:
        if account_id not in values_map:
            values_map[account_id] = 0.0

    return values_map


def soft_delete(db: Session, model: type[ModelT], entity_id: int) -> None:
    """
    Soft delete entity by setting is_active=False (idempotent).

    Args:
        db: Database session
        model: SQLAlchemy model class with is_active field
        entity_id: Primary key value

    Raises:
        HTTPException: 404 if not found
    """
    entity = get_or_404(db, model, entity_id)

    # Idempotent: early return if already inactive
    if not entity.is_active:  # type: ignore[attr-defined]
        return

    entity.is_active = False  # type: ignore[attr-defined]
    db.commit()


def check_duplicate_name(
    db: Session,
    model: type[ModelT],
    name: str,
    exclude_id: int | None = None,
    active_only: bool = True,
) -> None:
    """
    Check for duplicate name, raise 400 if exists.

    Args:
        db: Database session
        model: SQLAlchemy model class with name field
        name: Name to check
        exclude_id: Optional ID to exclude (for updates)
        active_only: Only check active records (is_active=True), default True

    Raises:
        HTTPException: 400 if duplicate name exists
    """
    query = select(model).where(model.name == name)  # type: ignore[attr-defined]
    if active_only:
        query = query.where(model.is_active.is_(True))  # type: ignore[attr-defined]
    if exclude_id is not None:
        query = query.where(model.id != exclude_id)  # type: ignore[attr-defined]

    existing = db.execute(query).scalar_one_or_none()
    if existing:
        model_name = model.__name__
        status = "Active " if active_only else ""
        raise HTTPException(
            status_code=400, detail=f"{status}{model_name.lower()} '{name}' already exists"
        )
