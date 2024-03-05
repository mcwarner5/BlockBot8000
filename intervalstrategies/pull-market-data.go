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
	"fmt"

	"github.com/mcwarner5/BlockBot8000/environment"
	"github.com/mcwarner5/BlockBot8000/exchanges"
	"github.com/mcwarner5/BlockBot8000/strategies"
)

type PullMarketData struct {
	IntervalStrategy
	CandlesEnabled bool
}

func NewPullMarketData(raw_strat environment.StrategyConfig) strategies.Strategy {
	//TODO validation
	//return strategies.NewIntervalStrategy(raw_strat)

	return &PullMarketData{
		IntervalStrategy: *NewIntervalStrategy(raw_strat),
		CandlesEnabled:   false,
	}
}

func (is PullMarketData) Setup(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) (strategies.Strategy, error) {
	fmt.Println("PullMarketData starting")
	return is, nil
}

func (is PullMarketData) OnUpdate(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) error {

	markets_info := make([]environment.MarketSummary, 0, len(markets))
	//candles_info := make([]environment.CandleStickChart, 0, len(markets))

	for _, market := range markets {
		data, err := wrappers[0].GetMarketSummary(market)
		if err != nil {
			return err
		}
		markets_info = append(markets_info, *data)

		_, err2 := wrappers[0].GetCandles(market)
		if err2 != nil {
			return err
		}
		//candles_info = append(candles_info, *candles)
	}

	is.IntervalStrategy.OnUpdate(wrappers, markets)
	return nil
}

func (is PullMarketData) OnError(err error) {
	fmt.Println(err)
}

func (is PullMarketData) TearDown(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) error {
	fmt.Println("Watch1Min exited")
	return nil
}
