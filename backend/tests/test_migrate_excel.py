"""Tests for Excel migration script helper functions."""

from scripts.migrate_excel import determine_category, determine_owner, determine_type


class TestDetermineType:
    """Test account type determination (asset vs liability)."""

    def test_asset_accounts(self) -> None:
        """Regular accounts should be classified as assets."""
        assert determine_type("Konto bankowe") == "asset"
        assert determine_type("IKE Marcin") == "asset"
        assert determine_type("Oszczędności") == "asset"
        assert determine_type("Mieszkanie") == "asset"

    def test_liability_accounts(self) -> None:
        """Loans and installments should be classified as liabilities."""
        assert determine_type("Raty 0%") == "liability"
        assert determine_type("Hipoteka") == "liability"
        assert determine_type("raty 0") == "liability"


class TestDetermineCategory:
    """Test account category mapping."""

    def test_retirement_accounts(self) -> None:
        """IKE, IKZE, PPK should be categorized correctly."""
        assert determine_category("IKE Marcin") == "ike"
        assert determine_category("IKZE Ewa") == "ikze"
        assert determine_category("PPK") == "ppk"

    def test_bank_accounts(self) -> None:
        """Bank accounts and savings should be categorized as bank."""
        assert determine_category("Konto bankowe") == "bank"
        assert determine_category("Oszczędności") == "bank"
        assert determine_category("Oszczednosc") == "bank"

    def test_real_estate(self) -> None:
        """Real estate should be categorized correctly."""
        assert determine_category("Mieszkanie") == "real_estate"
        assert determine_category("Działka") == "real_estate"
        assert determine_category("Dzialka") == "real_estate"

    def test_investments(self) -> None:
        """Investment types should be categorized correctly."""
        assert determine_category("Obligacje") == "bonds"
        assert determine_category("Akcje") == "stocks"
        assert determine_category("Fundusz") == "fund"
        assert determine_category("ETF") == "etf"

    def test_liabilities(self) -> None:
        """Liabilities should be categorized correctly."""
        assert determine_category("Hipoteka") == "mortgage"
        assert determine_category("Raty 0%") == "installment"

    def test_vehicles(self) -> None:
        """Vehicles should be categorized correctly."""
        assert determine_category("Samochód") == "vehicle"
        assert determine_category("Samochod") == "vehicle"

    def test_other_category(self) -> None:
        """Unknown accounts should default to 'other'."""
        assert determine_category("Unknown Account") == "other"


class TestDetermineOwner:
    """Test account owner determination."""

    def test_marcin_ownership(self) -> None:
        """Accounts with 'Marcin' should be assigned to Marcin."""
        assert determine_owner("IKE Marcin") == "Marcin"
        assert determine_owner("Konto Marcin") == "Marcin"

    def test_ewa_ownership(self) -> None:
        """Accounts with 'Ewa' should be assigned to Ewa."""
        assert determine_owner("IKZE Ewa") == "Ewa"
        assert determine_owner("Konto Ewa") == "Ewa"

    def test_shared_ownership(self) -> None:
        """Accounts without owner name should be marked as Shared."""
        assert determine_owner("Mieszkanie") == "Shared"
        assert determine_owner("Konto bankowe") == "Shared"
        assert determine_owner("Obligacje") == "Shared"
