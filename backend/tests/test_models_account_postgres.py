from app.models.account import Account


def test_account_creation_postgres(test_db_session_postgres):
    """Test creating an account with PostgreSQL backend."""
    account = Account(
        name="Savings Account", type="savings", category="banking", owner="John Doe", currency="USD"
    )
    test_db_session_postgres.add(account)
    test_db_session_postgres.commit()

    assert account.id is not None
    assert account.name == "Savings Account"
    assert account.type == "savings"
    assert account.category == "banking"
    assert account.owner == "John Doe"
    assert account.currency == "USD"
    assert account.is_active is True
    assert account.created_at is not None
