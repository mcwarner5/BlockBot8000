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
	"time"

	"github.com/gofrs/uuid"
	"github.com/julien040/go-ternary"
	"github.com/mcwarner5/BlockBot8000/environment"
	"github.com/mcwarner5/BlockBot8000/exchanges"
	strat "github.com/mcwarner5/BlockBot8000/strategies"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

// IntervalStrategy is an interval based strategy.
type RebalancerStrategy struct {
	IntervalStrategy
	AllowanceThreshold    decimal.Decimal
	MarketCapMultiplier   decimal.Decimal
	MinimumTradeSize      decimal.Decimal
	StaticCoin            string
	NuetralCoin           string
	PortfolioDistribution map[string]decimal.Decimal
	Portfolio             *strat.PortfolioAnalysis
}

func NewRebalancerStrategy(raw_strat environment.StrategyConfig) strat.Strategy {
	//TODO validation

	var old_map = raw_strat.Spec["portfolio_ratio_percent"].(map[string]interface{})
	new_map := make(map[string]decimal.Decimal)

	total := decimal.Zero
	one := decimal.NewFromInt(1)

	for k, v := range old_map {
		new_val := decimal.NewFromFloat(v.(float64))
		total = total.Add(new_val)
		new_map[k] = new_val
	}

	if total.Sub(one).Abs().GreaterThan(decimal.NewFromFloat(0.01)) {
		panic("Error: Rebalancer Portfolio does not add up to 100%")
	}

	return &RebalancerStrategy{
		IntervalStrategy:      *NewIntervalStrategy(raw_strat),
		AllowanceThreshold:    decimal.NewFromFloat(raw_strat.Spec["allowance_threshold"].(float64)),
		MarketCapMultiplier:   decimal.NewFromFloat(raw_strat.Spec["market_cap_multiplier"].(float64)),
		MinimumTradeSize:      decimal.NewFromFloat(raw_strat.Spec["min_trade_size"].(float64)),
		StaticCoin:            raw_strat.Spec["static_coin"].(string),
		NuetralCoin:           raw_strat.Spec["nuetral_coin"].(string),
		PortfolioDistribution: new_map,
		Portfolio:             nil,
	}
}

// String returns a string representation of the object.
func (is RebalancerStrategy) String() string {
	hundo := decimal.NewFromInt(100)
	rbs_str := fmt.Sprintln("Type: " + reflect.TypeOf(is).String() + " Name:" + is.GetName())
	rbs_str += fmt.Sprintf("Interval: %d min\n", is.Interval)
	rbs_str += fmt.Sprintln("AllowanceThreshold: " + is.AllowanceThreshold.Mul(hundo).String() + "%")
	rbs_str += fmt.Sprintln("MinimumTradeSize: " + is.MinimumTradeSize.Mul(hundo).String() + "%")
	rbs_str += fmt.Sprintln("MarketCapMultiplier: " + is.MarketCapMultiplier.String())
	rbs_str += fmt.Sprintln("StaticCoin: " + is.StaticCoin)
	rbs_str += fmt.Sprintln("NuetralCoin: " + is.NuetralCoin)
	rbs_str += fmt.Sprintln("PortfolioDistribution:")
	for coin, value := range is.PortfolioDistribution {
		rbs_str += "\t" + fmt.Sprintf(`%s, %s%%`, coin, value.Mul(hundo).String()) + "\n"
	}
	rbs_str += is.Portfolio.String()
	return rbs_str
}

func (is RebalancerStrategy) Setup(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) (strat.Strategy, error) {
	fmt.Println("RebalancerStrategy Setup")
	coin_balance_info := make(map[string]*strat.CoinBalance)
	for coin := range is.PortfolioDistribution {
		coin_found := false
		for _, market := range markets {
			if coin == market.BaseCurrency {
				coin_found = true
				balance, err := wrappers[0].GetBalance(market.BaseCurrency)
				if err != nil {
					panic("rebalancer portfolio coin " + coin + " could not pull balance ")
				}

				data, err := wrappers[0].GetMarketSummary(market)
				if err != nil {
					panic("rebalancer portfolio coin " + coin + " could not pull market data with market " + market.Name)
				}

				coin_balance_info[coin], err = strat.NewCoinBalance(coin, *balance, data, market)
				if err != nil {
					panic("rebalancer portfolio coin " + coin + " could not create a coin balance ")
				}

				if wrappers[0].Name() == "simulator" && balance.GreaterThan(decimal.Zero) {
					orderFakeID, err := uuid.NewV4()
					if err != nil {
						return is, err
					}

					new_trade := environment.Trade{
						Price:        data.Last,
						AskQuantity:  *balance,
						FillQuantity: *balance,
						Fees:         decimal.Zero,
						Market:       market.Name,
						Side:         environment.Buy,
						Status:       environment.Complete,
						Type:         environment.MarketPrice,
						TradeNumber:  orderFakeID.String(),
						Timestamp:    time.Now(),
					}
					err = wrappers[0].(*exchanges.ExchangeWrapperSimulator).AddTrade(market, new_trade)
					if err != nil {
						return is, err
					}
				}
			}
		}
		if !coin_found {
			panic("rebalancer portfolio coin " + coin + " was not found in market list")
		}
	}
	var err error
	initial_balances, err := strat.NewPortfolioBalance(is.StaticCoin, coin_balance_info)
	if err != nil {
		return is, err
	}

	new_portfolio, err := strat.NewPortfolioAnalysis(is.NuetralCoin, initial_balances)

	if err != nil {
		return is, err
	}
	is.Portfolio = new_portfolio

	logrus.Info(is.Portfolio.InitialBalances.String())
	return is, nil
}

