package retirement

import (
	"context"
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

// contractTypeUOP is the salary contract type ("umowa o pracę") PPK generation
// requires — only employment contracts mint contributions, manual or auto.
const contractTypeUOP = "UOP"

// ErrNotUOP marks an owner whose latest salary record isn't an employment
// contract, so PPK contributions must not be generated.
var ErrNotUOP = errors.New("salary contract type is not UOP")

// GenerateOptions carries the optional government-subsidy opt-ins.
type GenerateOptions struct {
	IncludeWelcome bool
	IncludeAnnual  bool
}

// GenerateOutcome reports the computed amounts and the insert result, for the
// caller to render or log.
type GenerateOutcome struct {
	Gross       decimal.Decimal
	EmployeeAmt decimal.Decimal
	EmployerAmt decimal.Decimal
	Subsidy     SubsidyConfig
	Result      PPKContributionResult
}

// GeneratePPK runs the contribution-generation flow for one owner and month:
// latest UOP salary -> PPK rates -> amounts -> active PPK account -> idempotent
// insert. It returns sentinel errors (ErrNoSalary, ErrNotUOP, ErrUserNotFound,
// ErrNoPPKAccount, ErrContributionsExist) for the caller to map. Shared by the
// HTTP handler and the monthly scheduler.
func GeneratePPK(ctx context.Context, store *Store, ownerUserID *int, month, year int, opts GenerateOptions) (GenerateOutcome, error) {
	out := GenerateOutcome{}
	firstDay := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	gross, contractType, err := store.CurrentSalaryRecordFor(ctx, ownerUserID, firstDay)
	if err != nil {
		return out, err
	}
	if contractType != contractTypeUOP {
		return out, ErrNotUOP
	}
	employeeRate, employerRate, err := store.UserPPKRates(ctx, ownerUserID)
	if err != nil {
		return out, err
	}
	employeeAmt, employerAmt := computePPKContributionAmounts(gross, employeeRate, employerRate)
	accountID, err := store.ActivePPKAccountForOwner(ctx, ownerUserID)
	if err != nil {
		return out, err
	}
	lastDay := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC)
	subsidy := SubsidyFor(year)
	contrib := PPKContribution{
		AccountID:   accountID,
		EmployeeAmt: employeeAmt,
		EmployerAmt: employerAmt,
		Date:        lastDay,
		OwnerUserID: ownerUserID,
	}
	if opts.IncludeWelcome {
		contrib.WelcomeAmt = subsidy.WelcomeAmount
	}
	if opts.IncludeAnnual {
		contrib.AnnualAmt = subsidy.AnnualAmount
	}
	result, err := store.InsertPPKContributions(ctx, contrib)
	if err != nil {
		return out, err
	}
	return GenerateOutcome{
		Gross:       gross,
		EmployeeAmt: employeeAmt,
		EmployerAmt: employerAmt,
		Subsidy:     subsidy,
		Result:      result,
	}, nil
}
