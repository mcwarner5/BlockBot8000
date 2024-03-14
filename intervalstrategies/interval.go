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
	"reflect"
	"time"

	"github.com/mcwarner5/BlockBot8000/environment"
	"github.com/mcwarner5/BlockBot8000/exchanges"
	"github.com/mcwarner5/BlockBot8000/strategies"
)

// IntervalStrategy is an interval based strategy.
type IntervalStrategy struct {
	strategies.StrategyModel
	Interval int
}

func (is IntervalStrategy) String() string {
	return "Type: " + reflect.TypeOf(is).String() + " Name:" + is.GetName()
}

func (is IntervalStrategy) OnUpdate(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) (strategies.Strategy, error) {
	//fmt.Println("OnUpdate " + is.String())
	if wrappers[0].Name() == "simulator" {
		return is, nil
	}

	var sleep_len = time.Duration(is.Interval) * time.Minute
	time.Sleep(sleep_len)
	return is, nil
}

func NewIntervalStrategy(raw_strat environment.StrategyConfig) *IntervalStrategy {
	//TODO validation

	return &IntervalStrategy{
		StrategyModel: *strategies.NewBaseStrategy(raw_strat),
		Interval:      raw_strat.Spec["interval"].(int),
	}
}