func (is RebalancerStrategy) OnError(err error) {
	err_str := fmt.Sprintln("RebalancerStrategy OnError")
	err_str += fmt.Sprintln(err)
	logrus.Error(err_str)
}

func (is RebalancerStrategy) TearDown(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) (strat.Strategy, error) {
	logrus.Info(fmt.Sprintln("RebalancerStrategy TearDown"))
	tradeBook, err := wrappers[0].GetAllTrades(markets)
	if err != nil {
		return is, err
	}

	logrus.Info(is.String())
	logrus.Info(tradeBook.String())

	return is, nil
}

func (is RebalancerStrategy) OnUpdate(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) (strat.Strategy, error) {
	//fmt.Println("OnUpdate " + is.String())

	is, err := is.UpdateCurrentBalances(wrappers, markets)

	if err != nil {
		return is, err
	}

	is, err = is.RebalanceSells(wrappers, markets)

	if err != nil {
		return is, err
	}

	is, err = is.RebalanceBuys(wrappers, markets)

	if err != nil {
		return is, err
	}

	//now call the wait function
	_, err = is.IntervalStrategy.OnUpdate(wrappers, markets)
	return is, err
}

func (is RebalancerStrategy) RebalanceBuys(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) (RebalancerStrategy, error) {
	var total_buy_back_percent decimal.Decimal
	var buy_logs string
	old_balance_str := is.Portfolio.CurrentBalances.String()

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
		buy_logs_headers := fmt.Sprintln("$$$ Transaction Logs, Total: "+total_buy_back_percent_str+" $$$") +
			fmt.Sprintln("type | tn_pf_%\t| coin\t|")
		buy_logs = buy_logs_headers + buy_logs

		logrus.Info(old_balance_str)
		logrus.Info(buy_logs)
		logrus.Info(is.Portfolio.CurrentBalances.String())
		logrus.Info(fmt.Sprintln("------------------------------------------------------------------------------"))
	}
	return is, nil
}

func (is RebalancerStrategy) RebalanceSells(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) (RebalancerStrategy, error) {
	var total_sell_off_percent decimal.Decimal
	var sell_logs string
	old_balance_str := is.Portfolio.CurrentBalances.String()
	//for portfolio_coin, expected_percent := range is.PortfolioDistribution {
	for _, coin_details := range is.GetSellList(is.GetAvailiblePercentToSell(wrappers, markets)) {
		portfolio_coin := coin_details.Key
		sell_percent := coin_details.Value
		//sell orders
		err := is.SellPercent(wrappers, markets, portfolio_coin, sell_percent)
		total_sell_off_percent = total_sell_off_percent.Add(sell_percent)

		if err != nil {
			return is, err
		}

		percent_str := sell_percent.Mul(decimal.NewFromInt(100)).Round(5).String()
		sell_logs = sell_logs + fmt.Sprintln("sell | "+percent_str+"\t| "+portfolio_coin)
	}

	is, err := is.UpdateCurrentBalances(wrappers, markets)

	if err != nil {
		return is, err
	}

	if total_sell_off_percent.GreaterThan(decimal.Zero) {
		total_sell_off_percent_str := total_sell_off_percent.Mul(decimal.NewFromInt(100)).Round(4).String()
		sell_logs_headers := fmt.Sprintln("$$$ Transaction Logs, Total: "+total_sell_off_percent_str+" $$$") +
			fmt.Sprintln("type | tn_pf_%\t| coin\t|")
		sell_logs = sell_logs_headers + sell_logs
		logrus.Info(fmt.Sprintln("$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$$"))
		logrus.Info(old_balance_str)
		logrus.Info(sell_logs)
		logrus.Info(is.Portfolio.CurrentBalances.String())
	}
	return is, nil
}

