package strategies

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/shopspring/decimal"
)

type PortfolioAnalysis struct {
	NuetralCoin     string
	InitialBalances *PortfolioBalance
	CurrentBalances *PortfolioBalance
}

func (is PortfolioAnalysis) String() string {
	pa_string := fmt.Sprintln("Type: " + reflect.TypeOf(is).String())
	pa_string += fmt.Sprintln("NuetralCoin: " + is.NuetralCoin)

	pa_string += fmt.Sprintln("Initial Balance:")
	pa_string += is.InitialBalances.String()
	pa_string += fmt.Sprintln("Current Balance:")
	pa_string += is.CurrentBalances.String()

	curr_diff, _ := is.GetCurrDiffInNuetralCoin(is.InitialBalances, is.CurrentBalances)
	pa_string += "Final difference in nuetral coin: " + curr_diff.String()

	return pa_string
}

func NewPortfolioAnalysis(nuetral_coin string, initial_balances *PortfolioBalance) (*PortfolioAnalysis, error) {

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
		InitialBalances: initial_balances,
		CurrentBalances: initial_balances,
	}, nil
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
