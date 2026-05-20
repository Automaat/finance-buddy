from datetime import UTC, date, datetime, timedelta
from decimal import Decimal

from fastapi import HTTPException
from sqlalchemy import select
from sqlalchemy.orm import Session

from app.models import Account, Goal
from app.schemas.goals import (
    GoalCreate,
    GoalResponse,
    GoalsListResponse,
    GoalUpdate,
)
from app.utils.db_helpers import get_or_404


def _project_hit_date(
    target_amount: Decimal,
    current_amount: Decimal,
    monthly_contribution: Decimal,
    is_completed: bool,
) -> date | None:
    """Project the date when a goal will be hit at current contribution rate.

    Returns None if already completed, target already reached, or no contribution.
    """
    today = datetime.now(UTC).date()
    if is_completed or current_amount >= target_amount:
        return today
    if monthly_contribution <= 0:
        return None

    remaining = target_amount - current_amount
    # Ceil division for whole months needed
    months_needed = int((remaining + monthly_contribution - Decimal("0.01")) / monthly_contribution)
    # Approximate days per month (30.44) for projected calendar date
    return today + timedelta(days=int(months_needed * 30.44))


def _to_response(goal: Goal, account_name: str | None) -> GoalResponse:
    target = float(goal.target_amount)
    current = float(goal.current_amount)
    remaining = max(0.0, target - current)
    progress = min(100.0, (current / target * 100.0)) if target > 0 else 0.0
    projected = _project_hit_date(
        goal.target_amount, goal.current_amount, goal.monthly_contribution, goal.is_completed
    )
    return GoalResponse(
        id=goal.id,
        name=goal.name,
        target_amount=target,
        target_date=goal.target_date,
        current_amount=current,
        monthly_contribution=float(goal.monthly_contribution),
        is_completed=goal.is_completed,
        account_id=goal.account_id,
        account_name=account_name,
        category=goal.category,
        created_at=goal.created_at,
        progress_percent=progress,
        remaining_amount=remaining,
        projected_hit_date=projected,
    )


def _get_account_names(db: Session, account_ids: list[int]) -> dict[int, str]:
    if not account_ids:
        return {}
    rows = db.execute(select(Account.id, Account.name).where(Account.id.in_(account_ids))).all()
    return {row[0]: row[1] for row in rows}


def _validate_account(db: Session, account_id: int | None) -> None:
    if account_id is None:
        return
    account = db.get(Account, account_id)
    if not account:
        raise HTTPException(status_code=404, detail=f"Account with id {account_id} not found")


def get_all_goals(db: Session) -> GoalsListResponse:
    """Get all goals with progress and projected hit date."""
    goals = db.execute(select(Goal).order_by(Goal.target_date)).scalars().all()

    account_ids = [g.account_id for g in goals if g.account_id is not None]
    account_names = _get_account_names(db, account_ids)

    items = [
        _to_response(g, account_names.get(g.account_id) if g.account_id else None) for g in goals
    ]
    completed = sum(1 for g in goals if g.is_completed)

    return GoalsListResponse(goals=items, total_count=len(items), completed_count=completed)


def get_goal(db: Session, goal_id: int) -> GoalResponse:
    """Get a single goal by ID."""
    goal = get_or_404(db, Goal, goal_id)
    account_name = None
    if goal.account_id:
        account = db.get(Account, goal.account_id)
        account_name = account.name if account else None
    return _to_response(goal, account_name)


def create_goal(db: Session, data: GoalCreate) -> GoalResponse:
    """Create a new goal."""
    _validate_account(db, data.account_id)

    goal = Goal(
        name=data.name,
        target_amount=Decimal(str(data.target_amount)),
        target_date=data.target_date,
        current_amount=Decimal(str(data.current_amount)),
        monthly_contribution=Decimal(str(data.monthly_contribution)),
        is_completed=data.is_completed,
        account_id=data.account_id,
        category=data.category.value if data.category else None,
    )
    db.add(goal)
    db.commit()
    db.refresh(goal)

    account_name = None
    if goal.account_id:
        account = db.get(Account, goal.account_id)
        account_name = account.name if account else None
    return _to_response(goal, account_name)


def update_goal(db: Session, goal_id: int, data: GoalUpdate) -> GoalResponse:
    """Update an existing goal."""
    goal = get_or_404(db, Goal, goal_id)
    fields_set = getattr(data, "model_fields_set", set())

    if data.name is not None:
        goal.name = data.name
    if data.target_amount is not None:
        goal.target_amount = Decimal(str(data.target_amount))
    if data.target_date is not None:
        goal.target_date = data.target_date
    if data.current_amount is not None:
        goal.current_amount = Decimal(str(data.current_amount))
    if data.monthly_contribution is not None:
        goal.monthly_contribution = Decimal(str(data.monthly_contribution))
    if data.is_completed is not None:
        goal.is_completed = data.is_completed
    if "account_id" in fields_set:
        _validate_account(db, data.account_id)
        goal.account_id = data.account_id
    if "category" in fields_set:
        goal.category = data.category.value if data.category else None

    db.commit()
    db.refresh(goal)

    account_name = None
    if goal.account_id:
        account = db.get(Account, goal.account_id)
        account_name = account.name if account else None
    return _to_response(goal, account_name)


def delete_goal(db: Session, goal_id: int) -> None:
    """Hard delete a goal."""
    goal = get_or_404(db, Goal, goal_id)
    db.delete(goal)
    db.commit()