func (is RebalancerStrategy) GetAvailiblePercentToSell(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) map[string]decimal.Decimal {

	avail_list := make(map[string]decimal.Decimal)
	for coin := range is.PortfolioDistribution {
		for _, market := range markets {
			if coin == market.BaseCurrency && coin != is.StaticCoin {
				avail_list[coin] = decimal.Zero

				marketTades, err := wrappers[0].GetAllMarketTrades(market)
				if err != nil {
					return avail_list
				}

				data, err := wrappers[0].GetMarketSummary(market)
				if err != nil {
					return avail_list
				}

				avail_amount, err := wrappers[0].GetBalance(market.BaseCurrency)
				if err != nil {
					return avail_list
				}
				total_worth := decimal.Zero
				if avail_amount.GreaterThan(decimal.Zero) {
					total_worth = avail_amount.Mul(data.Last)
				}

				for _, trade := range marketTades.Trades {
					if trade.Status != environment.Complete {
						continue
					}

					if trade.Side == environment.Buy {
						total_worth = total_worth.Sub(trade.Price.Mul(trade.FillQuantity))
						continue
					}
					if trade.Side == environment.Sell {
						total_worth = total_worth.Add(trade.Price.Mul(trade.FillQuantity))
						continue
					}

				}

				if total_worth.LessThan(is.MinimumTradeSize) {
					continue
				}

				curr_total := is.Portfolio.CurrentBalances.GetTotal()
				total_pos_percent := total_worth.DivRound(curr_total, 4)

				if total_pos_percent.LessThan(is.MinimumTradeSize) {
					continue
				}
				avail_list[coin] = total_pos_percent
			}
		}
		continue
	}
	return avail_list
}

func (is RebalancerStrategy) GetSellList(availible_to_sell map[string]decimal.Decimal) []strat.CoinPercentPair {
	full_portfolio_list := strat.MapToSlice(availible_to_sell)
	initial_sell_list := make([]strat.CoinPercentPair, 0)
	potential_sell_list := make([]strat.CoinPercentPair, 0)
	final_return_list := make([]strat.CoinPercentPair, 0)

	var total_above_expected, total_below_expected, avail_static_coin_percent decimal.Decimal
	var excess, debt decimal.Decimal

	for _, curr_coin := range full_portfolio_list {
		curr_percent := is.Portfolio.CurrentBalances.GetCoinCurrentPortfolioPercent(curr_coin.Key)
		exp_percent := is.PortfolioDistribution[curr_coin.Key]

		curr_difference := curr_percent.Sub(exp_percent)
		upper_bounds := exp_percent.Add(exp_percent.Mul(is.AllowanceThreshold))
		lower_bounds := exp_percent.Sub(exp_percent.Mul(is.AllowanceThreshold))

		if curr_difference.Abs().LessThan(is.MinimumTradeSize) {
			continue
		}

		if curr_coin.Key == is.StaticCoin {
			if curr_difference.GreaterThan(decimal.Zero) {
				avail_static_coin_percent = curr_difference
				continue
			}
		}

		if curr_percent.GreaterThan(upper_bounds) {
			excess = excess.Add(curr_difference)

			if curr_coin.Value.LessThan(is.MinimumTradeSize) {
				continue
			}
			aval_percet := ternary.If(curr_coin.Value.GreaterThanOrEqual(curr_difference), curr_difference, curr_coin.Value)
			if aval_percet.LessThan(is.MinimumTradeSize) {
				continue
			}
			pair, _ := strat.NewCoinPercetPair(curr_coin.Key, aval_percet)
			initial_sell_list = append(initial_sell_list, pair)
			continue
		} else if curr_percent.LessThan(lower_bounds) {
			debt = debt.Add(curr_difference.Abs())
			continue
		}

		if curr_difference.GreaterThan(is.MinimumTradeSize) {
			aval_percet := ternary.If(curr_coin.Value.GreaterThanOrEqual(curr_difference), curr_difference, curr_coin.Value)
			if aval_percet.LessThan(is.MinimumTradeSize) {
				continue
			}
			pair, _ := strat.NewCoinPercetPair(curr_coin.Key, aval_percet)
			potential_sell_list = append(potential_sell_list, pair)
			total_above_expected = total_above_expected.Add(curr_difference)
		} else if curr_difference.LessThan(decimal.Zero) {
			total_below_expected = total_below_expected.Add(curr_difference.Abs())
		}
	}

	if debt.IsZero() {
		return initial_sell_list
	}

	logrus.Info(fmt.Sprintf("total debt found: %s", debt.String()))
	logrus.Info(fmt.Sprintf("total excess found: %s", excess.String()))
	logrus.Info(fmt.Sprintf("total static avail found: %s", avail_static_coin_percent.String()))

	if excess.Add(avail_static_coin_percent).GreaterThan(debt) {
		logrus.Info("debt covered with excess and avail static coin")
		return initial_sell_list
	}

	remainder_debt := debt.Sub(excess.Add(avail_static_coin_percent))

	logrus.Info(fmt.Sprintf("total remainder debt found: %s", remainder_debt.String()))

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
		if remainder_debt.GreaterThan(is.MinimumTradeSize) {
			if curr_coin.Value.GreaterThan(remainder_debt) {
				logrus.Info("remaining debt covered with avail coin " + curr_coin.Key)
				pair, _ := strat.NewCoinPercetPair(curr_coin.Key, remainder_debt)
				return append(final_return_list, pair)
			}
			logrus.Info("partial debt covered with avail coin " + curr_coin.Key)
			pair, _ := strat.NewCoinPercetPair(curr_coin.Key, curr_coin.Value)
			final_return_list = append(final_return_list, pair)
			remainder_debt = remainder_debt.Sub(curr_coin.Value)
		} else {
			break
		}
	}

	return final_return_list
}

