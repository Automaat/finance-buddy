"""
Migration script to add account_wrapper field and update categories.

Mappings:
- category="ike" → category="fund", account_wrapper="IKE"
- category="ikze" → category="fund", account_wrapper="IKZE"
- category="ppk" → category="ppk", account_wrapper="PPK"
- category="bonds" → category="bond"
- category="stocks" → category="stock"
"""

# ruff: noqa: T201

from sqlalchemy import text

from app.core.database import engine


def migrate() -> None:
    """Execute migration to add account_wrapper column and update categories."""
    with engine.begin() as conn:
        # Add account_wrapper column if it doesn't exist
        print("Adding account_wrapper column...")
        conn.execute(
            text(
                "ALTER TABLE accounts ADD COLUMN IF NOT EXISTS "
                "account_wrapper VARCHAR(50) DEFAULT NULL"
            )
        )

        # Migrate IKE accounts
        print("Migrating IKE accounts...")
        result = conn.execute(
            text(
                "UPDATE accounts SET category = 'fund', account_wrapper = 'IKE' "
                "WHERE category = 'ike'"
            )
        )
        print(f"  Updated {result.rowcount} IKE accounts")

        # Migrate IKZE accounts
        print("Migrating IKZE accounts...")
        result = conn.execute(
            text(
                "UPDATE accounts SET category = 'fund', account_wrapper = 'IKZE' "
                "WHERE category = 'ikze'"
            )
        )
        print(f"  Updated {result.rowcount} IKZE accounts")

        # Migrate PPK accounts
        print("Migrating PPK accounts...")
        result = conn.execute(
            text(
                "UPDATE accounts SET account_wrapper = 'PPK' "
                "WHERE category = 'ppk'"
            )
        )
        print(f"  Updated {result.rowcount} PPK accounts")

        # Migrate bonds → bond
        print("Migrating bonds → bond...")
        result = conn.execute(
            text("UPDATE accounts SET category = 'bond' WHERE category = 'bonds'")
        )
        print(f"  Updated {result.rowcount} bonds accounts")

        # Migrate stocks → stock
        print("Migrating stocks → stock...")
        result = conn.execute(
            text("UPDATE accounts SET category = 'stock' WHERE category = 'stocks'")
        )
        print(f"  Updated {result.rowcount} stocks accounts")

        print("Migration completed successfully!")


if __name__ == "__main__":
    migrate()
