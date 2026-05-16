from datetime import UTC, datetime

from sqlalchemy import desc, select
from sqlalchemy.orm import Session

from app.models.account import Account
from app.models.app_config import AppConfig
from app.models.persona import Persona
from app.models.snapshot import Snapshot, SnapshotValue


def fetch_current_balances(db: Session) -> dict:
    """Get latest IKE/IKZE balances from snapshot, keyed by {wrapper}_{owner} lowercase"""
    latest_snapshot = db.execute(
        select(Snapshot).order_by(desc(Snapshot.date)).limit(1)
    ).scalar_one_or_none()

    # Build default keys from personas
    personas = db.execute(select(Persona).order_by(Persona.name)).scalars().all()
    balances: dict[str, float] = {}
    for p in personas:
        for wrapper in ["ike", "ikze"]:
            balances[f"{wrapper}_{p.name.lower()}"] = 0.0

    if not latest_snapshot:
        return balances

    accounts = (
        db.execute(
            select(Account).where(
                Account.is_active.is_(True), Account.account_wrapper.in_(["IKE", "IKZE"])
            )
        )
        .scalars()
        .all()
    )

    values = (
        db.execute(select(SnapshotValue).where(SnapshotValue.snapshot_id == latest_snapshot.id))
        .scalars()
        .all()
    )

    values_by_account_id = {value.account_id: value.value for value in values}

    for account in accounts:
        if account.id in values_by_account_id and account.account_wrapper:
            key = f"{account.account_wrapper.lower()}_{account.owner.lower()}"
            balances[key] = float(values_by_account_id[account.id])

    return balances


def fetch_ppk_balances(db: Session) -> dict[str, float]:
    """Get latest PPK balances from snapshot by owner"""
    latest_snapshot = db.execute(
        select(Snapshot).order_by(desc(Snapshot.date)).limit(1)
    ).scalar_one_or_none()

    # Build default keys from personas
    personas = db.execute(select(Persona).order_by(Persona.name)).scalars().all()
    balances: dict[str, float] = {p.name.lower(): 0.0 for p in personas}

    if not latest_snapshot:
        return balances

    accounts = (
        db.execute(
            select(Account).where(Account.is_active.is_(True), Account.account_wrapper == "PPK")
        )
        .scalars()
        .all()
    )

    values = (
        db.execute(select(SnapshotValue).where(SnapshotValue.snapshot_id == latest_snapshot.id))
        .scalars()
        .all()
    )

    values_by_account_id = {value.account_id: value.value for value in values}

    for account in accounts:
        if account.id in values_by_account_id and account.account_wrapper == "PPK":
            key = account.owner.lower()
            balances[key] = float(values_by_account_id[account.id])

    return balances


def get_age_from_config(db: Session) -> int:
    """Calculate current age from AppConfig.birth_date"""
    config = db.execute(select(AppConfig)).scalar_one_or_none()
    if not config or not config.birth_date:
        return 30  # Fallback default if config not initialized
    today = datetime.now(UTC).date()
    age = today.year - config.birth_date.year
    if today.month < config.birth_date.month or (
        today.month == config.birth_date.month and today.day < config.birth_date.day
    ):
        age -= 1
    return age