func (is RebalancerStrategy) GetBuyList() []strat.CoinPercentPair {

	full_portfolio_list := strat.MapToSlice(is.PortfolioDistribution)
	initial_buy_list := make([]strat.CoinPercentPair, 0)
	potential_buy_list := make([]strat.CoinPercentPair, 0)
	final_return_list := make([]strat.CoinPercentPair, 0)

	var total_below_expected, avail_static_coin_percent decimal.Decimal
	var debt decimal.Decimal

	for _, curr_coin := range full_portfolio_list {
		curr_percent := is.Portfolio.CurrentBalances.GetCoinCurrentPortfolioPercent(curr_coin.Key)
		exp_percent := is.PortfolioDistribution[curr_coin.Key]

		curr_difference := exp_percent.Sub(curr_percent).Round(6)
		lower_bounds := exp_percent.Sub(exp_percent.Mul(is.AllowanceThreshold))

		if curr_difference.Abs().LessThan(is.MinimumTradeSize) {
			continue
		}

		if curr_coin.Key == is.StaticCoin {
			if curr_difference.GreaterThan(decimal.Zero) {
				//not enough static coin for any buys, lets return empty buy list
				return final_return_list
			}
			avail_static_coin_percent = curr_difference.Abs()
			continue
		}

		if curr_percent.LessThan(lower_bounds) {
			debt = debt.Add(curr_difference.Abs())
			pair, _ := strat.NewCoinPercetPair(curr_coin.Key, curr_difference.Abs())
			initial_buy_list = append(initial_buy_list, pair)
			continue
		}
		if curr_difference.GreaterThan(is.MinimumTradeSize) {
			pair, _ := strat.NewCoinPercetPair(curr_coin.Key, curr_difference)
			potential_buy_list = append(potential_buy_list, pair)
			total_below_expected = total_below_expected.Add(curr_difference)
		}

	}

	if avail_static_coin_percent.GreaterThan(debt.Add(total_below_expected)) {
		return append(initial_buy_list, potential_buy_list...)
	}

	avail_static_coin_after_debt := avail_static_coin_percent.Sub(debt)

	if avail_static_coin_after_debt.LessThan(decimal.Zero) {
		multiplier := avail_static_coin_percent.DivRound(debt, 5)
		for _, curr_coin := range initial_buy_list {
			final_amounts := curr_coin.Value.Mul(multiplier)
			if final_amounts.LessThan(is.MinimumTradeSize) {
				continue
			}
			pair, _ := strat.NewCoinPercetPair(curr_coin.Key, final_amounts)
			final_return_list = append(final_return_list, pair)
		}
		return final_return_list
	}

	if avail_static_coin_after_debt.LessThan(is.MinimumTradeSize) {
		return initial_buy_list
	}

	final_return_list = append(final_return_list, initial_buy_list...)

	multiplier := avail_static_coin_after_debt.DivRound(total_below_expected, 5)
	for _, curr_coin := range potential_buy_list {
		final_amounts := curr_coin.Value.Mul(multiplier)
		if final_amounts.LessThan(is.MinimumTradeSize) {
			continue
		}
		pair, _ := strat.NewCoinPercetPair(curr_coin.Key, final_amounts)
		final_return_list = append(final_return_list, pair)
	}

	return final_return_list
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
		coin_balnce, err := strat.NewCoinBalance(market.BaseCurrency, *balance, data, market)
		if err != nil {
			return is, err
		}

		is.Portfolio.CurrentBalances.Balances[market.BaseCurrency] = coin_balnce
		break
	}
	return is, nil
}

