"""Add receives_contributions column to accounts table.

This migration adds a column to track which PPK accounts are currently receiving
new contributions (active employment account) vs old accounts that just track value.
"""

from sqlalchemy import text

from app.core.database import SessionLocal


def migrate() -> None:
    db = SessionLocal()
    try:
        # Add receives_contributions column (default True)
        db.execute(
            text(
                "ALTER TABLE accounts ADD COLUMN IF NOT EXISTS "
                "receives_contributions BOOLEAN DEFAULT TRUE"
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
