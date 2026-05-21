from datetime import UTC, date, datetime
from decimal import Decimal
from typing import Any

from sqlalchemy import JSON, Boolean, Integer, Numeric, String
from sqlalchemy.orm import Mapped, mapped_column

from app.core.database import Base


class EquityGrant(Base):
    __tablename__ = "equity_grants"

    id: Mapped[int] = mapped_column(primary_key=True)
    grant_date: Mapped[date]
    type: Mapped[str] = mapped_column(String(20))
    company: Mapped[str] = mapped_column(String(200))
    owner: Mapped[str] = mapped_column(String(100))
    total_shares: Mapped[int] = mapped_column(Integer)
    strike_price: Mapped[Decimal | None] = mapped_column(Numeric(15, 4), nullable=True)
    currency: Mapped[str] = mapped_column(String(3), default="USD")

    vest_start_date: Mapped[date]
    vest_cliff_months: Mapped[int] = mapped_column(Integer, default=0)
    vest_total_months: Mapped[int] = mapped_column(Integer)
    vest_frequency: Mapped[str] = mapped_column(String(20))
    vest_custom_schedule: Mapped[list[dict[str, Any]] | None] = mapped_column(JSON, nullable=True)

    requires_liquidity_event: Mapped[bool] = mapped_column(Boolean, default=False)
    liquidity_event_date: Mapped[date | None] = mapped_column(nullable=True)

    tax_treatment: Mapped[str] = mapped_column(String(30), default="capital_gains_19")

    notes: Mapped[str | None] = mapped_column(String(500), default=None)
    is_active: Mapped[bool] = mapped_column(Boolean, default=True)
    created_at: Mapped[datetime] = mapped_column(default=lambda: datetime.now(UTC))
