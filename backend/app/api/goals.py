from typing import Annotated

from fastapi import APIRouter, Depends
from sqlalchemy.orm import Session

from app.core.database import get_db
from app.schemas.goals import GoalCreate, GoalResponse, GoalsListResponse, GoalUpdate
from app.services import goals

router = APIRouter(prefix="/api/goals", tags=["goals"])


@router.get("", response_model=GoalsListResponse)
def get_goals(db: Annotated[Session, Depends(get_db)]) -> GoalsListResponse:
    """Get all goals with progress and projected hit date."""
    return goals.get_all_goals(db)


@router.post("", response_model=GoalResponse, status_code=201)
def create_goal(data: GoalCreate, db: Annotated[Session, Depends(get_db)]) -> GoalResponse:
    """Create a new goal."""
    return goals.create_goal(db, data)


@router.get("/{goal_id}", response_model=GoalResponse)
def get_goal(goal_id: int, db: Annotated[Session, Depends(get_db)]) -> GoalResponse:
    """Get a single goal by ID."""
    return goals.get_goal(db, goal_id)


@router.put("/{goal_id}", response_model=GoalResponse)
def update_goal(
    goal_id: int,
    data: GoalUpdate,
    db: Annotated[Session, Depends(get_db)],
) -> GoalResponse:
    """Update a goal."""
    return goals.update_goal(db, goal_id, data)


@router.delete("/{goal_id}", status_code=204)
def delete_goal(goal_id: int, db: Annotated[Session, Depends(get_db)]) -> None:
    """Delete a goal."""
    goals.delete_goal(db, goal_id)
