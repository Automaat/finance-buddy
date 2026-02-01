from decimal import Decimal

from fastapi import HTTPException
from sqlalchemy import delete, desc, select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from app.models import Account, Asset, Snapshot, SnapshotValue
from app.schemas.snapshots import (
    SnapshotCreate,
    SnapshotListItem,
    SnapshotResponse,
    SnapshotUpdate,
    SnapshotValueResponse,
)


def create_snapshot(db: Session, data: SnapshotCreate) -> SnapshotResponse:
    """Create new snapshot with all asset and account values atomically"""
    # Check if snapshot for this date already exists
    existing = db.execute(select(Snapshot).where(Snapshot.date == data.date)).scalar_one_or_none()
    if existing:
        raise HTTPException(status_code=400, detail=f"Snapshot for date {data.date} already exists")

    # Separate asset and account IDs
    asset_ids = [v.asset_id for v in data.values if v.asset_id is not None]
    account_ids = [v.account_id for v in data.values if v.account_id is not None]

    # Validate no duplicates within each type
    if len(asset_ids) != len(set(asset_ids)):
        raise HTTPException(status_code=400, detail="Duplicate asset IDs in snapshot values")
    if len(account_ids) != len(set(account_ids)):
        raise HTTPException(status_code=400, detail="Duplicate account IDs in snapshot values")

    # Validate all assets exist
    if asset_ids:
        assets = (
            db.execute(select(Asset).where(Asset.id.in_(asset_ids), Asset.is_active.is_(True)))
            .scalars()
            .all()
        )
        if len(assets) != len(asset_ids):
            found_ids = {a.id for a in assets}
            missing = set(asset_ids) - found_ids
            raise HTTPException(status_code=404, detail=f"Assets not found: {missing}")
    else:
        assets = []

    # Validate all accounts exist
    if account_ids:
        accounts = (
            db.execute(
                select(Account).where(Account.id.in_(account_ids), Account.is_active.is_(True))
            )
            .scalars()
            .all()
        )
        if len(accounts) != len(account_ids):
            found_ids = {a.id for a in accounts}
            missing = set(account_ids) - found_ids
            raise HTTPException(status_code=404, detail=f"Accounts not found: {missing}")
    else:
        accounts = []

    # Create snapshot
    snapshot = Snapshot(date=data.date, notes=data.notes)
    db.add(snapshot)
    db.flush()  # Get ID for FK constraint

    # Create all snapshot values
    snapshot_values = []
    for value_input in data.values:
        sv = SnapshotValue(
            snapshot_id=snapshot.id,
            asset_id=value_input.asset_id,
            account_id=value_input.account_id,
            value=Decimal(str(value_input.value)),
        )
        db.add(sv)
        snapshot_values.append(sv)

    try:
        db.commit()
    except IntegrityError as e:
        db.rollback()
        raise HTTPException(status_code=400, detail="Failed to create snapshot") from e

    # Build response
    asset_map = {a.id: a.name for a in assets}
    account_map = {a.id: a.name for a in accounts}
    values = [
        SnapshotValueResponse(
            id=sv.id,
            asset_id=sv.asset_id,
            asset_name=asset_map.get(sv.asset_id) if sv.asset_id else None,
            account_id=sv.account_id,
            account_name=account_map.get(sv.account_id) if sv.account_id else None,
            value=float(sv.value),
        )
        for sv in snapshot_values
    ]

    return SnapshotResponse(id=snapshot.id, date=snapshot.date, notes=snapshot.notes, values=values)


def get_all_snapshots(db: Session) -> list[SnapshotListItem]:
    """Get all snapshots with net worth calculation"""
    snapshots = db.execute(select(Snapshot).order_by(desc(Snapshot.date))).scalars().all()

    result = []
    for snapshot in snapshots:
        # Calculate net worth for this snapshot
        # LEFT JOIN both Asset and Account tables
        values = db.execute(
            select(SnapshotValue, Asset, Account)
            .outerjoin(Asset, SnapshotValue.asset_id == Asset.id)
            .outerjoin(Account, SnapshotValue.account_id == Account.id)
            .where(SnapshotValue.snapshot_id == snapshot.id)
        ).all()

        net_worth = 0.0
        for sv, asset, account in values:
            # Assets contribute positively, account type determines sign
            if asset is not None:
                net_worth += float(sv.value)
            elif account is not None:
                if account.type == "asset":
                    net_worth += float(sv.value)
                else:  # liability
                    net_worth -= float(sv.value)

        result.append(
            SnapshotListItem(
                id=snapshot.id, date=snapshot.date, notes=snapshot.notes, total_net_worth=net_worth
            )
        )

    return result


