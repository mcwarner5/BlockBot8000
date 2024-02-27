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
	"reflect"

	"github.com/mcwarner5/BlockBot8000/environment"
	"github.com/mcwarner5/BlockBot8000/exchanges"
	"github.com/mcwarner5/BlockBot8000/strategies"
	"github.com/shopspring/decimal"
)

// IntervalStrategy is an interval based strategy.
type RebalancerStrategy struct {
	IntervalStrategy
	AllowanceThreshold  decimal.Decimal
	MarketCapMultiplier decimal.Decimal
	StaticCoin          string
	Portfolio           map[string]decimal.Decimal
}

// String returns a string representation of the object.
func (is RebalancerStrategy) String() string {
	return "Type: " + reflect.TypeOf(is).String() + " Name:" + is.GetName()
}

func (is RebalancerStrategy) Setup(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) error {
	fmt.Println("RebalancerStrategy Setup")
	return nil
}

func (is RebalancerStrategy) OnUpdate(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) error {
	fmt.Println("OnUpdate " + is.String())

	is.IntervalStrategy.OnUpdate(wrappers, markets)
	return nil
}

func (is RebalancerStrategy) OnError(err error) {
	fmt.Println("RebalancerStrategy OnError")
	fmt.Println(err)
}

func (is RebalancerStrategy) TearDown(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) error {
	fmt.Println("RebalancerStrategy TearDown")
	return nil
}

func NewRebalancerStrategy(raw_strat environment.StrategyConfig) strategies.Strategy {
	//TODO validation

	return &RebalancerStrategy{
		IntervalStrategy:    *NewIntervalStrategy(raw_strat),
		AllowanceThreshold:  decimal.NewFromFloat(raw_strat.Spec["allowance_threshold"].(float64)),
		MarketCapMultiplier: decimal.NewFromFloat(raw_strat.Spec["market_cap_multiplier"].(float64)),
		StaticCoin:          raw_strat.Spec["static_coin"].(string),
	}
}
