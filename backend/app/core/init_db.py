import importlib
import logging
import sys
from decimal import Decimal
from pathlib import Path

from app.core.database import Base, SessionLocal, engine
from app.core.enums import Wrapper

# Import models to register them with SQLAlchemy Base.metadata
# These imports are required for Base.metadata.create_all() to work
from app.models import (
    Account,
    AppConfig,
    Asset,
    Debt,
    DebtPayment,
    Goal,
    Persona,
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
    Persona,
    RetirementLimit,
    SalaryRecord,
    Snapshot,
    SnapshotValue,
    Transaction,
)

MIGRATION_MODULES = [
    "add_account_purpose",
    "add_metric_fields",
    "add_ppk_rates",
    "add_receives_contributions",
    "add_transaction_unique_constraint",
]


def init_db() -> None:
    Base.metadata.create_all(bind=engine)

    # Run migrations
    try:
        migrations_dir = Path(__file__).parent.parent.parent / "migrations"
        sys.path.insert(0, str(migrations_dir))

        for module_name in MIGRATION_MODULES:
            mod = importlib.import_module(module_name)
            mod.migrate()
    except Exception as e:
        # Migrations may have already run or columns may already exist
        logger.warning("Migration warning (may be expected if already applied): %s", e)

    # Seed default personas if none exist
    db = SessionLocal()
    try:
        if db.query(Persona).count() == 0:
            default_personas = [
                Persona(
                    name="Marcin",
                    ppk_employee_rate=Decimal("2.0"),
                    ppk_employer_rate=Decimal("1.5"),
                ),
                Persona(
                    name="Ewa",
                    ppk_employee_rate=Decimal("2.0"),
                    ppk_employer_rate=Decimal("1.5"),
                ),
            ]
            db.add_all(default_personas)
            db.commit()

        # Seed default retirement limits if none exist
        if db.query(RetirementLimit).count() == 0:
            personas = db.query(Persona).all()
            defaults = []
            for persona in personas:
                defaults.append(
                    RetirementLimit(
                        year=2026,
                        account_wrapper=Wrapper.IKE.value,
                        owner=persona.name,
                        limit_amount=28260.00,
                        notes="",
                    )
                )
                defaults.append(
                    RetirementLimit(
                        year=2026,
                        account_wrapper=Wrapper.IKZE.value,
                        owner=persona.name,
                        limit_amount=11304.00,
                        notes="",
                    )
                )
            db.add_all(defaults)
            db.commit()
    finally:
        db.close()


if __name__ == "__main__":
    init_db()
