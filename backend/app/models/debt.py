from datetime import UTC, date, datetime
from decimal import Decimal

from sqlalchemy import Boolean, ForeignKey, Numeric, String, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column

from app.core.database import Base


class Debt(Base):
    __tablename__ = "debts"
    __table_args__ = (UniqueConstraint("account_id", name="uq_debt_account"),)

    id: Mapped[int] = mapped_column(primary_key=True)
    account_id: Mapped[int] = mapped_column(ForeignKey("accounts.id", ondelete="CASCADE"))
    name: Mapped[str] = mapped_column(String(255))
    debt_type: Mapped[str] = mapped_column(String(50))
    start_date: Mapped[date]
    initial_amount: Mapped[Decimal] = mapped_column(Numeric(15, 2))
    interest_rate: Mapped[Decimal] = mapped_column(Numeric(5, 2))
    currency: Mapped[str] = mapped_column(String(10))
    notes: Mapped[str | None] = mapped_column(String(500), nullable=True, default=None)
    is_active: Mapped[bool] = mapped_column(Boolean, default=True)
    created_at: Mapped[datetime] = mapped_column(default=lambda: datetime.now(UTC))