def get_snapshot_by_id(db: Session, snapshot_id: int) -> SnapshotResponse:
    """Get single snapshot with all values"""
    snapshot = db.execute(select(Snapshot).where(Snapshot.id == snapshot_id)).scalar_one_or_none()
    if not snapshot:
        raise HTTPException(status_code=404, detail=f"Snapshot {snapshot_id} not found")

    # Get all values with asset and account names (LEFT JOIN both)
    values_query = (
        select(SnapshotValue, Asset, Account)
        .outerjoin(Asset, SnapshotValue.asset_id == Asset.id)
        .outerjoin(Account, SnapshotValue.account_id == Account.id)
        .where(SnapshotValue.snapshot_id == snapshot_id)
    )
    values = db.execute(values_query).all()

    value_responses = [
        SnapshotValueResponse(
            id=sv.id,
            asset_id=sv.asset_id,
            asset_name=asset.name if asset else None,
            account_id=sv.account_id,
            account_name=account.name if account else None,
            value=float(sv.value),
        )
        for sv, asset, account in values
    ]

    return SnapshotResponse(
        id=snapshot.id, date=snapshot.date, notes=snapshot.notes, values=value_responses
    )


def update_snapshot(db: Session, snapshot_id: int, data: SnapshotUpdate) -> SnapshotResponse:
    """Update existing snapshot with optional date, notes, and/or values"""
    # Get snapshot or 404
    snapshot = db.execute(select(Snapshot).where(Snapshot.id == snapshot_id)).scalar_one_or_none()
    if not snapshot:
        raise HTTPException(status_code=404, detail=f"Snapshot {snapshot_id} not found")

    # Update date if provided (validate uniqueness excluding current snapshot)
    if data.date is not None:
        existing = db.execute(
            select(Snapshot).where(Snapshot.date == data.date, Snapshot.id != snapshot_id)
        ).scalar_one_or_none()
        if existing:
            raise HTTPException(
                status_code=400, detail=f"Snapshot for date {data.date} already exists"
            )
        snapshot.date = data.date

    # Update notes (use model_fields_set to distinguish None from omitted)
    if "notes" in data.model_fields_set:
        snapshot.notes = data.notes

    # Replace values if provided
    if data.values is not None:
        # Separate asset and account IDs
        asset_ids = [v.asset_id for v in data.values if v.asset_id is not None]
        account_ids = [v.account_id for v in data.values if v.account_id is not None]

        # Validate no duplicates within each type
        if len(asset_ids) != len(set(asset_ids)):
            raise HTTPException(status_code=400, detail="Duplicate asset IDs in snapshot values")
        if len(account_ids) != len(set(account_ids)):
            raise HTTPException(status_code=400, detail="Duplicate account IDs in snapshot values")

        # Validate all assets exist
        if asset_ids:
            assets = (
                db.execute(select(Asset).where(Asset.id.in_(asset_ids)))
                .scalars()
                .all()
            )
            if len(assets) != len(asset_ids):
                found_ids = {a.id for a in assets}
                missing = set(asset_ids) - found_ids
                raise HTTPException(status_code=404, detail=f"Assets not found: {missing}")
        else:
            assets = []

        # Validate all accounts exist
        if account_ids:
            accounts = (
                db.execute(
                    select(Account).where(Account.id.in_(account_ids))
                )
                .scalars()
                .all()
            )
            if len(accounts) != len(account_ids):
                found_ids = {a.id for a in accounts}
                missing = set(account_ids) - found_ids
                raise HTTPException(status_code=404, detail=f"Accounts not found: {missing}")
        else:
            accounts = []

        # Delete existing values
        db.execute(delete(SnapshotValue).where(SnapshotValue.snapshot_id == snapshot_id))

        # Create new values
        snapshot_values = []
        for value_input in data.values:
            sv = SnapshotValue(
                snapshot_id=snapshot.id,
                asset_id=value_input.asset_id,
                account_id=value_input.account_id,
                value=Decimal(str(value_input.value)),
            )
            db.add(sv)
            snapshot_values.append(sv)
    else:
        # No values update - fetch existing
        assets = []
        accounts = []
        snapshot_values = []

    try:
        db.commit()
        db.refresh(snapshot)
    except IntegrityError as e:
        db.rollback()
        detail = "Failed to update snapshot"
        if hasattr(e, "orig") and e.orig:
            detail = f"{detail}: {e.orig}"
        raise HTTPException(status_code=400, detail=detail) from e

    # Build response - fetch fresh values if not replaced
    if data.values is None:
        values_query = (
            select(SnapshotValue, Asset, Account)
            .outerjoin(Asset, SnapshotValue.asset_id == Asset.id)
            .outerjoin(Account, SnapshotValue.account_id == Account.id)
            .where(SnapshotValue.snapshot_id == snapshot_id)
        )
        values_result = db.execute(values_query).all()
        value_responses = [
            SnapshotValueResponse(
                id=sv.id,
                asset_id=sv.asset_id,
                asset_name=asset.name if asset else None,
                account_id=sv.account_id,
                account_name=account.name if account else None,
                value=float(sv.value),
            )
            for sv, asset, account in values_result
        ]
    else:
        # Build response from newly created values
        asset_map = {a.id: a.name for a in assets}
        account_map = {a.id: a.name for a in accounts}
        value_responses = [
            SnapshotValueResponse(
                id=sv.id,
                asset_id=sv.asset_id,
                asset_name=asset_map.get(sv.asset_id) if sv.asset_id else None,
                account_id=sv.account_id,
                account_name=account_map.get(sv.account_id) if sv.account_id else None,
                value=float(sv.value),
            )
            for sv in snapshot_values
        ]

    return SnapshotResponse(
        id=snapshot.id, date=snapshot.date, notes=snapshot.notes, values=value_responses
    )
