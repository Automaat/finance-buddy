from datetime import date

from app.schemas.zus import SalaryHistoryEntry, ZusCalculatorInputs
from app.services.zus_calculator import (
    apply_pension_tax,
    calculate_zus_pension,
    get_life_expectancy,
)


def _make_inputs(**overrides) -> ZusCalculatorInputs:
    """Helper to create test inputs with sensible defaults."""
    defaults = {
        "owner": "Marcin",
        "birth_date": date(1990, 1, 1),
        "gender": "M",
        "retirement_age": 65,
        "current_gross_monthly_salary": 15000.0,
        "salary_growth_rate": 3.0,
        "inflation_rate": 3.0,
        "valorization_rate_konto": 5.0,
        "valorization_rate_subkonto": 4.0,
        "has_ofe": False,
        "kapital_poczatkowy": 0.0,
        "work_start_year": 2015,
        "salary_history": [],
    }
    defaults.update(overrides)
    return ZusCalculatorInputs(**defaults)


class TestBasicProjection:
    def test_returns_projections(self):
        inputs = _make_inputs()
        result = calculate_zus_pension(inputs)

        # Should have projections from work_start_year to retirement year
        retirement_year = 1990 + 65  # 2055
        expected_years = retirement_year - 2015  # 40 years
        assert len(result.yearly_projections) == expected_years

    def test_pension_is_positive(self):
        inputs = _make_inputs()
        result = calculate_zus_pension(inputs)

        assert result.monthly_pension_gross > 0
        assert result.monthly_pension_net > 0
        assert result.total_capital > 0

    def test_pension_reasonable_range(self):
        """Average salary should yield pension roughly 2000-6000 PLN gross."""
        inputs = _make_inputs(current_gross_monthly_salary=10000.0, work_start_year=2015)
        result = calculate_zus_pension(inputs)

        assert result.monthly_pension_gross > 1000
        assert result.monthly_pension_gross < 20000

    def test_net_less_than_gross(self):
        inputs = _make_inputs()
        result = calculate_zus_pension(inputs)

        assert result.monthly_pension_net < result.monthly_pension_gross

    def test_replacement_rate_positive(self):
        inputs = _make_inputs()
        result = calculate_zus_pension(inputs)

        assert 0 < result.replacement_rate < 100

    def test_life_expectancy_returned(self):
        inputs = _make_inputs(retirement_age=65)
        result = calculate_zus_pension(inputs)

        assert result.life_expectancy_months == 220.8


class TestCapApplication:
    def test_cap_triggers_for_high_salary(self):
        """Very high salary should hit 30x cap."""
        inputs = _make_inputs(current_gross_monthly_salary=50000.0)
        result = calculate_zus_pension(inputs)

        # Some future projections should show salary_capped=True
        capped_years = [p for p in result.yearly_projections if p.salary_capped]
        assert len(capped_years) > 0

    def test_cap_does_not_trigger_for_low_salary(self):
        """Normal salary should not hit 30x cap."""
        inputs = _make_inputs(current_gross_monthly_salary=8000.0)
        result = calculate_zus_pension(inputs)

        capped_years = [p for p in result.yearly_projections if p.salary_capped]
        assert len(capped_years) == 0


class TestSuwakBezpieczenstwa:
    def test_suwak_activates_10_years_before_retirement(self):
        """In last 10 years, subkonto contributions should be 0."""
        inputs = _make_inputs()
        result = calculate_zus_pension(inputs)

        retirement_year = 1990 + 65  # 2055
        suwak_start = retirement_year - 10  # 2045

        for p in result.yearly_projections:
            if p.year >= suwak_start:
                assert p.contribution_subkonto == 0.0, (
                    f"Year {p.year}: subkonto contribution should be 0 during suwak"
                )

    def test_suwak_redirects_to_konto(self):
        """During suwak, konto contributions should be higher than normal."""
        inputs = _make_inputs(valorization_rate_konto=0.0, valorization_rate_subkonto=0.0)
        result = calculate_zus_pension(inputs)

        retirement_year = 1990 + 65
        # Check a year before and during suwak
        before_suwak = [p for p in result.yearly_projections if p.year == retirement_year - 11]
        during_suwak = [p for p in result.yearly_projections if p.year == retirement_year - 5]

        if before_suwak and during_suwak:
            # During suwak, konto contribution should include subkonto portion
            assert during_suwak[0].contribution_konto > before_suwak[0].contribution_konto


