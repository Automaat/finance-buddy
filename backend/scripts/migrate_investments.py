"""
Migrate investment accounts from inwestycje sheet.

Imports detailed investment accounts (IKE, IKZE, PPK, bonds, stocks)
with historical values.
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
from app.models import Account, Snapshot, SnapshotValue

load_dotenv()

EXCEL_FILE = "Finansowa Forteca.xlsx"


def determine_category(account_name: str) -> str:
    """Map account name to category."""
    name_lower = account_name.lower()
    if "ike" in name_lower and "ikze" not in name_lower:
        return "ike"
    if "ikze" in name_lower:
        return "ikze"
    if "ppk" in name_lower:
        return "ppk"
    if "obligacje" in name_lower:
        return "bonds"
    if "akcje" in name_lower:
        return "stocks"
    return "other"


def determine_owner(account_name: str) -> str:
    """Determine owner from account name."""
    name_lower = account_name.lower()
    if "marcin" in name_lower:
        return "Marcin"
    if "ewa" in name_lower:
        return "Ewa"
    return "Shared"


def migrate() -> None:
    """Import investment accounts from inwestycje sheet."""
    print(f"📖 Reading {EXCEL_FILE} (inwestycje sheet)...")

    # Try backend dir first, then project root
    excel_path = Path(__file__).parent.parent / EXCEL_FILE
    if not excel_path.exists():
        excel_path = Path(__file__).parent.parent.parent / EXCEL_FILE
    if not excel_path.exists():
        print(f"❌ Error: {EXCEL_FILE} not found")
        sys.exit(1)

    df = pd.read_excel(excel_path, sheet_name="inwestycje")
    print(f"  Investments sheet: {df.shape[0]} rows, {df.shape[1]} columns")

    db = Session(engine)
    try:
        # Skip metadata columns
        skip_columns = [
            "Data",
            "data",
            "suma inwestycji",
            "Suma inwestycji",
            "zarobek na inwestycji",
            "Zarobek na inwestycji",
        ]

        account_columns = [
            col
            for col in df.columns
            if col not in skip_columns
            and not pd.isna(col)
            and df[col].notna().any()
        ]

        print(f"\n💰 Creating {len(account_columns)} investment accounts...")
        accounts_map: dict[str, int] = {}

        for column in account_columns:
            # Check if account already exists
            existing = db.query(Account).filter_by(name=column).first()
            if existing:
                accounts_map[column] = existing.id
                print(f"  ↻ {column} (exists, id={existing.id})")
                continue

            account = Account(
                name=column,
                type="asset",
                category=determine_category(column),
                owner=determine_owner(column),
                currency="PLN",
            )
            db.add(account)
            db.flush()
            accounts_map[column] = account.id
            print(f"  ✓ {column} → {account.category} ({account.owner})")

        db.commit()
        print(f"  Created {len([k for k, v in accounts_map.items() if v > 44])} new accounts")

        # Import historical values
        print("\n📸 Importing snapshot values...")
        value_count = 0

        for _idx, row in df.iterrows():
            date = row.get("Data") or row.get("data")
            if pd.isna(date):
                continue

            # Find or skip snapshot
            snapshot = db.query(Snapshot).filter_by(date=date).first()
            if not snapshot:
                print(f"  ⚠ Skipping date {date} (no snapshot)")
                continue

            for account_name, account_id in accounts_map.items():
                value = row[account_name]
                if pd.notna(value) and value != 0:
                    # Check if value already exists
                    existing = (
                        db.query(SnapshotValue)
                        .filter_by(snapshot_id=snapshot.id, account_id=account_id)
                        .first()
                    )
                    if existing:
                        continue

                    snapshot_value = SnapshotValue(
                        snapshot_id=snapshot.id,
                        account_id=account_id,
                        value=Decimal(str(abs(value))),
                    )
                    db.add(snapshot_value)
                    value_count += 1

            if value_count % 100 == 0 and value_count > 0:
                print(f"  Processed {value_count} values...")
                db.commit()

        db.commit()
        print(f"  ✓ Imported {value_count} snapshot values")

        print("\n✅ Migration completed!")
        print(f"  📊 {len(accounts_map)} investment accounts")
        print(f"  📸 {value_count} historical values")
    except Exception:
        db.rollback()
        raise
    finally:
        db.close()


if __name__ == "__main__":
    migrate()
