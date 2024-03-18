package strategies

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

type PortfolioAnalysis struct {
	NuetralCoin      string
	TotalBuyCount    decimal.Decimal
	TotalBuyPercent  decimal.Decimal
	TotalSellCount   decimal.Decimal
	TotalSellPercent decimal.Decimal
	TotalTrades      decimal.Decimal
	TotalFeesPercent decimal.Decimal
	TotalFessAmount  decimal.Decimal
	InitialBalances  *PortfolioBalance
	CurrentBalances  *PortfolioBalance
}

func (is PortfolioAnalysis) String() string {
	hundo := decimal.NewFromInt(100)
	pa_string := fmt.Sprintln("Type: " + reflect.TypeOf(is).String())
	pa_string += fmt.Sprintln("NuetralCoin: " + is.NuetralCoin)
	pa_string += fmt.Sprintln("TotalBuyCount: " + is.TotalBuyCount.String())
	pa_string += fmt.Sprintln("TotalBuyPercent: " + is.TotalBuyPercent.Mul(hundo).String() + "%")
	pa_string += fmt.Sprintln("TotalSellCount: " + is.TotalSellCount.String())
	pa_string += fmt.Sprintln("TotalSellPercent: " + is.TotalSellPercent.Mul(hundo).String() + "%")
	pa_string += fmt.Sprintln("TotalTrades: " + is.TotalTrades.String())
	pa_string += fmt.Sprintln("TotalFeesPercent: " + is.TotalFeesPercent.Mul(hundo).String() + "%")
	pa_string += fmt.Sprintln("TotalFessAmount: " + is.TotalFessAmount.String())
	pa_string += fmt.Sprintln("Initial Balance:")
	pa_string += is.InitialBalances.String()
	pa_string += fmt.Sprintln("Current Balance:")
	pa_string += is.CurrentBalances.String()

	curr_diff, _ := is.GetCurrDiffInNuetralCoin()
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
		NuetralCoin:      nuetral_coin,
		TotalBuyCount:    decimal.Zero,
		TotalBuyPercent:  decimal.Zero,
		TotalSellCount:   decimal.Zero,
		TotalSellPercent: decimal.Zero,
		TotalTrades:      decimal.Zero,
		TotalFeesPercent: decimal.Zero,
		TotalFessAmount:  decimal.Zero,
		InitialBalances:  initial_balances,
		CurrentBalances:  initial_balances,
	}, nil
}

func (is *PortfolioAnalysis) GetCurrDiffInNuetralCoin() (decimal.Decimal, error) {
	logrus.Info(is.InitialBalances.String())
	logrus.Info(is.CurrentBalances.String())

	init_val, err := is.InitialBalances.GetTotalValueInCoin(is.NuetralCoin)
	if err != nil {
		return decimal.Decimal{}, err
	}

	curr_val, err := is.CurrentBalances.GetTotalValueInCoin(is.NuetralCoin)
	if err != nil {
		return decimal.Decimal{}, err
	}

	return curr_val.Sub(init_val), nil
}
