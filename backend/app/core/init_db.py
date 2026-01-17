from app.core.database import Base, SessionLocal, engine

# Import models to register them with SQLAlchemy Base.metadata
# These imports are required for Base.metadata.create_all() to work
from app.models import Account, Asset, Goal, RetirementLimit, Snapshot, SnapshotValue, Transaction

# Reference imports to satisfy linter (models are registered via import side effect)
_ = (Account, Asset, Goal, RetirementLimit, Snapshot, SnapshotValue, Transaction)


def init_db() -> None:
    Base.metadata.create_all(bind=engine)

    # Seed default retirement limits if none exist
    db = SessionLocal()
    try:
        if db.query(RetirementLimit).count() == 0:
            defaults = [
                # 2026 limits - Marcin
                RetirementLimit(
                    year=2026,
                    account_wrapper="IKE",
                    owner="Marcin",
                    limit_amount=28260.00,
                    notes="",
                ),
                RetirementLimit(
                    year=2026,
                    account_wrapper="IKZE",
                    owner="Marcin",
                    limit_amount=11304.00,
                    notes="",
                ),
                # 2026 limits - Ewa
                RetirementLimit(
                    year=2026,
                    account_wrapper="IKE",
                    owner="Ewa",
                    limit_amount=28260.00,
                    notes="",
                ),
                RetirementLimit(
                    year=2026,
                    account_wrapper="IKZE",
                    owner="Ewa",
                    limit_amount=11304.00,
                    notes="",
                ),
            ]
            db.add_all(defaults)
            db.commit()
    finally:
        db.close()


if __name__ == "__main__":
    init_db()
