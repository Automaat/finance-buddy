"""Add unique constraint to transactions table for active transactions.

This migration prevents duplicate contributions for the same account, date, and type.
Uses a partial unique index to only enforce constraint on active transactions.
"""

from sqlalchemy import text

from app.core.database import SessionLocal


def migrate() -> None:
    db = SessionLocal()
    try:
        # Add unique constraint on (account_id, date, transaction_type) for active transactions
        db.execute(
            text(
                "CREATE UNIQUE INDEX IF NOT EXISTS unique_active_transaction "
                "ON transactions (account_id, date, transaction_type) "
                "WHERE is_active = TRUE"
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
