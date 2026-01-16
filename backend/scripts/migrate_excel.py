"""
Excel to PostgreSQL migration using pandas.

Migrates data from two sheets:
- wartosc_netto: net worth snapshots over time (accounts + historical values)
- inwestycje: investment details (IKE, IKZE, PPK, stocks, bonds)

pandas operations demonstrated:
- read_excel(): Load Excel sheets
- DataFrame.columns: Access column names
- DataFrame.iterrows(): Iterate rows
- pd.isna(), pd.notna(): Handle missing values
- Boolean indexing: df[condition]
- String operations: .str.lower(), .str.contains()
"""

import sys
from decimal import Decimal
from pathlib import Path

import pandas as pd
from dotenv import load_dotenv
from sqlalchemy.orm import Session

# Add parent directory to path for direct script execution
if __name__ == "__main__":
    sys.path.insert(0, str(Path(__file__).parent.parent))

from app.core.database import engine
from app.models import Account, Snapshot, SnapshotValue

load_dotenv()

EXCEL_FILE = "Finansowa Forteca.xlsx"


def determine_type(account_name: str) -> str:
    """Determine if account is asset or liability using pandas-style logic."""
    liabilities = ["raty 0", "hipoteka"]
    return "liability" if any(lib in account_name.lower() for lib in liabilities) else "asset"


def determine_category(account_name: str) -> str:
    """Map account name to category - could use pandas.Series.map() for bulk operations."""
    name_lower = account_name.lower()
    category_map = {
        "ike": "ike",
        "ikze": "ikze",
        "ppk": "ppk",
        "konto": "bank",
        "oszczednosc": "bank",
        "oszczędności": "bank",
        "mieszkanie": "real_estate",
        "działka": "real_estate",
        "dzialka": "real_estate",
        "samochod": "vehicle",
        "samochód": "vehicle",
        "obligacje": "bonds",
        "akcje": "stocks",
        "hipoteka": "mortgage",
        "raty": "installment",
        "fundusz": "fund",
        "etf": "etf",
    }

    for key, value in category_map.items():
        if key in name_lower:
            return value
    return "other"


def determine_owner(account_name: str) -> str:
    """Determine owner - demonstrates pattern matching in pandas workflows."""
    name_lower = account_name.lower()
    if "marcin" in name_lower:
        return "Marcin"
    if "ewa" in name_lower:
        return "Ewa"
    return "Shared"


def migrate() -> None:
    """Main migration - demonstrates pandas Excel I/O and data transformation."""
    # pandas: read_excel() - Load multiple sheets
    print(f"📖 Reading {EXCEL_FILE}...")

    excel_path = Path(__file__).parent.parent / EXCEL_FILE
    if not excel_path.exists():
        print(f"❌ Error: {EXCEL_FILE} not found in {excel_path.parent}")
        print(f"   Please place the Excel file in: {excel_path.parent}")
        sys.exit(1)

    df_net_worth = pd.read_excel(excel_path, sheet_name="wartosc_netto")
    df_investments = pd.read_excel(excel_path, sheet_name="inwestycje")

    print(f"  Net worth sheet: {df_net_worth.shape[0]} rows, {df_net_worth.shape[1]} columns")
    print(f"  Investments sheet: {df_investments.shape[0]} rows, {df_investments.shape[1]} columns")

    # SQLAlchemy session
    db = Session(engine)
    try:
        # 1. Create accounts from net worth DataFrame columns
        print("\n💰 Creating accounts from net worth sheet...")
        accounts_map: dict[str, int] = {}
        skip_columns = ["Data", "wartość netto", "wartosc netto"]

        # pandas: .columns returns Index of column names
        account_columns = [col for col in df_net_worth.columns if col not in skip_columns]

        for column in account_columns:
            # pandas: check if column has any non-NaN values
            if df_net_worth[column].notna().any():
                account = Account(
                    name=column,
                    type=determine_type(column),
                    category=determine_category(column),
                    owner=determine_owner(column),
                    currency="PLN",
                )
                db.add(account)
                db.flush()  # Get ID before commit
                accounts_map[column] = account.id
                print(f"  ✓ {column} → {account.category} ({account.owner})")

        db.commit()
        print(f"  Created {len(accounts_map)} accounts")

        # 2. Create snapshots from DataFrame rows
        print("\n📸 Creating snapshots...")
        snapshot_count = 0

        # pandas: .iterrows() - iterate over DataFrame rows
        for _idx, row in df_net_worth.iterrows():
            date = row["Data"]

            # pandas: pd.isna() - check for NaN/None values
            if pd.isna(date):
                continue

            # Create snapshot or use existing for duplicate dates
            existing = db.query(Snapshot).filter_by(date=date).first()
            if existing:
                snapshot = existing
            else:
                snapshot = Snapshot(date=date)
                db.add(snapshot)
                db.flush()

            # pandas: Access row values by column name
            for account_name, account_id in accounts_map.items():
                value = row[account_name]

                # pandas: pd.notna() - check value exists
                if pd.notna(value) and value != 0:
                    # Store absolute value - account type determines if asset/liability
                    snapshot_value = SnapshotValue(
                        snapshot_id=snapshot.id,
                        account_id=account_id,
                        value=Decimal(str(abs(value))),
                    )
                    db.add(snapshot_value)

            snapshot_count += 1
            if snapshot_count % 10 == 0:
                print(f"  Processed {snapshot_count} snapshots...")
                db.commit()  # Commit in batches

        db.commit()
        print(f"  ✓ Created {snapshot_count} snapshots")

        # 3. Process investment sheet (informational - structure may vary)
        print("\n📊 Investment sheet analysis...")
        print(f"  Columns: {list(df_investments.columns)}")
        print(f"  Shape: {df_investments.shape}")
        print("  (Investment data structure varies - manual review recommended)")

        print("\n✅ Migration completed successfully!")
        print(f"  📊 {len(accounts_map)} accounts")
        print(f"  📸 {snapshot_count} snapshots")
    except Exception:
        db.rollback()
        raise
    finally:
        db.close()


if __name__ == "__main__":
    migrate()
