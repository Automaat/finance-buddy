from datetime import UTC, date, datetime
from decimal import Decimal

from sqlalchemy import Boolean, Date, ForeignKey, Numeric, String
from sqlalchemy.orm import Mapped, mapped_column

from app.core.database import Base


class Goal(Base):
    __tablename__ = "goals"

    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String(255))
    target_amount: Mapped[Decimal] = mapped_column(Numeric(precision=15, scale=2))
    target_date: Mapped[date] = mapped_column(Date)
    current_amount: Mapped[Decimal] = mapped_column(Numeric(precision=15, scale=2), default=0)
    monthly_contribution: Mapped[Decimal] = mapped_column(Numeric(precision=15, scale=2), default=0)
    is_completed: Mapped[bool] = mapped_column(Boolean, default=False)
    account_id: Mapped[int | None] = mapped_column(
        ForeignKey("accounts.id"), nullable=True, default=None
    )
    category: Mapped[str | None] = mapped_column(String(100), nullable=True, default=None)
    created_at: Mapped[datetime] = mapped_column(default=lambda: datetime.now(UTC))
