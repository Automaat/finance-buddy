from app.core.database import Base, engine
from app.models import Account, Goal, Snapshot, SnapshotValue  # noqa: F401


def init_db() -> None:
    Base.metadata.create_all(bind=engine)


if __name__ == "__main__":
    init_db()
