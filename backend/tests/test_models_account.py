from app.models.account import Account


def test_account_creation(test_db_session):
    """Test creating an account with all required fields."""
    account = Account(
        name="Savings Account", type="savings", category="banking", owner="John Doe", currency="USD"
    )
    test_db_session.add(account)
    test_db_session.commit()

    assert account.id is not None
    assert account.name == "Savings Account"
    assert account.type == "savings"
    assert account.category == "banking"
    assert account.owner == "John Doe"
    assert account.currency == "USD"
    assert account.is_active is True
    assert account.created_at is not None


def test_account_default_is_active(test_db_session):
    """Test that is_active defaults to True."""
    account = Account(
        name="Checking Account",
        type="checking",
        category="banking",
        owner="Jane Smith",
        currency="EUR",
    )
    test_db_session.add(account)
    test_db_session.commit()

    assert account.is_active is True


def test_account_explicit_is_active_false(test_db_session):
    """Test setting is_active to False explicitly."""
    account = Account(
        name="Old Account",
        type="savings",
        category="banking",
        owner="John Doe",
        currency="USD",
        is_active=False,
    )
    test_db_session.add(account)
    test_db_session.commit()

    assert account.is_active is False


def test_account_update(test_db_session):
    """Test updating account fields."""
    account = Account(
        name="Original Name", type="savings", category="banking", owner="John Doe", currency="USD"
    )
    test_db_session.add(account)
    test_db_session.commit()

    account.name = "Updated Name"
    account.is_active = False
    test_db_session.commit()

    retrieved = test_db_session.get(Account, account.id)
    assert retrieved.name == "Updated Name"
    assert retrieved.is_active is False


def test_account_multiple_accounts(test_db_session):
    """Test creating multiple accounts."""
    account1 = Account(
        name="Savings", type="savings", category="banking", owner="John Doe", currency="USD"
    )
    account2 = Account(
        name="Checking", type="checking", category="banking", owner="Jane Smith", currency="EUR"
    )
    test_db_session.add_all([account1, account2])
    test_db_session.commit()

    accounts = test_db_session.query(Account).all()
    assert len(accounts) == 2


def test_account_string_length_constraints(test_db_session):
    """Test account with maximum length strings."""
    account = Account(
        name="A" * 255, type="B" * 50, category="C" * 100, owner="D" * 100, currency="E" * 10
    )
    test_db_session.add(account)
    test_db_session.commit()

    retrieved = test_db_session.get(Account, account.id)
    assert len(retrieved.name) == 255
    assert len(retrieved.type) == 50
    assert len(retrieved.category) == 100
    assert len(retrieved.owner) == 100
    assert len(retrieved.currency) == 10


def test_account_deletion(test_db_session):
    """Test deleting an account."""
    account = Account(
        name="To Delete", type="savings", category="banking", owner="John Doe", currency="USD"
    )
    test_db_session.add(account)
    test_db_session.commit()

    account_id = account.id
    test_db_session.delete(account)
    test_db_session.commit()

    deleted = test_db_session.get(Account, account_id)
    assert deleted is None
