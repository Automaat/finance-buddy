from datetime import date
from decimal import Decimal

from sqlalchemy import CheckConstraint, Numeric
from sqlalchemy.orm import Mapped, mapped_column

from app.core.database import Base


class AppConfig(Base):
    __tablename__ = "app_config"

    id: Mapped[int] = mapped_column(primary_key=True)
    birth_date: Mapped[date] = mapped_column(nullable=False)
    retirement_age: Mapped[int] = mapped_column(nullable=False)
    retirement_monthly_salary: Mapped[Decimal] = mapped_column(Numeric(15, 2), nullable=False)
    allocation_real_estate: Mapped[int] = mapped_column(nullable=False)
    allocation_stocks: Mapped[int] = mapped_column(nullable=False)
    allocation_bonds: Mapped[int] = mapped_column(nullable=False)
    allocation_gold: Mapped[int] = mapped_column(nullable=False)
    allocation_commodities: Mapped[int] = mapped_column(nullable=False)
    monthly_expenses: Mapped[Decimal] = mapped_column(Numeric(15, 2), nullable=False, default=0)
    monthly_mortgage_payment: Mapped[Decimal] = mapped_column(
        Numeric(15, 2), nullable=False, default=0
    )
    ppk_employee_rate_marcin: Mapped[Decimal] = mapped_column(
        Numeric(5, 2), nullable=False, default=Decimal("2.0")
    )
    ppk_employer_rate_marcin: Mapped[Decimal] = mapped_column(
        Numeric(5, 2), nullable=False, default=Decimal("1.5")
    )
    ppk_employee_rate_ewa: Mapped[Decimal] = mapped_column(
        Numeric(5, 2), nullable=False, default=Decimal("2.0")
    )
    ppk_employer_rate_ewa: Mapped[Decimal] = mapped_column(
        Numeric(5, 2), nullable=False, default=Decimal("1.5")
    )

    __table_args__ = (CheckConstraint("id = 1", name="single_row"),)
