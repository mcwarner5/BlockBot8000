// Copyright © 2017 Alessandro Sanino <saninoale@gmail.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package intervalstrategies

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/julien040/go-ternary"
	"github.com/mcwarner5/BlockBot8000/environment"
	"github.com/mcwarner5/BlockBot8000/exchanges"
	"github.com/mcwarner5/BlockBot8000/strategies"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

type CoinPercentPair struct {
	Key   string
	Value decimal.Decimal
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

type PortfolioBalance struct {
	Balances map[string]*CoinBalance
}

func (is PortfolioBalance) String() string {
	total_str := is.GetTotal().Round(4).String()
	pb_string := fmt.Sprintln("***	Portfolio Balance, Total: " + total_str + " ***")
	pb_string = pb_string + fmt.Sprintln(" COIN\t\t| PF%\t\t| QTY\t\t| PRICE\t\t| USD\t\t|")
	for coin, balance := range is.Balances {
		c_balance := balance.Balance.Round(2).String()
		c_price := balance.MarketData.Last.Round(4).String()
		c_value := balance.Balance.Mul(balance.MarketData.Last).Round(2).String()
		c_percent := is.GetPortfolioPercent(coin).Mul(decimal.NewFromFloat32(100.0)).String()

		log := " " + coin + ternary.If(len(coin) > 5, "\t", "\t\t") + "| "
		log = log + c_percent + ternary.If(len(c_percent) > 5, "\t", "\t\t") + "| "
		log = log + c_balance + ternary.If(len(c_balance) > 5, "\t", "\t\t") + "| "
		log = log + c_price + ternary.If(len(c_price) > 5, "\t", "\t\t") + "| "
		log = log + c_value + ternary.If(len(c_value) > 5, "\t", "\t\t") + "| "
		pb_string = pb_string + fmt.Sprintln(log)
	}

	return pb_string
}

func (is PortfolioBalance) GetTotal() decimal.Decimal {
	var curr_total decimal.Decimal

	for _, coin := range is.Balances {
		curr_total = curr_total.Add(coin.Balance.Mul(coin.MarketData.Ask))
	}

	return curr_total
}

func (is PortfolioBalance) GetPortfolioPercent(coin string) decimal.Decimal {
	coin_balance, ok := is.Balances[coin]
	if !ok {
		panic("coin not found")
	}

	return (coin_balance.Balance.Mul(coin_balance.MarketData.Last)).DivRound(is.GetTotal(), 4)
}

// IntervalStrategy is an interval based strategy.
type RebalancerStrategy struct {
	IntervalStrategy
	AllowanceThreshold    decimal.Decimal
	MarketCapMultiplier   decimal.Decimal
	StaticCoin            string
	PortfolioDistribution map[string]decimal.Decimal
	PortfolioBalances     *PortfolioBalance
	InitialBalances       *PortfolioBalance
}

func NewRebalancerStrategy(raw_strat environment.StrategyConfig) strategies.Strategy {
	//TODO validation

	var old_map = raw_strat.Spec["portfolio_ratio_percent"].(map[string]interface{})
	new_map := make(map[string]decimal.Decimal)

	for k, v := range old_map {
		new_map[k] = decimal.NewFromFloat(v.(float64))
	}

	return &RebalancerStrategy{
		IntervalStrategy:      *NewIntervalStrategy(raw_strat),
		AllowanceThreshold:    decimal.NewFromFloat(raw_strat.Spec["allowance_threshold"].(float64)),
		MarketCapMultiplier:   decimal.NewFromFloat(raw_strat.Spec["market_cap_multiplier"].(float64)),
		StaticCoin:            raw_strat.Spec["static_coin"].(string),
		PortfolioDistribution: new_map,
	}
}

// String returns a string representation of the object.
func (is RebalancerStrategy) String() string {
	return "Type: " + reflect.TypeOf(is).String() + " Name:" + is.GetName()
}

func (is RebalancerStrategy) Setup(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) (strategies.Strategy, error) {
	fmt.Println("RebalancerStrategy Setup")

	coin_balance_info := make(map[string]*CoinBalance)
	for _, market := range markets {
		coin := market.Name
		balance, err := wrappers[0].GetBalance(market.BaseCurrency)
		if err != nil {
			return is, err
		}

		data, err := wrappers[0].GetMarketSummary(market)
		if err != nil {
			return is, err
		}

		coin_balance_info[coin] = &CoinBalance{coin, *balance, data, market}
	}
	is.InitialBalances = &PortfolioBalance{Balances: coin_balance_info}

	//logrus.Info(is.InitialBalances.String())
	return is, nil
}

func (is RebalancerStrategy) OnError(err error) {
	fmt.Println("RebalancerStrategy OnError")
	fmt.Println(err)
}

func (is RebalancerStrategy) TearDown(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) error {
	fmt.Println("RebalancerStrategy TearDown")
	return nil
}

func (is RebalancerStrategy) OnUpdate(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) error {
	fmt.Println("OnUpdate " + is.String())

	is, err := is.UpdateCurrentBalances(wrappers, markets)

	if err != nil {
		return err
	}

	is, err = is.RebalanceSells(wrappers, markets)

	if err != nil {
		return err
	}

	is, err = is.RebalanceBuys(wrappers, markets)

	if err != nil {
		return err
	}

	//now call the wait function
	is.IntervalStrategy.OnUpdate(wrappers, markets)
	return nil
}

func (is RebalancerStrategy) RebalanceBuys(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) (RebalancerStrategy, error) {
	var total_buy_back_percent decimal.Decimal
	var buy_logs string

	for _, coin_details := range is.GetBuyList() {
		buy_back_coin := coin_details.Key
		buy_percent := coin_details.Value

		err := is.BuyPercent(wrappers, markets, buy_back_coin, buy_percent)
		total_buy_back_percent = total_buy_back_percent.Add(buy_percent)

		if err != nil {
			return is, err
		}

		percent_str := buy_percent.Mul(decimal.NewFromInt(100)).Round(3).String()
		buy_logs = buy_logs + fmt.Sprintln("buy  | "+percent_str+"\t| "+buy_back_coin)
	}

	is, err := is.UpdateCurrentBalances(wrappers, markets)

	if err != nil {
		return is, err
	}

	if total_buy_back_percent.GreaterThan(decimal.Zero) {
		total_buy_back_percent_str := total_buy_back_percent.Mul(decimal.NewFromInt(100)).Round(4).String()
		buy_logs = fmt.Sprintln("$$$ Transaction Logs, Total: "+total_buy_back_percent_str+" $$$") + fmt.Sprintln("type | tn_pf_%\t| coin\t|") + buy_logs

		logrus.Info(buy_logs)
		logrus.Info(is.PortfolioBalances.String())
	}
	return is, nil
}

func (is RebalancerStrategy) RebalanceSells(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) (RebalancerStrategy, error) {
	var total_sell_off_percent decimal.Decimal
	var sell_logs string

	//for portfolio_coin, expected_percent := range is.PortfolioDistribution {
	for _, coin_details := range is.GetSellList() {
		portfolio_coin := coin_details.Key
		sell_percent := coin_details.Value
		//sell orders
		err := is.SellPercent(wrappers, markets, portfolio_coin, sell_percent)
		total_sell_off_percent = total_sell_off_percent.Add(sell_percent)

		if err != nil {
			return is, err
		}

		percent_str := sell_percent.Mul(decimal.NewFromInt(100)).Round(3).String()
		sell_logs = sell_logs + fmt.Sprintln("sell | "+percent_str+ternary.If(len(percent_str) > 5, "\t", "\t\t")+"| "+portfolio_coin)
	}

	is, err := is.UpdateCurrentBalances(wrappers, markets)

	if err != nil {
		return is, err
	}

	if total_sell_off_percent.GreaterThan(decimal.Zero) {
		total_sell_off_percent_str := total_sell_off_percent.Mul(decimal.NewFromInt(100)).Round(4).String()
		sell_logs = fmt.Sprintln("$$$ Transaction Logs, Total: "+total_sell_off_percent_str+" $$$") + fmt.Sprintln("type | tn_pf_%\t| coin\t|") + sell_logs

		logrus.Info(sell_logs)
		logrus.Info(is.PortfolioBalances.String())
	}
	return is, nil
}

func (is RebalancerStrategy) GetSellList() []CoinPercentPair {
	full_portfolio_list := MapToSlice(is.PortfolioDistribution)
	initial_sell_list := make([]CoinPercentPair, 0)
	potential_sell_list := make([]CoinPercentPair, 0)
	final_return_list := make([]CoinPercentPair, 0)

	var total_above_expected, total_below_expected, avail_static_coin_percent decimal.Decimal
	var excess, debt decimal.Decimal

	for _, curr_coin := range full_portfolio_list {
		curr_percent := is.CurrPercent(curr_coin.Key)
		exp_percent := is.PortfolioDistribution[curr_coin.Key]

		curr_difference := curr_percent.Sub(exp_percent).Round(6)
		upper_bounds := exp_percent.Add(exp_percent.Mul(is.AllowanceThreshold))
		lower_bounds := exp_percent.Sub(exp_percent.Mul(is.AllowanceThreshold))

		if curr_coin.Key == is.StaticCoin {
			if curr_difference.GreaterThan(decimal.Zero) {
				total_above_expected = total_above_expected.Add(curr_difference)
				avail_static_coin_percent = curr_difference
				continue
			}
		}

		if curr_percent.GreaterThan(upper_bounds) {
			excess = excess.Add(curr_difference)
			initial_sell_list = append(initial_sell_list, CoinPercentPair{curr_coin.Key, curr_difference})
			continue
		} else if curr_percent.LessThan(lower_bounds) {
			debt = debt.Add(curr_difference)
			continue
		}

		if curr_difference.GreaterThan(decimal.NewFromFloat(1.0 / 100.0)) {
			potential_sell_list = append(potential_sell_list, CoinPercentPair{curr_coin.Key, curr_difference})
			total_above_expected = total_above_expected.Add(curr_difference)
		} else if curr_difference.LessThan(decimal.Zero) {
			total_below_expected = total_below_expected.Add(curr_difference.Abs())
		}
	}

	if excess.Add(avail_static_coin_percent).GreaterThan(debt) {
		return initial_sell_list
	}

	remainder_debt := debt.Sub(excess.Add(avail_static_coin_percent))

	sort.Slice(potential_sell_list, func(i, j int) bool {
		// 1. value is different - sort by value (in reverse order)
		if potential_sell_list[i].Value != potential_sell_list[j].Value {
			return potential_sell_list[i].Value.GreaterThan(potential_sell_list[j].Value)
		}
		// 2. only when value is the same - sort by key
		return potential_sell_list[i].Key < potential_sell_list[j].Key
	})

	final_return_list = append(final_return_list, initial_sell_list...)

	for _, curr_coin := range potential_sell_list {
		if curr_coin.Value.GreaterThan(remainder_debt) {
			final_return_list = append(final_return_list, CoinPercentPair{curr_coin.Key, remainder_debt})
		}
	}

	return final_return_list
}

func (is RebalancerStrategy) GetBuyList() []CoinPercentPair {

	full_portfolio_list := MapToSlice(is.PortfolioDistribution)
	initial_buy_list := make([]CoinPercentPair, 0)
	final_return_list := make([]CoinPercentPair, 0)

	var total_above_expected, total_below_expected, avail_static_coin_percent decimal.Decimal

	for _, curr_coin := range full_portfolio_list {
		curr_percent := is.CurrPercent(curr_coin.Key)
		exp_percent := is.PortfolioDistribution[curr_coin.Key]

		curr_difference := exp_percent.Sub(curr_percent).Round(6)

		if curr_coin.Key == is.StaticCoin {
			if curr_difference.GreaterThan(decimal.Zero) {
				//not enough static coin for any buys, lets return empty buy list
				return final_return_list
			}
			avail_static_coin_percent = curr_difference.Abs()
			continue
		}

		if curr_difference.GreaterThan(decimal.NewFromFloat(1.0 / 100.0)) {
			initial_buy_list = append(initial_buy_list, CoinPercentPair{curr_coin.Key, curr_difference})
			total_below_expected = total_below_expected.Add(curr_difference)
		}

		if curr_difference.LessThan(decimal.Zero) {
			total_above_expected = total_above_expected.Add(curr_difference.Abs())
		}

	}

	multiplier := decimal.NewFromFloat32(1.0)

	if avail_static_coin_percent.LessThan(total_below_expected) {
		multiplier = avail_static_coin_percent.Div(total_below_expected)
	}

	for _, curr_coin := range initial_buy_list {
		final_return_list = append(final_return_list, CoinPercentPair{curr_coin.Key, curr_coin.Value.Mul(multiplier)})
	}

	return final_return_list
}

func (is RebalancerStrategy) CurrPercent(coin string) decimal.Decimal {
	curr_balance := is.PortfolioBalances.Balances[coin].Balance
	curr_price := is.PortfolioBalances.Balances[coin].MarketData.Last
	return (curr_balance.Mul(curr_price)).Div(is.PortfolioBalances.GetTotal())
}

func (is RebalancerStrategy) CurrPercentFromBlanace(coin string, balance decimal.Decimal) decimal.Decimal {
	curr_price := is.PortfolioBalances.Balances[coin].MarketData.Last
	return (balance.Mul(curr_price)).Div(is.PortfolioBalances.GetTotal())
}

func (is RebalancerStrategy) UpdateStaticCoinBalance(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) (RebalancerStrategy, error) {

	for _, market := range markets {
		if market.BaseCurrency != is.StaticCoin {
			continue
		}
		balance, err := wrappers[0].GetBalance(market.BaseCurrency)
		if err != nil {
			return is, err
		}
		data, err := wrappers[0].GetMarketSummary(market)
		if err != nil {
			return is, err
		}

		is.PortfolioBalances.Balances[market.BaseCurrency] = &CoinBalance{market.BaseCurrency, *balance, data, market}
		break
	}
	return is, nil
}

func (is RebalancerStrategy) UpdateCurrentBalances(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) (RebalancerStrategy, error) {

	coin_balance_info := make(map[string]*CoinBalance)

	for coin := range is.PortfolioDistribution {
		for _, market := range markets {
			if coin == market.BaseCurrency {
				balance, err := wrappers[0].GetBalance(market.BaseCurrency)
				if err != nil {
					return is, err
				}
				data, err := wrappers[0].GetMarketSummary(market)
				if err != nil {
					return is, err
				}

				coin_balance_info[coin] = &CoinBalance{coin, *balance, data, market}
			}
		}
		_, ok := coin_balance_info[coin]
		if !ok {
			return is, errors.New("market not found for coin " + coin)
		}
	}

	is.PortfolioBalances = &PortfolioBalance{Balances: coin_balance_info}

	return is, nil
}

func (is RebalancerStrategy) SellBackToExpected(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market, coin string) (decimal.Decimal, error) {
	curr_percent := is.CurrPercent(coin)

	if curr_percent.LessThan(is.PortfolioDistribution[coin]) {
		return decimal.Decimal{}, errors.New("cannot sell coin that is already below its threshold")
	}

	sell_percet := curr_percent.Sub(is.PortfolioDistribution[coin])

	return sell_percet, is.SellPercent(wrappers, markets, coin, sell_percet)
}

func (is RebalancerStrategy) SellPercent(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market, coin string, sell_percent decimal.Decimal) error {
	if coin != is.StaticCoin {
		curr_price := is.PortfolioBalances.Balances[coin].MarketData.Bid
		total := is.PortfolioBalances.GetTotal()
		sell_amount := total.Mul(sell_percent).DivRound(curr_price, 2)

		f_sell_amount, _ := sell_amount.Float64()

		_, err := wrappers[0].SellMarket(is.PortfolioBalances.Balances[coin].Market, f_sell_amount)

		if err != nil {
			return err
		}

		return nil
	}

	return errors.New("cannont sell Static Coin")
}

func (is RebalancerStrategy) BuyBackToExpected(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market, coin string) (decimal.Decimal, error) {

	curr_percent := is.CurrPercent(coin)

	if curr_percent.GreaterThan(is.PortfolioDistribution[coin]) {
		return decimal.Decimal{}, errors.New("cannot buy coin that is already above its threshold")
	}

	buy_percet := is.PortfolioDistribution[coin].Sub(curr_percent)

	return buy_percet, is.BuyPercent(wrappers, markets, coin, buy_percet)

}

func (is RebalancerStrategy) BuyPercent(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market, coin string, buy_percent decimal.Decimal) error {
	if coin != is.StaticCoin {
		curr_price := is.PortfolioBalances.Balances[coin].MarketData.Ask
		total := is.PortfolioBalances.GetTotal()
		total_cost := total.Mul(buy_percent)
		total_static_coin := (is.PortfolioBalances.Balances[is.StaticCoin].Balance).Mul(is.PortfolioBalances.Balances[is.StaticCoin].MarketData.Ask)
		buy_amount := total_cost.DivRound(curr_price, 2)

		if total_cost.GreaterThan(total_static_coin) {

			return errors.New("not enough static coin to make purchase")
			//buy_amount = total_static_coin.Mul(decimal.NewFromFloat(0.5))
		}

		f_buy_amount, _ := buy_amount.Float64()

		_, err := wrappers[0].BuyMarket(is.PortfolioBalances.Balances[coin].Market, f_buy_amount)

		if err != nil {
			return err
		}

		return nil
	}

	return errors.New("cannont buy static coin")
}