class TestOfeFlag:
    def test_ofe_changes_subkonto_rate(self):
        """OFE flag should reduce subkonto contribution rate."""
        inputs_no_ofe = _make_inputs(
            has_ofe=False, valorization_rate_konto=0, valorization_rate_subkonto=0
        )
        inputs_ofe = _make_inputs(
            has_ofe=True, valorization_rate_konto=0, valorization_rate_subkonto=0
        )

        result_no_ofe = calculate_zus_pension(inputs_no_ofe)
        result_ofe = calculate_zus_pension(inputs_ofe)

        # First year (before suwak) — subkonto contribution should differ
        first_no_ofe = result_no_ofe.yearly_projections[0]
        first_ofe = result_ofe.yearly_projections[0]

        assert first_no_ofe.contribution_subkonto > first_ofe.contribution_subkonto


class TestSensitivity:
    def test_produces_three_scenarios(self):
        inputs = _make_inputs()
        result = calculate_zus_pension(inputs)

        assert len(result.sensitivity) == 3

    def test_scenarios_ordered(self):
        inputs = _make_inputs()
        result = calculate_zus_pension(inputs)

        labels = [s.label for s in result.sensitivity]
        assert labels == ["Pesymistyczny", "Bazowy", "Optymistyczny"]

    def test_optimistic_higher_than_pessimistic(self):
        inputs = _make_inputs()
        result = calculate_zus_pension(inputs)

        pessimistic = result.sensitivity[0]
        optimistic = result.sensitivity[2]

        assert optimistic.monthly_pension_gross > pessimistic.monthly_pension_gross


class TestTaxCalculation:
    def test_low_pension_mostly_health_tax(self):
        """Pension below 2500/month (30k/year) should only pay health insurance."""
        net = apply_pension_tax(2000.0)
        # 2000 * 12 = 24000 < 30000 threshold → no PIT
        # Health: 24000 * 0.09 = 2160
        expected = (24000 - 2160) / 12
        assert abs(net - expected) < 1.0

    def test_higher_pension_pays_pit(self):
        """Pension above 2500/month should pay PIT on excess."""
        net = apply_pension_tax(5000.0)
        # 60000 gross, health: 5400, PIT: (60000-30000)*0.12 = 3600
        expected = (60000 - 5400 - 3600) / 12
        assert abs(net - expected) < 1.0

    def test_net_always_positive(self):
        net = apply_pension_tax(1000.0)
        assert net > 0


class TestSalaryHistory:
    def test_uses_actual_salary_from_history(self):
        """Past salary entries should be used instead of projections."""
        history = [
            SalaryHistoryEntry(year=2020, annual_gross=120000.0),
            SalaryHistoryEntry(year=2021, annual_gross=132000.0),
        ]
        inputs = _make_inputs(
            salary_history=history,
            work_start_year=2020,
            current_gross_monthly_salary=15000.0,
        )
        result = calculate_zus_pension(inputs)

        # First projection year should use history value
        first = result.yearly_projections[0]
        assert first.year == 2020
        assert first.annual_gross_salary == 120000.0


class TestLifeExpectancy:
    def test_known_age(self):
        assert get_life_expectancy(65) == 220.8
        assert get_life_expectancy(60) == 266.4

    def test_clamp_below(self):
        assert get_life_expectancy(50) == 318.0

    def test_clamp_above(self):
        assert get_life_expectancy(75) == 190.8


class TestKapitalPoczatkowy:
    def test_kapital_poczatkowy_included(self):
        """Kapitał początkowy should be valorized and added to total."""
        inputs_no_kp = _make_inputs(kapital_poczatkowy=0.0)
        inputs_with_kp = _make_inputs(kapital_poczatkowy=100000.0)

        result_no = calculate_zus_pension(inputs_no_kp)
        result_with = calculate_zus_pension(inputs_with_kp)

        assert result_with.total_capital > result_no.total_capital
        assert result_with.kapital_poczatkowy_valorized > 100000.0  # Should grow with valorization
