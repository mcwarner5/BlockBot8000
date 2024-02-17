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

package examples

import (
	"fmt"
	"time"

	"github.com/saniales/golang-crypto-trading-bot/environment"
	"github.com/saniales/golang-crypto-trading-bot/exchanges"
	"github.com/saniales/golang-crypto-trading-bot/strategies"

	"github.com/sirupsen/logrus"
)

// Watch5Sec prints out the info of the market every 5 seconds.
var Watch1Min = strategies.IntervalStrategy{
	Model: strategies.StrategyModel{
		Name: "Watch1Min",
		Setup: func(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) error {
			fmt.Println("Watch1Min starting")
			return nil
		},
		OnUpdate: func(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) error {

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

			for i, market := range markets {
				logrus.Info(market)
				logrus.Info(markets_info[i])
			}

			logrus.Info(wrappers)
			return nil
		},
		OnError: func(err error) {
			fmt.Println(err)
		},
		TearDown: func(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) error {
			fmt.Println("Watch1Min exited")
			return nil
		},
	},
	Interval: time.Second * 60,
}
