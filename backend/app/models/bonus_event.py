from datetime import UTC, date, datetime
from decimal import Decimal

from sqlalchemy import Boolean, Numeric, String
from sqlalchemy.orm import Mapped, mapped_column

from app.core.database import Base


class BonusEvent(Base):
    __tablename__ = "bonus_events"

    id: Mapped[int] = mapped_column(primary_key=True)
    date: Mapped[date]
    amount: Mapped[Decimal] = mapped_column(Numeric(15, 2))
    currency: Mapped[str] = mapped_column(String(3), default="PLN")
    type: Mapped[str] = mapped_column(String(20))
    company: Mapped[str] = mapped_column(String(200))
    owner: Mapped[str] = mapped_column(String(100))
    contract_type: Mapped[str] = mapped_column(String(50))
    notes: Mapped[str | None] = mapped_column(String(500), default=None)
    is_active: Mapped[bool] = mapped_column(Boolean, default=True)
    created_at: Mapped[datetime] = mapped_column(default=lambda: datetime.now(UTC))
