from datetime import UTC, date, datetime
from decimal import Decimal

from sqlalchemy import Numeric, String, UniqueConstraint
from sqlalchemy.orm import Mapped, mapped_column

from app.core.database import Base


class FxRate(Base):
    __tablename__ = "fx_rates"
    __table_args__ = (UniqueConstraint("date", "currency", name="uq_fx_rates_date_currency"),)

    id: Mapped[int] = mapped_column(primary_key=True)
    date: Mapped[date]
    currency: Mapped[str] = mapped_column(String(3))
    rate_pln: Mapped[Decimal] = mapped_column(Numeric(15, 6))
    created_at: Mapped[datetime] = mapped_column(default=lambda: datetime.now(UTC))
