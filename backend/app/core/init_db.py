import logging

from app.core.database import Base, SessionLocal, engine
from app.core.enums import Owner, Wrapper

# Import models to register them with SQLAlchemy Base.metadata
# These imports are required for Base.metadata.create_all() to work
from app.models import (
    Account,
    AppConfig,
    Asset,
    Debt,
    DebtPayment,
    Goal,
    RetirementLimit,
    SalaryRecord,
    Snapshot,
    SnapshotValue,
    Transaction,
)

logger = logging.getLogger(__name__)

# Reference imports to satisfy linter (models are registered via import side effect)
_ = (
    Account,
    AppConfig,
    Asset,
    Debt,
    DebtPayment,
    Goal,
    RetirementLimit,
    SalaryRecord,
    Snapshot,
    SnapshotValue,
    Transaction,
)


def init_db() -> None:
    Base.metadata.create_all(bind=engine)

    # Run migrations
    try:
        import sys
        from pathlib import Path

        # Add migrations directory to path
        migrations_dir = Path(__file__).parent.parent.parent / "migrations"
        sys.path.insert(0, str(migrations_dir))

        # Import and run migrations
        import add_account_purpose  # type: ignore
        import add_metric_fields  # type: ignore
        import add_ppk_rates  # type: ignore
        import add_receives_contributions  # type: ignore
        import add_transaction_unique_constraint  # type: ignore

        add_account_purpose.migrate()  # type: ignore
        add_metric_fields.migrate()  # type: ignore
        add_ppk_rates.migrate()  # type: ignore
        add_receives_contributions.migrate()  # type: ignore
        add_transaction_unique_constraint.migrate()  # type: ignore
    except Exception as e:
        # Migrations may have already run or columns may already exist
        logger.warning("Migration warning (may be expected if already applied): %s", e)

    # Seed default retirement limits if none exist
    db = SessionLocal()
    try:
        if db.query(RetirementLimit).count() == 0:
            defaults = [
                # 2026 limits - Marcin
                RetirementLimit(
                    year=2026,
                    account_wrapper=Wrapper.IKE.value,
                    owner=Owner.MARCIN.value,
                    limit_amount=28260.00,
                    notes="",
                ),
                RetirementLimit(
                    year=2026,
                    account_wrapper=Wrapper.IKZE.value,
                    owner=Owner.MARCIN.value,
                    limit_amount=11304.00,
                    notes="",
                ),
                # 2026 limits - Ewa
                RetirementLimit(
                    year=2026,
                    account_wrapper=Wrapper.IKE.value,
                    owner=Owner.EWA.value,
                    limit_amount=28260.00,
                    notes="",
                ),
                RetirementLimit(
                    year=2026,
                    account_wrapper=Wrapper.IKZE.value,
                    owner=Owner.EWA.value,
                    limit_amount=11304.00,
                    notes="",
                ),
            ]
            db.add_all(defaults)
            db.commit()
    finally:
        db.close()


if __name__ == "__main__":
    init_db()
