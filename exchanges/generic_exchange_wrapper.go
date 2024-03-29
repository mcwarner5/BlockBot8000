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

package exchanges

import (
	"errors"
	"time"

	"github.com/mcwarner5/BlockBot8000/environment"
	"github.com/shopspring/decimal"
)

// ExchangeWrapper provides a generic wrapper for exchange services.
type ExchangeWrapper interface {
	Name() string                                                                                                                     // Gets the name of the exchange.
	GetCandles(market *environment.Market) ([]environment.CandleStick, error)                                                         // Gets the candle data from the exchange.
	GetHistoricalCandles(market *environment.Market, start time.Time, end time.Time, interval int) ([]environment.CandleStick, error) // Gets the candle data from the exchange.
	GetMarkets() ([]*environment.Market, error)
	GetMarketSummary(market *environment.Market) (*environment.MarketSummary, error) // Gets the current market summary.
	GetOrderBook(market *environment.Market) (*environment.OrderBook, error)         // Gets the order(ASK + BID) book of a market.

	BuyLimit(market *environment.Market, amount decimal.Decimal, limit decimal.Decimal) (string, error)  // Performs a limit buy action.
	SellLimit(market *environment.Market, amount decimal.Decimal, limit decimal.Decimal) (string, error) // Performs a limit sell action.
	BuyMarket(market *environment.Market, amount decimal.Decimal) (string, error)                        // Performs a market buy action.
	SellMarket(market *environment.Market, amount decimal.Decimal) (string, error)                       // Performs a market sell action.

	GetHistoricalTrades(market *environment.Market, start time.Time, end time.Time) (*environment.TradeBook, error)
	GetAllTrades(markets []*environment.Market) (*environment.TradeBook, error)
	GetAllMarketTrades(market *environment.Market) (*environment.TradeBook, error)
	GetFilteredTrades(market *environment.Market, symbol string, tradeSide environment.TradeSide, tradeType environment.TradeType, tradeStatus environment.TradeStatus) (*environment.TradeBook, error)
	CalculateTradingFees(market *environment.Market, amount decimal.Decimal, limit decimal.Decimal, orderSide environment.TradeSide) decimal.Decimal // Calculates the trading fees for an order on a specified market.
	CalculateWithdrawFees(market *environment.Market, amount decimal.Decimal) decimal.Decimal                                                        // Calculates the withdrawal fees on a specified market.

	GetBalance(symbol string) (*decimal.Decimal, error) // Gets the balance of the user of the specified currency.
	GetDepositAddress(coinTicker string) (string, bool) // Gets the deposit address for the specified coin on the exchange, if exists.

	FeedConnect(markets []*environment.Market) error // Connects to the feed of the exchange.

	Withdraw(destinationAddress string, coinTicker string, amount decimal.Decimal) error // Performs a withdraw operation from the exchange to a destination address.

	String() string // Returns a string representation of the object.
	IsHistoricalSimulation() bool
}

// ErrWebsocketNotSupported is the error representing when an exchange does not support websocket.
var ErrWebsocketNotSupported = errors.New("cannot use websocket: exchange does not support it")

// MarketNameFor gets the market name as seen by the exchange.
func MarketNameFor(m *environment.Market, wrapper ExchangeWrapper) string {
	return m.ExchangeNames[wrapper.Name()]
}
