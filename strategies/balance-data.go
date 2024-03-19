package strategies

import (
	"errors"
	"fmt"
	"strings"

	"github.com/julien040/go-ternary"
	"github.com/mcwarner5/BlockBot8000/environment"
	"github.com/shopspring/decimal"
)

type CoinPercentPair struct {
	Key   string
	Value decimal.Decimal
}

func NewCoinPercetPair(coin string, percent decimal.Decimal) (CoinPercentPair, error) {
	if coin == "" {
		return CoinPercentPair{}, errors.New("cannot create CoinPercentPair with empty coin")
	}
	if percent.IsNegative() {
		return CoinPercentPair{}, errors.New("cannot create CoinPercentPair with negative value")
	}
	if percent.GreaterThan(decimal.NewFromInt(1)) {
		return CoinPercentPair{}, errors.New("cannot create CoinPercentPair with value greater thatn 100%")
	}

	return CoinPercentPair{
		Key:   coin,
		Value: percent,
	}, nil
}

func (is CoinPercentPair) String() string {
	ret := fmt.Sprintln("Key: ", is.Key) + fmt.Sprintln("Value: ", is.Value)
	return strings.TrimSpace(ret)
}

func MapToSlice(in map[string]decimal.Decimal) []CoinPercentPair {
	vec := make([]CoinPercentPair, len(in))
	i := 0
	for k, v := range in {
		vec[i].Key = k
		vec[i].Value = v
		i++
	}
	return vec
}

type CoinBalance struct {
	Coin       string
	Balance    decimal.Decimal
	MarketData *environment.MarketSummary
	Market     *environment.Market
}

func NewCoinBalance(coin string, balance decimal.Decimal, market_data *environment.MarketSummary, market *environment.Market) (*CoinBalance, error) {

	if market_data == nil {
		return nil, errors.New("Cannot create CoinBalance entry for coin:" + coin + " with nil market data")
	}

	if market == nil {
		return nil, errors.New("Cannot create CoinBalance entry for coin:" + coin + " with nil market")
	}
	ret := &CoinBalance{
		Coin:       coin,
		Balance:    balance,
		MarketData: market_data,
		Market:     market,
	}
	return ret, nil
}

type PortfolioBalance struct {
	StaticCoin string
	Balances   map[string]*CoinBalance
}

func NewPortfolioBalance(static_coin string, balances map[string]*CoinBalance) (*PortfolioBalance, error) {

	if static_coin == "" {
		return nil, errors.New("cannot create PortfolioBalance with empty static coin")
	}

	staticCoinFound := false
	for coin_key, coin_balance := range balances {
		if coin_balance == nil {
			return nil, errors.New("cannot create PortfolioBalance with nil coin_balance for coin: " + coin_key)
		}
		if coin_key == static_coin {
			staticCoinFound = true
		}
	}

	if !staticCoinFound {
		return nil, errors.New("cannot create PortfolioBalance without static coin presence")
	}

	return &PortfolioBalance{
		StaticCoin: static_coin,
		Balances:   balances}, nil

}

func (is PortfolioBalance) String() string {
	total_str := is.GetTotal().Round(4).String()
	pb_string := fmt.Sprintln("***	Portfolio Balance, Total: " + total_str + " ***")
	pb_string = pb_string + fmt.Sprintln(" COIN\t\t| PF%\t\t| QTY\t\t| PRICE\t\t| USD\t\t|")
	for coin, balance := range is.Balances {
		c_balance := balance.Balance.Round(2).String()
		c_price := balance.MarketData.Last.Round(4).String()
		c_value := balance.Balance.Mul(balance.MarketData.Last).Round(2).String()
		c_percent := is.GetCoinCurrentPortfolioPercent(coin).Mul(decimal.NewFromFloat32(100.0)).String()

		log := " " + coin + ternary.If(len(coin) > 5, "\t", "\t\t") + "| "
		log = log + c_percent + ternary.If(len(c_percent) > 5, "\t", "\t\t") + "| "
		log = log + c_balance + ternary.If(len(c_balance) > 5, "\t", "\t\t") + "| "
		log = log + c_price + ternary.If(len(c_price) > 5, "\t", "\t\t") + "| "
		log = log + c_value + ternary.If(len(c_value) > 5, "\t", "\t\t") + "| "
		pb_string = pb_string + fmt.Sprintln(log)
	}

	return pb_string
}

func (is PortfolioBalance) GetValue(coin string) decimal.Decimal {
	var curr_total decimal.Decimal
	static_price := is.Balances[is.StaticCoin].MarketData.Last
	coinBalance := is.Balances[coin]

	if coin != is.StaticCoin {
		curr_total = curr_total.Add(coinBalance.Balance.Mul(coinBalance.MarketData.Last).Mul(static_price))
	} else {
		curr_total = curr_total.Add(coinBalance.Balance.Mul(coinBalance.MarketData.Last))
	}

	if curr_total.IsZero() {
		panic("No total found in portfolio")
	}

	return curr_total
}

func (is PortfolioBalance) GetTotal() decimal.Decimal {
	var curr_total decimal.Decimal
	static_price := is.Balances[is.StaticCoin].MarketData.Last

	for _, coin := range is.Balances {
		if coin.Coin != is.StaticCoin {
			curr_total = curr_total.Add(coin.Balance.Mul(coin.MarketData.Last).Mul(static_price))
		} else {
			curr_total = curr_total.Add(coin.Balance.Mul(coin.MarketData.Last))
		}
	}

	if curr_total.IsZero() {
		panic("No total found in portfolio")
	}

	return curr_total
}

func (is PortfolioBalance) GetTotalValueInCoin(nuetral_coin string) (decimal.Decimal, error) {
	static_coin := is.StaticCoin
	curr_total := is.GetTotal()

	staticMarket := is.Balances[static_coin].MarketData
	if staticMarket == nil {
		return decimal.Decimal{}, errors.New("no static coin market data found in portfolio")
	}

	nuetralMarket := is.Balances[nuetral_coin].MarketData
	if nuetralMarket == nil {
		return decimal.Decimal{}, errors.New("no nuetral coin market data found in portfolio")
	}

	curr_static_price := staticMarket.Last
	curr_nuetral_price := nuetralMarket.Last

	if curr_nuetral_price.IsZero() {
		curr_nuetral_price = curr_static_price
	}
	if nuetral_coin != static_coin {
		curr_nuetral_price = curr_nuetral_price.Mul(curr_static_price)
	}
	return curr_total.DivRound(curr_nuetral_price, 5), nil
}

func (is PortfolioBalance) GetCoinCurrentPortfolioPercent(coin string) decimal.Decimal {
	coin_balance, ok := is.Balances[coin]
	if !ok {
		panic("coin not found")
	}
	if coin == is.StaticCoin {
		return (coin_balance.Balance.Mul(coin_balance.MarketData.Last)).DivRound(is.GetTotal(), 4)
	}

	static_price := is.Balances[is.StaticCoin].MarketData.Last

	return (coin_balance.Balance.Mul(coin_balance.MarketData.Last).Mul(static_price)).DivRound(is.GetTotal(), 4)
}
