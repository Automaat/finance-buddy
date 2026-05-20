"""Add account_id and category columns to goals table.

Revision ID: 0003
Revises: 0002
Create Date: 2026-05-20

"""

from collections.abc import Sequence

import sqlalchemy as sa
from sqlalchemy import inspect

from alembic import op

revision: str = "0003"
down_revision: str | None = "0002"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    inspector = inspect(op.get_bind())
    existing = {col["name"] for col in inspector.get_columns("goals")}

    if "account_id" not in existing:
        op.add_column("goals", sa.Column("account_id", sa.Integer(), nullable=True))
        op.create_foreign_key(
            "fk_goals_account_id",
            "goals",
            "accounts",
            ["account_id"],
            ["id"],
        )

    if "category" not in existing:
        op.add_column("goals", sa.Column("category", sa.String(length=100), nullable=True))


def downgrade() -> None:
    op.drop_constraint("fk_goals_account_id", "goals", type_="foreignkey")
    op.drop_column("goals", "account_id")
    op.drop_column("goals", "category")
