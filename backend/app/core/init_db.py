import asyncio
import logging
from decimal import Decimal
from pathlib import Path

from alembic.config import Config
from sqlalchemy import func, inspect, select

from alembic import command
from app.core.database import SessionLocal, engine
from app.core.enums import Wrapper
from app.models import Persona, RetirementLimit

logger = logging.getLogger(__name__)

# Pre-Alembic databases are stamped at this revision, then upgraded forward
# through any newer revisions.
BASELINE_REVISION = "0001"

_BACKEND_ROOT = Path(__file__).resolve().parents[2]


def _alembic_config() -> Config:
    config = Config(str(_BACKEND_ROOT / "alembic.ini"))
    config.set_main_option("script_location", str(_BACKEND_ROOT / "alembic"))
    return config


def _run_migrations() -> None:
    """Bring the schema to head, handling databases created before Alembic.

    A pre-Alembic database already has every table but no ``alembic_version``
    row. It is stamped at the baseline so the baseline migration is not
    re-applied, then upgraded through any newer revisions.
    """
    config = _alembic_config()
    existing_tables = set(inspect(engine).get_table_names())
    if existing_tables and "alembic_version" not in existing_tables:
        logger.info("Pre-Alembic database detected; stamping baseline %s", BASELINE_REVISION)
        command.stamp(config, BASELINE_REVISION)
    command.upgrade(config, "head")


def _seed_defaults() -> None:
    """Seed default personas and retirement limits when the tables are empty."""
    db = SessionLocal()
    try:
        if db.scalar(select(func.count()).select_from(Persona)) == 0:
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

        if db.scalar(select(func.count()).select_from(RetirementLimit)) == 0:
            personas = db.execute(select(Persona)).scalars().all()
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


def _init_db_sync() -> None:
    _run_migrations()
    _seed_defaults()


async def init_db() -> None:
    """Async-first database initialization for the app lifespan.

    Migrations and seeding run on the synchronous SQLAlchemy/Alembic stack,
    so the blocking work is offloaded to a worker thread to keep the startup
    event loop responsive.
    """
    await asyncio.to_thread(_init_db_sync)


if __name__ == "__main__":
    asyncio.run(init_db())
