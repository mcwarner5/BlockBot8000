package strategies

import (
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

type PortfolioAnalysis struct {
	NuetralCoin     string
	StartDate       time.Time
	CurrentDate     time.Time
	InitialBalances *PortfolioBalance
	CurrentBalances *PortfolioBalance
}

func (is PortfolioAnalysis) SetCurrDate(currDate time.Time) (*PortfolioAnalysis, error) {
	if currDate.Before(is.StartDate) {
		return &is, errors.New("portfolio analysis current date set is before its start date")
	}
	is.CurrentDate = currDate
	return &is, nil
}

func (is PortfolioAnalysis) CalcCurrAPR() decimal.Decimal {
	curr_diff, _ := is.GetCurrDiffInNuetralCoin(is.InitialBalances, is.CurrentBalances)
	init_val, _ := is.InitialBalances.GetTotalValueInCoin(is.NuetralCoin)
	diff_days := decimal.NewFromInt(1)
	year := decimal.NewFromInt(365)
	hundo := decimal.NewFromInt(100)

	if is.CurrentDate.After(is.StartDate) {
		diff_days = decimal.NewFromFloat(is.CurrentDate.Sub(is.StartDate).Hours()).Div(decimal.NewFromInt(24))
	}

	apr := (curr_diff.Div(init_val)).Div(diff_days).Mul(year).Mul(hundo).Round(4)
	return apr
}

func (is PortfolioAnalysis) String() string {
	pa_string := fmt.Sprintln("Initial Balance:")
	pa_string += is.InitialBalances.String()
	pa_string += fmt.Sprintln("Current Balance:")
	pa_string += is.CurrentBalances.String()

	curr_diff, _ := is.GetCurrDiffInNuetralCoin(is.InitialBalances, is.CurrentBalances)
	init_val, _ := is.InitialBalances.GetTotalValueInCoin(is.NuetralCoin)
	curr_val, _ := is.CurrentBalances.GetTotalValueInCoin(is.NuetralCoin)

	pa_string += fmt.Sprintf("Initial Value in nuetral coin: %s %s", init_val.String(), is.NuetralCoin) + "\n"
	pa_string += fmt.Sprintf("Current Value in nuetral coin: %s %s", curr_val.String(), is.NuetralCoin) + "\n"

	pa_string += fmt.Sprintf("Final difference in nuetral coin: %s %s", curr_diff.String(), is.NuetralCoin) + "\n"
	pa_string += fmt.Sprintf("Final APR in nuetral coin: %s%%", is.CalcCurrAPR().String()) + "\n"
	return pa_string
}

func NewPortfolioAnalysis(nuetral_coin string, start_date time.Time, initial_balances *PortfolioBalance) (*PortfolioAnalysis, error) {

	if nuetral_coin == "" {
		return nil, errors.New("cannot create PortfolioAnalysis with empty nuetral coin")
	}

	if initial_balances == nil {
		return nil, errors.New("cannot create PortfolioAnalysis with nil initial balance object")
	}
	nuetral_coin_found := false
	for coin, coin_balance := range initial_balances.Balances {
		if coin == nuetral_coin {
			nuetral_coin_found = true
		}
		if coin_balance == nil {
			return nil, errors.New("cannot create PortfolioAnalysis with invalid initial balance object")
		}

	}
	if !nuetral_coin_found {
		return nil, errors.New("cannot create PortfolioAnalysis with nil initial balance object")
	}

	return &PortfolioAnalysis{
		NuetralCoin:     nuetral_coin,
		StartDate:       start_date,
		CurrentDate:     start_date,
		InitialBalances: initial_balances,
		CurrentBalances: initial_balances,
	}, nil
}

func (is *PortfolioAnalysis) GetCurrDiffInValue(start *PortfolioBalance, end *PortfolioBalance) (decimal.Decimal, error) {
	init_val := start.GetTotal()
	end_val := end.GetTotal()

	return end_val.Sub(init_val), nil
}

func (is *PortfolioAnalysis) GetCurrDiffInNuetralCoin(start *PortfolioBalance, end *PortfolioBalance) (decimal.Decimal, error) {
	init_val, err := start.GetTotalValueInCoin(is.NuetralCoin)
	if err != nil {
		return decimal.Decimal{}, err
	}

	end_val, err := end.GetTotalValueInCoin(is.NuetralCoin)
	if err != nil {
		return decimal.Decimal{}, err
	}

	return end_val.Sub(init_val), nil
}
