from app.core.database import Base, engine

# Import models to register them with SQLAlchemy Base.metadata
# These imports are required for Base.metadata.create_all() to work
from app.models import Account, Asset, Goal, Snapshot, SnapshotValue

# Reference imports to satisfy linter (models are registered via import side effect)
_ = (Account, Asset, Goal, Snapshot, SnapshotValue)


def init_db() -> None:
    Base.metadata.create_all(bind=engine)


if __name__ == "__main__":
    init_db()
