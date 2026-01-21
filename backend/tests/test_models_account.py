from app.models.account import Account
from tests.factories import create_test_account


def test_account_creation(test_db_session):
    """Test creating an account with all required fields."""
    account = create_test_account(
        test_db_session,
        name="Savings Account",
        account_type="savings",
        category="banking",
        owner="John Doe",
        currency="USD",
        purpose="general",
    )

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
    account = create_test_account(
        test_db_session,
        name="Checking Account",
        account_type="checking",
        category="banking",
        owner="Jane Smith",
        currency="EUR",
    )

    assert account.is_active is True


def test_account_explicit_is_active_false(test_db_session):
    """Test setting is_active to False explicitly."""
    account = create_test_account(
        test_db_session,
        name="Old Account",
        account_type="savings",
        category="banking",
        owner="John Doe",
        currency="USD",
        is_active=False,
    )

    assert account.is_active is False


def test_account_update(test_db_session):
    """Test updating account fields."""
    account = create_test_account(
        test_db_session,
        name="Original Name",
        account_type="savings",
        category="banking",
        owner="John Doe",
        currency="USD",
    )

    account.name = "Updated Name"
    account.is_active = False
    test_db_session.commit()

    retrieved = test_db_session.get(Account, account.id)
    assert retrieved.name == "Updated Name"
    assert retrieved.is_active is False


def test_account_multiple_accounts(test_db_session):
    """Test creating multiple accounts."""
    create_test_account(
        test_db_session,
        name="Savings",
        account_type="savings",
        category="banking",
        owner="John Doe",
        currency="USD",
    )
    create_test_account(
        test_db_session,
        name="Checking",
        account_type="checking",
        category="banking",
        owner="Jane Smith",
        currency="EUR",
    )

    accounts = test_db_session.query(Account).all()
    assert len(accounts) == 2


def test_account_string_length_constraints(test_db_session):
    """Test account with maximum length strings."""
    account = create_test_account(
        test_db_session,
        name="A" * 255,
        account_type="B" * 50,
        category="C" * 100,
        owner="D" * 100,
        currency="E" * 10,
        purpose="general",
    )

    retrieved = test_db_session.get(Account, account.id)
    assert len(retrieved.name) == 255
    assert len(retrieved.type) == 50
    assert len(retrieved.category) == 100
    assert len(retrieved.owner) == 100
    assert len(retrieved.currency) == 10


def test_account_deletion(test_db_session):
    """Test deleting an account."""
    account = create_test_account(
        test_db_session,
        name="To Delete",
        account_type="savings",
        category="banking",
        owner="John Doe",
        currency="USD",
    )

    account_id = account.id
    test_db_session.delete(account)
    test_db_session.commit()

    deleted = test_db_session.get(Account, account_id)
    assert deleted is None
