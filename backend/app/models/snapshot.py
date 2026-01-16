from datetime import UTC, date, datetime
from decimal import Decimal

from sqlalchemy import Date, ForeignKey, Numeric, Text, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column

from app.core.database import Base


class Snapshot(Base):
    __tablename__ = "snapshots"

    id: Mapped[int] = mapped_column(primary_key=True)
    date: Mapped[date] = mapped_column(Date, unique=True)
    notes: Mapped[str | None] = mapped_column(Text, nullable=True)
    created_at: Mapped[datetime] = mapped_column(default=lambda: datetime.now(UTC))


class SnapshotValue(Base):
    __tablename__ = "snapshot_values"
    __table_args__ = (UniqueConstraint("snapshot_id", "account_id", name="uix_snapshot_account"),)

    id: Mapped[int] = mapped_column(primary_key=True)
    snapshot_id: Mapped[int] = mapped_column(ForeignKey("snapshots.id", ondelete="CASCADE"))
    account_id: Mapped[int] = mapped_column(ForeignKey("accounts.id", ondelete="CASCADE"))
    value: Mapped[Decimal] = mapped_column(Numeric(precision=15, scale=2))
