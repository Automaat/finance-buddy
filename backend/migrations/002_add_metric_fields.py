"""Add metric-related columns to accounts and app_config tables.

This migration:
1. Adds square_meters column to accounts table (for real estate tracking)
2. Adds monthly_expenses column to app_config table (for emergency fund calculation)
3. Adds monthly_mortgage_payment column to app_config table (for mortgage payoff calculation)
"""

from sqlalchemy import text

from app.core.database import SessionLocal


def migrate() -> None:
    db = SessionLocal()
    try:
        # Add square_meters column to accounts table (nullable)
        db.execute(
            text("ALTER TABLE accounts ADD COLUMN IF NOT EXISTS square_meters NUMERIC(10, 2)")
        )
        db.commit()

        # Add monthly_expenses column to app_config table (default 0)
        db.execute(
            text(
                "ALTER TABLE app_config ADD COLUMN IF NOT EXISTS "
                "monthly_expenses NUMERIC(15, 2) DEFAULT 0"
            )
        )
        db.commit()

        # Add monthly_mortgage_payment column to app_config table (default 0)
        db.execute(
            text(
                "ALTER TABLE app_config ADD COLUMN IF NOT EXISTS "
                "monthly_mortgage_payment NUMERIC(15, 2) DEFAULT 0"
            )
        )
        db.commit()

    except Exception as e:
        db.rollback()
        raise RuntimeError(f"Migration failed: {e}") from e
    finally:
        db.close()


if __name__ == "__main__":
    migrate()
