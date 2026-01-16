from datetime import date
from decimal import Decimal

from app.models.goal import Goal


def test_goal_creation(test_db_session):
    """Test creating a goal with all required fields."""
    goal = Goal(
        name="Emergency Fund",
        target_amount=Decimal("10000.00"),
        target_date=date(2025, 12, 31)
    )
    test_db_session.add(goal)
    test_db_session.commit()

    assert goal.id is not None
    assert goal.name == "Emergency Fund"
    assert goal.target_amount == Decimal("10000.00")
    assert goal.target_date == date(2025, 12, 31)
    assert goal.current_amount == 0
    assert goal.monthly_contribution == 0
    assert goal.is_completed is False
    assert goal.created_at is not None


def test_goal_with_current_amount(test_db_session):
    """Test creating a goal with existing progress."""
    goal = Goal(
        name="Vacation Fund",
        target_amount=Decimal("5000.00"),
        target_date=date(2025, 6, 30),
        current_amount=Decimal("1500.00")
    )
    test_db_session.add(goal)
    test_db_session.commit()

    assert goal.current_amount == Decimal("1500.00")


def test_goal_with_monthly_contribution(test_db_session):
    """Test creating a goal with monthly contribution set."""
    goal = Goal(
        name="Down Payment",
        target_amount=Decimal("50000.00"),
        target_date=date(2027, 12, 31),
        monthly_contribution=Decimal("500.00")
    )
    test_db_session.add(goal)
    test_db_session.commit()

    assert goal.monthly_contribution == Decimal("500.00")


def test_goal_mark_completed(test_db_session):
    """Test marking a goal as completed."""
    goal = Goal(
        name="Short Term Goal",
        target_amount=Decimal("1000.00"),
        target_date=date(2025, 3, 31),
        current_amount=Decimal("1000.00"),
        is_completed=True
    )
    test_db_session.add(goal)
    test_db_session.commit()

    assert goal.is_completed is True


def test_goal_update_progress(test_db_session):
    """Test updating goal progress."""
    goal = Goal(
        name="Investment Goal",
        target_amount=Decimal("20000.00"),
        target_date=date(2026, 12, 31),
        current_amount=Decimal("5000.00")
    )
    test_db_session.add(goal)
    test_db_session.commit()

    goal.current_amount = Decimal("7500.00")
    test_db_session.commit()

    retrieved = test_db_session.get(Goal, goal.id)
    assert retrieved.current_amount == Decimal("7500.00")


def test_goal_decimal_precision(test_db_session):
    """Test that goal amounts maintain decimal precision."""
    goal = Goal(
        name="Precise Goal",
        target_amount=Decimal("12345.67"),
        target_date=date(2025, 12, 31),
        current_amount=Decimal("6789.12"),
        monthly_contribution=Decimal("234.56")
    )
    test_db_session.add(goal)
    test_db_session.commit()

    retrieved = test_db_session.get(Goal, goal.id)
    assert retrieved.target_amount == Decimal("12345.67")
    assert retrieved.current_amount == Decimal("6789.12")
    assert retrieved.monthly_contribution == Decimal("234.56")


def test_goal_large_amounts(test_db_session):
    """Test goal with large decimal amounts."""
    goal = Goal(
        name="Large Goal",
        target_amount=Decimal("9999999999999.99"),
        target_date=date(2030, 12, 31)
    )
    test_db_session.add(goal)
    test_db_session.commit()

    retrieved = test_db_session.get(Goal, goal.id)
    assert retrieved.target_amount == Decimal("9999999999999.99")


def test_goal_multiple_goals(test_db_session):
    """Test creating multiple goals."""
    goal1 = Goal(
        name="Goal 1",
        target_amount=Decimal("5000.00"),
        target_date=date(2025, 6, 30)
    )
    goal2 = Goal(
        name="Goal 2",
        target_amount=Decimal("10000.00"),
        target_date=date(2026, 12, 31)
    )
    test_db_session.add_all([goal1, goal2])
    test_db_session.commit()

    goals = test_db_session.query(Goal).all()
    assert len(goals) == 2


def test_goal_deletion(test_db_session):
    """Test deleting a goal."""
    goal = Goal(
        name="To Delete",
        target_amount=Decimal("1000.00"),
        target_date=date(2025, 12, 31)
    )
    test_db_session.add(goal)
    test_db_session.commit()

    goal_id = goal.id
    test_db_session.delete(goal)
    test_db_session.commit()

    deleted = test_db_session.get(Goal, goal_id)
    assert deleted is None


def test_goal_name_length(test_db_session):
    """Test goal with maximum name length."""
    goal = Goal(
        name="A" * 255,
        target_amount=Decimal("1000.00"),
        target_date=date(2025, 12, 31)
    )
    test_db_session.add(goal)
    test_db_session.commit()

    retrieved = test_db_session.get(Goal, goal.id)
    assert len(retrieved.name) == 255
