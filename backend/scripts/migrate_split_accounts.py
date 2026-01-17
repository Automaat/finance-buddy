"""
Interactive migration to split Account table into Asset and Account.

This script:
1. Fetches all active accounts
2. Prompts user to classify each as "asset" (physical item) or "account" (financial)
3. Creates Asset records for items classified as assets
4. Updates SnapshotValue FK references from account_id to asset_id
5. Soft deletes Account records that became Assets
"""

import sys
from pathlib import Path

from dotenv import load_dotenv
from sqlalchemy import select
from sqlalchemy.orm import Session

# Add parent directory to path for direct script execution
if __name__ == "__main__":
    sys.path.insert(0, str(Path(__file__).parent.parent))

from app.core.database import engine
from app.models import Account, Asset, SnapshotValue

load_dotenv()


def classify_account(account_name: str) -> str:
    """Prompt user to classify account as asset or account."""
    while True:
        response = input(f"\nClassify '{account_name}'?\n  1. Asset (physical item)\n  2. Account (financial)\nChoice (1/2): ").strip()
        if response == "1":
            return "asset"
        elif response == "2":
            return "account"
        else:
            print("Invalid choice. Please enter 1 or 2.")


def run_migration(dry_run: bool = True) -> None:
    """Run the migration with user prompts."""
    with Session(engine) as db:
        try:
            # Fetch all active accounts
            accounts = db.execute(
                select(Account).where(Account.is_active.is_(True)).order_by(Account.name)
            ).scalars().all()

            if not accounts:
                print("No active accounts found. Nothing to migrate.")
                return

            print(f"\nFound {len(accounts)} active accounts to classify.")
            print("=" * 60)

            # Track classifications
            to_migrate = []
            to_keep = []

            # Classify each account
            for account in accounts:
                classification = classify_account(account.name)
                if classification == "asset":
                    to_migrate.append(account)
                else:
                    to_keep.append(account)

            # Summary
            print("\n" + "=" * 60)
            print(f"\nSummary:")
            print(f"  Accounts to convert to Assets: {len(to_migrate)}")
            for acc in to_migrate:
                print(f"    - {acc.name}")
            print(f"\n  Accounts to keep as Accounts: {len(to_keep)}")
            for acc in to_keep:
                print(f"    - {acc.name}")

            if not to_migrate:
                print("\nNo accounts to migrate. Exiting.")
                return

            # Confirm
            if dry_run:
                print("\n[DRY RUN] No changes will be made.")
                return

            confirm = input("\nProceed with migration? (yes/no): ").strip().lower()
            if confirm != "yes":
                print("Migration cancelled.")
                return

            # Perform migration
            print("\nStarting migration...")
            migrated_count = 0

            for account in to_migrate:
                print(f"  Migrating '{account.name}'...")

                # Create Asset
                asset = Asset(name=account.name, is_active=True)
                db.add(asset)
                db.flush()  # Get ID

                # Update SnapshotValue FK
                snapshot_values = db.execute(
                    select(SnapshotValue).where(SnapshotValue.account_id == account.id)
                ).scalars().all()

                for sv in snapshot_values:
                    sv.asset_id = asset.id
                    sv.account_id = None

                # Soft delete Account
                account.is_active = False

                migrated_count += 1
                print(f"    ✓ Created Asset #{asset.id}, updated {len(snapshot_values)} snapshot values")

            # Commit transaction
            db.commit()
            print(f"\n✅ Migration complete! Migrated {migrated_count} accounts to assets.")

        except Exception as e:
            db.rollback()
            print(f"\n❌ Migration failed: {e}")
            raise


def main():
    """Main entry point."""
    import argparse

    parser = argparse.ArgumentParser(description="Migrate accounts to assets interactively")
    parser.add_argument(
        "--execute",
        action="store_true",
        help="Execute the migration (default is dry-run)",
    )
    args = parser.parse_args()

    dry_run = not args.execute

    if dry_run:
        print("Running in DRY RUN mode. Use --execute to apply changes.\n")
    else:
        print("⚠️  EXECUTING MIGRATION - changes will be permanent!\n")
        confirm = input("Are you sure? Type 'CONFIRM' to proceed: ").strip()
        if confirm != "CONFIRM":
            print("Migration cancelled.")
            return

    run_migration(dry_run=dry_run)


if __name__ == "__main__":
    main()
