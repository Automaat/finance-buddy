"""Add cpi_index table for Polish CPI data from GUS BDL.

Revision ID: 0002
Revises: 0001
Create Date: 2026-05-20

"""

from collections.abc import Sequence

import sqlalchemy as sa

from alembic import op

revision: str = "0002"
down_revision: str | None = "0001"
branch_labels: str | Sequence[str] | None = None
depends_on: str | Sequence[str] | None = None


def upgrade() -> None:
    op.create_table(
        "cpi_index",
        sa.Column("year", sa.Integer(), primary_key=True),
        sa.Column("yoy_rate", sa.Numeric(8, 4), nullable=False),
        sa.Column("source", sa.String(64), nullable=False, server_default="GUS-BDL-217230"),
        sa.Column(
            "fetched_at",
            sa.DateTime(timezone=True),
            nullable=False,
            server_default=sa.func.now(),
        ),
    )


def downgrade() -> None:
    op.drop_table("cpi_index")
