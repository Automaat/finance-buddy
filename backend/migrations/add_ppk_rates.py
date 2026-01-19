"""Add PPK rate columns to app_config table.

This migration adds 4 columns to track PPK contribution rates:
1. ppk_employee_rate_marcin - Employee contribution rate for Marcin (default 2.0%)
2. ppk_employer_rate_marcin - Employer contribution rate for Marcin (default 1.5%)
3. ppk_employee_rate_ewa - Employee contribution rate for Ewa (default 2.0%)
4. ppk_employer_rate_ewa - Employer contribution rate for Ewa (default 1.5%)
"""

from sqlalchemy import text

from app.core.database import SessionLocal


def migrate() -> None:
    db = SessionLocal()
    try:
        # Add ppk_employee_rate_marcin column (default 2.0%)
        db.execute(
            text(
                "ALTER TABLE app_config ADD COLUMN IF NOT EXISTS "
                "ppk_employee_rate_marcin NUMERIC(5, 2) DEFAULT 2.0"
            )
        )
        db.commit()

        # Add ppk_employer_rate_marcin column (default 1.5%)
        db.execute(
            text(
                "ALTER TABLE app_config ADD COLUMN IF NOT EXISTS "
                "ppk_employer_rate_marcin NUMERIC(5, 2) DEFAULT 1.5"
            )
        )
        db.commit()

        # Add ppk_employee_rate_ewa column (default 2.0%)
        db.execute(
            text(
                "ALTER TABLE app_config ADD COLUMN IF NOT EXISTS "
                "ppk_employee_rate_ewa NUMERIC(5, 2) DEFAULT 2.0"
            )
        )
        db.commit()

        # Add ppk_employer_rate_ewa column (default 1.5%)
        db.execute(
            text(
                "ALTER TABLE app_config ADD COLUMN IF NOT EXISTS "
                "ppk_employer_rate_ewa NUMERIC(5, 2) DEFAULT 1.5"
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