func (is RebalancerStrategy) UpdateCurrentBalances(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) (RebalancerStrategy, error) {

	coin_balance_info := make(map[string]*strat.CoinBalance)

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
				coin_balnce, err := strat.NewCoinBalance(market.BaseCurrency, *balance, data, market)
				if err != nil {
					return is, err
				}

				coin_balance_info[coin] = coin_balnce
				continue
			}
		}
		_, ok := coin_balance_info[coin]
		if !ok {
			return is, errors.New("market not found for coin " + coin)
		}
	}

	var err error
	is.Portfolio.CurrentBalances, err = strat.NewPortfolioBalance(is.StaticCoin, coin_balance_info)
	if err != nil {
		return is, err
	}

	return is, nil
}

func (is RebalancerStrategy) SellBackToExpected(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market, coin string) (decimal.Decimal, error) {
	curr_percent := is.Portfolio.CurrentBalances.GetCoinCurrentPortfolioPercent(coin)

	if curr_percent.LessThan(is.PortfolioDistribution[coin]) {
		return decimal.Decimal{}, errors.New("cannot sell coin that is already below its threshold")
	}

	sell_percet := curr_percent.Sub(is.PortfolioDistribution[coin])

	return sell_percet, is.SellPercent(wrappers, markets, coin, sell_percet)
}

func (is RebalancerStrategy) SellPercent(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market, coin string, sell_percent decimal.Decimal) error {
	if coin != is.StaticCoin {
		curr_price := is.Portfolio.CurrentBalances.Balances[coin].MarketData.Last
		curr_static_price := is.Portfolio.CurrentBalances.Balances[is.StaticCoin].MarketData.Last
		total := is.Portfolio.CurrentBalances.GetTotal()
		coin_price := curr_price.Mul(curr_static_price)
		sell_amount := total.Mul(sell_percent).DivRound(coin_price, 8)

		f_sell_amount, _ := sell_amount.Float64()

		//_, err := wrappers[0].SellMarket(is.PortfolioBalances.Balances[coin].Market, f_sell_amount)
		_, err := wrappers[0].SellLimit(is.Portfolio.CurrentBalances.Balances[coin].Market, f_sell_amount, curr_price.InexactFloat64())

		if err != nil {
			return err
		}

		return nil
	}

	return errors.New("cannont sell Static Coin")
}

func (is RebalancerStrategy) BuyBackToExpected(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market, coin string) (decimal.Decimal, error) {

	curr_percent := is.Portfolio.CurrentBalances.GetCoinCurrentPortfolioPercent(coin)

	if curr_percent.GreaterThan(is.PortfolioDistribution[coin]) {
		return decimal.Decimal{}, errors.New("cannot buy coin that is already above its threshold")
	}

	buy_percet := is.PortfolioDistribution[coin].Sub(curr_percent)

	return buy_percet, is.BuyPercent(wrappers, markets, coin, buy_percet)

}

func (is RebalancerStrategy) BuyPercent(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market, coin string, buy_percent decimal.Decimal) error {
	if coin != is.StaticCoin {
		curr_price := is.Portfolio.CurrentBalances.Balances[coin].MarketData.Last
		curr_static_price := is.Portfolio.CurrentBalances.Balances[is.StaticCoin].MarketData.Last
		coin_price := curr_price.Mul(curr_static_price)
		total := is.Portfolio.CurrentBalances.GetTotal()
		total_cost := total.Mul(buy_percent)
		total_static_coin := (is.Portfolio.CurrentBalances.Balances[is.StaticCoin].Balance).Mul(is.Portfolio.CurrentBalances.Balances[is.StaticCoin].MarketData.Last)
		buy_amount := total_cost.DivRound(coin_price, 8)

		if total_cost.GreaterThan(total_static_coin) {

			return errors.New("not enough static coin to make purchase")
			//buy_amount = total_static_coin.Mul(decimal.NewFromFloat(0.5))
		}

		f_buy_amount, _ := buy_amount.Float64()
		//_, err := wrappers[0].BuyMarket(is.PortfolioBalances.Balances[coin].Market, f_buy_amount)
		_, err := wrappers[0].BuyLimit(is.Portfolio.CurrentBalances.Balances[coin].Market, f_buy_amount, curr_price.InexactFloat64())

		if err != nil {
			return err
		}

		return nil
	}

	return errors.New("cannont buy static coin")
}
