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

package strategies

import (
	"errors"
	"fmt"
	"sync"

	"github.com/saniales/golang-crypto-trading-bot/environment"
	"github.com/saniales/golang-crypto-trading-bot/exchanges"
)

var available map[string]Strategy //mapped name -> strategy
var appliedTactics []Tactic

// Strategy represents a generic strategy.
type Strategy interface {
	//CreateFromSpec(environment.BaseStrategyConfig) Strategy
	GetName() string // Name returns the name of the strategy.
	//Apply([]exchanges.ExchangeWrapper, []*environment.Market) // Apply applies the strategy when called, using the specified wrapper.
	Setup([]exchanges.ExchangeWrapper, []*environment.Market) error
	TearDown([]exchanges.ExchangeWrapper, []*environment.Market) error
	OnUpdate([]exchanges.ExchangeWrapper, []*environment.Market) error
	OnError(error)
}

// StrategyModel represents a strategy model used by strategies.
type StrategyModel struct {
	Name string
}

func NewBaseStrategy(raw_strat environment.StrategyConfig) *StrategyModel {
	return &StrategyModel{
		Name: raw_strat.Spec["name"].(string),
	}
}

// Name returns the name of the strategy.
func (is StrategyModel) GetName() string {
	return is.Name
}

// String returns a string representation of the object.
func (is StrategyModel) String() string {
	return is.GetName()
}

// Apply executes Cyclically the On Update, basing on provided interval.

func (is StrategyModel) Setup(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) error {
	fmt.Println("Base Setup")
	return nil
}

func (is StrategyModel) OnUpdate(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) error {
	return errors.New("BaseStrategy OnUpdate not implemented")
}

func (is StrategyModel) OnError(err error) {
	fmt.Println("Base OnError")
	fmt.Println(err)
}

func (is StrategyModel) TearDown(wrappers []exchanges.ExchangeWrapper, markets []*environment.Market) error {
	fmt.Println("Base TearDown")
	return nil
}

// Tactic represents the effective appliance of a strategy.
type Tactic struct {
	Markets  []*environment.Market
	Strategy Strategy
}

// Execute executes effectively a tactic.
func (t *Tactic) Execute(wrappers []exchanges.ExchangeWrapper) {
	Apply(wrappers, t.Strategy, t.Markets)
}

func init() {
	available = make(map[string]Strategy)
}

// AddCustomStrategy adds a strategy to the available set.
func AddCustomStrategy(s Strategy) string {
	available[s.GetName()] = s
	return s.GetName()
}

// MatchWithMarkets matches a strategy with the markets.
func MatchWithMarkets(strategyName string, markets []*environment.Market) error {
	s, exists := available[strategyName]
	if !exists {
		return fmt.Errorf("Strategy %s does not exist, cannot bind to markets %v", strategyName, markets)
	}
	appliedTactics = append(appliedTactics, Tactic{
		Markets:  markets,
		Strategy: s,
	})
	return nil
}

func Apply(wrappers []exchanges.ExchangeWrapper, strategy Strategy, markets []*environment.Market) {
	var err error

	err = strategy.Setup(wrappers, markets)
	if err != nil {
		strategy.OnError(err)
	}

	for err == nil {
		err = strategy.OnUpdate(wrappers, markets)
		if err != nil {
			strategy.OnError(err)
		}
	}

	err = strategy.TearDown(wrappers, markets)
	if err != nil {
		strategy.OnError(err)
	}

}

// ApplyAllStrategies applies all matched strategies concurrently.
func ApplyAllStrategies(wrappers []exchanges.ExchangeWrapper) {
	var wg sync.WaitGroup
	wg.Add(len(appliedTactics))
	for _, t := range appliedTactics {
		go func(wrappers []exchanges.ExchangeWrapper, t Tactic, wg *sync.WaitGroup) {
			defer wg.Done()
			t.Execute(wrappers)
		}(wrappers, t, &wg)
	}
	wg.Wait()
}
