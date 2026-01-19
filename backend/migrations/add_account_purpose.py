"""Add purpose column to accounts table with smart defaults.

This migration:
1. Adds purpose column to accounts table
2. Sets purpose='retirement' for accounts with IKE/IKZE/PPK wrapper
3. Sets purpose='general' for all other accounts
"""

from sqlalchemy import text

from app.core.database import SessionLocal


def migrate() -> None:
    db = SessionLocal()
    try:
        # Add purpose column with temporary default
        db.execute(
            text(
                "ALTER TABLE accounts ADD COLUMN IF NOT EXISTS "
                "purpose VARCHAR(50) DEFAULT 'general'"
            )
        )
        db.commit()

        # Update purpose for retirement accounts (with IKE/IKZE/PPK wrapper)
        db.execute(
            text(
                """
                UPDATE accounts
                SET purpose = 'retirement'
                WHERE account_wrapper IN ('IKE', 'IKZE', 'PPK')
                """
            )
        )
        db.commit()

        # Remove default constraint (purpose should be required but no default in schema)
        db.execute(text("ALTER TABLE accounts ALTER COLUMN purpose DROP DEFAULT"))
        db.commit()

    except Exception as e:
        db.rollback()
        raise RuntimeError(f"Migration failed: {e}") from e
    finally:
        db.close()


if __name__ == "__main__":
    migrate()
