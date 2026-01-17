"""
Migrate mortgage payment history from wplaty_hipoteka sheet.

Imports 106 historical mortgage payments.
"""

import sys
from decimal import Decimal
from pathlib import Path

import pandas as pd
from dotenv import load_dotenv
from sqlalchemy.orm import Session

if __name__ == "__main__":
    sys.path.insert(0, str(Path(__file__).parent.parent))

from app.core.database import engine
from app.models import Account, DebtPayment

load_dotenv()

EXCEL_FILE = "Finansowa Forteca.xlsx"


def migrate() -> None:
    """Import mortgage payments from wplaty_hipoteka sheet."""
    print(f"📖 Reading {EXCEL_FILE} (wplaty_hipoteka sheet)...")

    # Try backend dir first, then project root
    excel_path = Path(__file__).parent.parent / EXCEL_FILE
    if not excel_path.exists():
        excel_path = Path(__file__).parent.parent.parent / EXCEL_FILE
    if not excel_path.exists():
        print(f"❌ Error: {EXCEL_FILE} not found")
        sys.exit(1)

    df = pd.read_excel(excel_path, sheet_name="wplaty_hipoteka")
    print(f"  Payments sheet: {df.shape[0]} rows, {df.shape[1]} columns")

    db = Session(engine)
    try:
        # Find mortgage liability account
        print("\n🔍 Finding mortgage liability account...")
        account = (
            db.query(Account)
            .filter_by(type="liability", category="mortgage", is_active=True)
            .first()
        )

        if not account:
            print("❌ No active mortgage account found")
            print("   Create a mortgage account first before importing payments")
            sys.exit(1)

        print(f"  ✓ Using account: {account.name} (id={account.id})")

        # Import payments
        print("\n💳 Importing debt payments...")
        payment_count = 0
        skipped_count = 0

        for _idx, row in df.iterrows():
            date = row.get("Data")
            amount = row.get("Kwota")
            owner = row.get("Kto wpłacił")

            # Skip rows with missing data
            if pd.isna(date) or pd.isna(amount):
                skipped_count += 1
                continue

            # Normalize owner name
            if pd.notna(owner):
                owner_str = str(owner).strip()
                if owner_str == "Marcin":
                    owner = "Marcin"
                elif owner_str == "Ewa":
                    owner = "Ewa"
                else:
                    owner = "Shared"
            else:
                owner = "Shared"

            # Check for duplicate (by date + amount + owner)
            existing = (
                db.query(DebtPayment)
                .filter_by(
                    account_id=account.id,
                    date=date,
                    amount=Decimal(str(amount)),
                    owner=owner,
                )
                .first()
            )
            if existing:
                skipped_count += 1
                continue

            # Create payment
            payment = DebtPayment(
                account_id=account.id,
                amount=Decimal(str(amount)),
                date=date,
                owner=owner,
                is_active=True,
            )
            db.add(payment)
            payment_count += 1

            # Batch commits
            if payment_count % 20 == 0:
                db.commit()
                print(f"  Processed {payment_count} payments...")

        db.commit()
        print("\n✅ Migration completed!")
        print(f"  ✓ Imported {payment_count} debt payments")
        print(f"  ↻ Skipped {skipped_count} rows (duplicates or missing data)")
    except Exception:
        db.rollback()
        raise
    finally:
        db.close()


if __name__ == "__main__":
    migrate()
