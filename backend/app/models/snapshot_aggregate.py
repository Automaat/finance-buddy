from datetime import UTC, date, datetime
from decimal import Decimal
from typing import Any

from sqlalchemy import JSON, Date, DateTime, ForeignKey, Index, Numeric, String, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column

from app.core.database import Base


class SnapshotAggregate(Base):
    """Precomputed aggregate per (snapshot_id, owner).

    Grain: one row per distinct account-owner found in a snapshot's values,
    plus 'Shared' for Asset-table (non-account) entries.

    The 'month' column is denormalized (= snapshot.date with day=1) for fast
    month-bucket grouping in the dashboard. It is NOT part of the uniqueness
    constraint — the unique key is (snapshot_id, owner). Storing month here
    avoids a JOIN to the snapshots table on hot reads.

    Table name is 'snapshot_aggregates' (not 'monthly_aggregates') because the
    grain is per-snapshot, not per-calendar-month. Two snapshots in the same
    calendar month produce two distinct sets of rows.

    Populated by services.snapshot_aggregates.recompute_for_snapshot(), which
    uses the signed-value logic in services.aggregate_spec.compute_aggregates().
    On snapshot delete the FK ondelete=CASCADE removes aggregate rows automatically.
    """

    __tablename__ = "snapshot_aggregates"
    __table_args__ = (
        UniqueConstraint("snapshot_id", "owner", name="uix_snapshot_agg_snapshot_owner"),
        Index("ix_snapshot_aggregates_month", "month"),
    )

    id: Mapped[int] = mapped_column(primary_key=True)
    snapshot_id: Mapped[int] = mapped_column(ForeignKey("snapshots.id", ondelete="CASCADE"))
    month: Mapped[date] = mapped_column(Date)
    owner: Mapped[str] = mapped_column(String(100))
    total_assets: Mapped[Decimal] = mapped_column(Numeric(15, 2))
    total_liabilities: Mapped[Decimal] = mapped_column(Numeric(15, 2))
    net_worth: Mapped[Decimal] = mapped_column(Numeric(15, 2))
    allocation_json: Mapped[Any] = mapped_column(JSON)
    computed_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), default=lambda: datetime.now(UTC)
    )
