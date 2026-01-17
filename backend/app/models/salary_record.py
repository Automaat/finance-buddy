from datetime import UTC, date, datetime
from decimal import Decimal

from sqlalchemy import Boolean, Numeric, String
from sqlalchemy.orm import Mapped, mapped_column

from app.core.database import Base


class SalaryRecord(Base):
    __tablename__ = "salary_records"

    id: Mapped[int] = mapped_column(primary_key=True)
    date: Mapped[date]
    gross_amount: Mapped[Decimal] = mapped_column(Numeric(15, 2))
    contract_type: Mapped[str] = mapped_column(String(50))
    company: Mapped[str] = mapped_column(String(200))
    owner: Mapped[str] = mapped_column(String(100))
    is_active: Mapped[bool] = mapped_column(Boolean, default=True)
    created_at: Mapped[datetime] = mapped_column(default=lambda: datetime.now(UTC))
