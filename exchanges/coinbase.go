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
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"github.com/mcwarner5/BlockBot8000/environment"
	client "github.com/mcwarner5/BlockBot8000/libraries/coinbase-adv/client"
	"github.com/mcwarner5/BlockBot8000/libraries/coinbase-adv/model"
	"github.com/shopspring/decimal"
)

// coinbaseWrapper represents the wrapper for the coinbase exchange.
type CoinbaseWrapper struct {
	api              client.CoinbaseClient
	summaries        *SummaryCache
	candles          *CandlesCache
	orderbook        *OrderbookCache
	depositAddresses map[string]string
	websocketOn      bool
}

// NewCoinbaseWrapper creates a generic wrapper of the coinbase API.
func NewCoinbaseWrapper(publicKey string, secretKey string, depositAddresses map[string]string) ExchangeWrapper {
	creds := client.Credentials{
		ApiKey:      publicKey,
		ApiSKey:     secretKey,
		AccessToken: "",
	}

	return &CoinbaseWrapper{
		api:              client.NewClient(&creds),
		summaries:        NewSummaryCache(),
		candles:          NewCandlesCache(),
		orderbook:        NewOrderbookCache(),
		depositAddresses: depositAddresses,
		websocketOn:      false,
	}
}

// Name returns the name of the wrapped exchange.
func (wrapper *CoinbaseWrapper) Name() string {
	return "coinbase"
}

func (wrapper *CoinbaseWrapper) String() string {
	return wrapper.Name()
}

// GetMarkets Gets all the markets info.
func (wrapper *CoinbaseWrapper) GetMarkets() ([]*environment.Market, error) {
	ctx := context.Background()
	var start int32 = 1

	wrappedMarkets := make([]*environment.Market, 0, 400)
	for i := start; i < 4; i++ {
		var params = client.ListProductsParams{
			Limit:  client.MaxLimit,
			Offset: client.MaxLimit * (i - 1),
		}

		res_products, err := wrapper.api.ListProducts(ctx, &params)
		if err != nil {
			return nil, err
		}

		for _, product := range res_products.Products {
			wrappedMarkets = append(wrappedMarkets, &environment.Market{
				Name:           *product.ProductId,
				BaseCurrency:   *product.BaseCurrencyId,
				MarketCurrency: *product.QuoteCurrencyId,
			})
		}

		var sleep_len = time.Duration(1) * time.Second
		time.Sleep(sleep_len)
	}
	return wrappedMarkets, nil
}

// GetOrderBook gets the order(ASK + BID) book of a market.
func (wrapper *CoinbaseWrapper) GetOrderBook(market *environment.Market) (*environment.OrderBook, error) {
	if !wrapper.websocketOn {
		orderbook, err := wrapper.orderbookFromREST(market)
		if err != nil {
			return nil, err
		}

		wrapper.orderbook.Set(market, orderbook)
		return orderbook, nil
	}

	orderbook, exists := wrapper.orderbook.Get(market)
	if !exists {
		return nil, errors.New("orderbook not loaded")
	}

	return orderbook, nil
}

func (wrapper *CoinbaseWrapper) orderbookFromREST(market *environment.Market) (*environment.OrderBook, error) {

	params := client.GetProductBookParams{
		Product: MarketNameFor(market, wrapper),
		Limit:   client.MaxLimit,
	}

	coinbaseProdcutBookResponse, err := wrapper.api.GetProductBook(context.Background(), &params)
	if err != nil {
		return nil, err
	}
	var orderBook environment.OrderBook
	pricebook := coinbaseProdcutBookResponse.GetPriceBook()

	for _, ask := range pricebook.GetAsks() {
		qty := decimal.NewFromFloat(*ask.Size)
		value := decimal.NewFromFloat(*ask.Price)

		orderBook.Asks = append(orderBook.Asks, environment.Order{
			Quantity: qty,
			Value:    value,
		})
	}

	for _, bid := range pricebook.GetAsks() {
		qty := decimal.NewFromFloat(*bid.Size)
		value := decimal.NewFromFloat(*bid.Price)

		orderBook.Bids = append(orderBook.Bids, environment.Order{
			Quantity: qty,
			Value:    value,
		})
	}

	return &orderBook, nil
}

// BuyLimit performs a limit buy action.
func (wrapper *CoinbaseWrapper) BuyLimit(market *environment.Market, amount float64, limit float64) (string, error) {
	amount_str := fmt.Sprint(amount)
	limit_str := fmt.Sprint(limit)
	order_confg := model.CreateOrderRequestOrderConfiguration{
		MarketMarketIoc: model.NewCreateOrderRequestOrderConfigurationMarketMarketIoc(),
		LimitLimitGtc: &model.CreateOrderRequestOrderConfigurationLimitLimitGtc{
			BaseSize:   &amount_str,
			LimitPrice: &limit_str,
			PostOnly:   model.PtrBool(true),
		},
		LimitLimitGtd:         model.NewCreateOrderRequestOrderConfigurationLimitLimitGtd(),
		StopLimitStopLimitGtc: model.NewCreateOrderRequestOrderConfigurationStopLimitStopLimitGtc(),
		StopLimitStopLimitGtd: model.NewCreateOrderRequestOrderConfigurationStopLimitStopLimitGtd(),
	}
	new_order_id, _ := uuid.NewV4()
	new_order_id_str := new_order_id.String()
	market_name := MarketNameFor(market, wrapper)
	side, _ := model.NewOrderSideFromValue("BUY")

	order := model.CreateOrderRequest{
		ClientOrderId:      &new_order_id_str,
		ProductId:          &market_name,
		Side:               (*string)(side.Ptr()),
		OrderConfiguration: &order_confg,
	}

	orderResponse, err := wrapper.api.CreateOrder(context.Background(), &order)
	if err != nil {
		return "", err
	}
	if orderResponse == nil || orderResponse.OrderId == nil || !(*orderResponse.Success) {
		return "", errors.New(*orderResponse.FailureReason)
	}
	return *orderResponse.OrderId, nil
}

// SellLimit performs a limit sell action.
func (wrapper *CoinbaseWrapper) SellLimit(market *environment.Market, amount float64, limit float64) (string, error) {
	amount_str := fmt.Sprint(amount)
	limit_str := fmt.Sprint(limit)
	order_confg := model.CreateOrderRequestOrderConfiguration{
		MarketMarketIoc: model.NewCreateOrderRequestOrderConfigurationMarketMarketIoc(),
		LimitLimitGtc: &model.CreateOrderRequestOrderConfigurationLimitLimitGtc{
			BaseSize:   &amount_str,
			LimitPrice: &limit_str,
			PostOnly:   model.PtrBool(true),
		},
		LimitLimitGtd:         model.NewCreateOrderRequestOrderConfigurationLimitLimitGtd(),
		StopLimitStopLimitGtc: model.NewCreateOrderRequestOrderConfigurationStopLimitStopLimitGtc(),
		StopLimitStopLimitGtd: model.NewCreateOrderRequestOrderConfigurationStopLimitStopLimitGtd(),
	}
	new_order_id, _ := uuid.NewV4()
	new_order_id_str := new_order_id.String()
	market_name := MarketNameFor(market, wrapper)
	side, _ := model.NewOrderSideFromValue("SELL")

	order := model.CreateOrderRequest{
		ClientOrderId:      &new_order_id_str,
		ProductId:          &market_name,
		Side:               (*string)(side.Ptr()),
		OrderConfiguration: &order_confg,
	}

	orderResponse, err := wrapper.api.CreateOrder(context.Background(), &order)
	if err != nil {
		return "", err
	}
	if orderResponse == nil || orderResponse.OrderId == nil || !(*orderResponse.Success) {
		return "", errors.New(*orderResponse.FailureReason)
	}

	return *orderResponse.OrderId, nil
}

// BuyMarket performs a market buy action.
func (wrapper *CoinbaseWrapper) BuyMarket(market *environment.Market, amount float64) (string, error) {
	amount_str := fmt.Sprint(amount)
	order_confg := model.CreateOrderRequestOrderConfiguration{
		MarketMarketIoc: &model.CreateOrderRequestOrderConfigurationMarketMarketIoc{
			QuoteSize: &amount_str,
		},
		LimitLimitGtc:         model.NewCreateOrderRequestOrderConfigurationLimitLimitGtc(),
		LimitLimitGtd:         model.NewCreateOrderRequestOrderConfigurationLimitLimitGtd(),
		StopLimitStopLimitGtc: model.NewCreateOrderRequestOrderConfigurationStopLimitStopLimitGtc(),
		StopLimitStopLimitGtd: model.NewCreateOrderRequestOrderConfigurationStopLimitStopLimitGtd(),
	}
	new_order_id, _ := uuid.NewV4()
	new_order_id_str := new_order_id.String()
	market_name := MarketNameFor(market, wrapper)
	side, _ := model.NewOrderSideFromValue("BUY")

	order := model.CreateOrderRequest{
		ClientOrderId:      &new_order_id_str,
		ProductId:          &market_name,
		Side:               (*string)(side.Ptr()),
		OrderConfiguration: &order_confg,
	}

	orderResponse, err := wrapper.api.CreateOrder(context.Background(), &order)
	if err != nil {
		return "", err
	}
	if orderResponse == nil || orderResponse.OrderId == nil || !(*orderResponse.Success) {
		return "", errors.New(*orderResponse.FailureReason)
	}
	return *orderResponse.OrderId, nil
}

// SellMarket performs a market sell action.
func (wrapper *CoinbaseWrapper) SellMarket(market *environment.Market, amount float64) (string, error) {
	amount_str := fmt.Sprint(amount)
	order_confg := model.CreateOrderRequestOrderConfiguration{
		MarketMarketIoc: &model.CreateOrderRequestOrderConfigurationMarketMarketIoc{
			BaseSize: &amount_str,
		},
		LimitLimitGtc:         model.NewCreateOrderRequestOrderConfigurationLimitLimitGtc(),
		LimitLimitGtd:         model.NewCreateOrderRequestOrderConfigurationLimitLimitGtd(),
		StopLimitStopLimitGtc: model.NewCreateOrderRequestOrderConfigurationStopLimitStopLimitGtc(),
		StopLimitStopLimitGtd: model.NewCreateOrderRequestOrderConfigurationStopLimitStopLimitGtd(),
	}
	new_order_id, _ := uuid.NewV4()
	new_order_id_str := new_order_id.String()
	market_name := MarketNameFor(market, wrapper)
	side, _ := model.NewOrderSideFromValue("SELL")

	order := model.CreateOrderRequest{
		ClientOrderId:      &new_order_id_str,
		ProductId:          &market_name,
		Side:               (*string)(side.Ptr()),
		OrderConfiguration: &order_confg,
	}

	orderResponse, err := wrapper.api.CreateOrder(context.Background(), &order)
	if err != nil {
		return "", err
	}
	if orderResponse == nil || orderResponse.OrderId == nil || !(*orderResponse.Success) {
		return "", errors.New(*orderResponse.FailureReason)
	}
	return *orderResponse.OrderId, nil
}

func (wrapper *CoinbaseWrapper) GetHistoricalTrades(market *environment.Market, start time.Time, end time.Time) (*environment.TradeBook, error) {
	var params = client.ListProductsTickerHistoryParams{
		ProductId: MarketNameFor(market, wrapper),
		StartTime: start,
		EndTime:   end,
	}
	response, err := wrapper.api.ListProductsTickerHistory(context.Background(), &params)

	if err != nil {
		return nil, err
	}
	result := environment.NewSizedTradeBook(len(response.Trades))

	for _, trade := range response.Trades {
		trade_time, err := time.Parse(time.RFC3339, *trade.Time)
		if err != nil {
			return nil, err
		}
		if *trade.Side == "UNKNOWN_ORDER_SIDE" {
			continue
		}

		trade_side, err := environment.TradeSideFromString(*trade.Side)
		if err != nil {
			return nil, err
		}

		result.Trades = append(result.Trades, environment.Trade{
			Price:        decimal.NewFromFloat(*trade.Price),
			AskQuantity:  decimal.NewFromFloat(*trade.Size),
			FillQuantity: decimal.NewFromFloat(*trade.Size),
			Market:       *trade.ProductId,
			Side:         trade_side,
			Status:       environment.Complete,
			Type:         environment.MarketPrice,
			TradeNumber:  *trade.TradeId,
			Timestamp:    trade_time,
		})
	}

	return result, nil
}

// GetTicker gets the updated ticker for a market.
func (wrapper *CoinbaseWrapper) GetTicker(market *environment.Market) (*environment.Ticker, error) {
	var params = client.ListProductsTickerHistoryParams{
		ProductId: MarketNameFor(market, wrapper),
	}
	ticker, err := wrapper.api.ListProductsTickerHistory(context.Background(), &params)
	if err != nil {
		return nil, err
	}

	ask, _ := decimal.NewFromString(*ticker.BestAsk)
	bid, _ := decimal.NewFromString(*ticker.BestBid)

	return &environment.Ticker{
		Last: ask,
		Ask:  ask,
		Bid:  bid,
	}, nil
}

// GetMarketSummary gets the current market summary.
func (wrapper *CoinbaseWrapper) GetMarketSummary(market *environment.Market) (*environment.MarketSummary, error) {
	if !wrapper.websocketOn {

		var candle_params = client.ListProductsCandlesParams{
			Product:   MarketNameFor(market, wrapper),
			StartTime: time.Now().Add(-2 * time.Minute),
			EndTime:   time.Now(),
			Interval:  1,
		}

		coinbaseCandles, err := wrapper.api.GetProductCandles(context.Background(), &candle_params)
		if err != nil {
			return nil, err
		}
		curr_candles := coinbaseCandles.GetCandleSticks()
		if len(curr_candles) == 0 {
			return nil, errors.New("no Candles Found for Coinbase MarketSummary")
		}

		var params = client.ListProductsTickerHistoryParams{
			ProductId: MarketNameFor(market, wrapper),
		}
		ticker, err := wrapper.api.ListProductsTickerHistory(context.Background(), &params)
		if err != nil {
			return nil, err
		}

		ask, _ := decimal.NewFromString(*ticker.Trades[0].Ask)
		bid, _ := decimal.NewFromString(*ticker.Trades[0].Bid)
		high, _ := decimal.NewFromString(*curr_candles[0].High)
		low, _ := decimal.NewFromString(*curr_candles[0].Low)
		last, _ := decimal.NewFromString(*curr_candles[0].Open)
		volume := decimal.NewFromFloat(*ticker.Trades[0].Size)

		wrapper.summaries.Set(market, &environment.MarketSummary{
			Last:   last,
			Ask:    ask,
			Bid:    bid,
			High:   high,
			Low:    low,
			Volume: volume,
		})
	}

	ret, summaryLoaded := wrapper.summaries.Get(market)
	if !summaryLoaded {
		return nil, errors.New("summary not loaded")
	}

	return ret, nil
}

func (wrapper *CoinbaseWrapper) GetHistoricalCandles(market *environment.Market, start time.Time, end time.Time, interval int) ([]environment.CandleStick, error) {
	var params = client.ListProductsCandlesParams{
		Product:   MarketNameFor(market, wrapper),
		StartTime: start,
		EndTime:   end,
		Interval:  interval,
	}
	response, err := wrapper.api.GetProductCandles(context.Background(), &params)

	if err != nil {
		return nil, err
	}

	coinbaseCandles := response.GetCandleSticks()

	sort.Slice(coinbaseCandles, func(i, j int) bool {
		m_t_u_str, _ := strconv.ParseInt(*coinbaseCandles[i].Start, 10, 64)
		o_t_u_str, _ := strconv.ParseInt(*coinbaseCandles[j].Start, 10, 64)
		m_t := time.Unix(m_t_u_str, 0).UTC()
		o_t := time.Unix(o_t_u_str, 0).UTC()
		return m_t.Before(o_t)
	})

	ret := make([]environment.CandleStick, len(coinbaseCandles))

	for i, coinbaseCandle := range coinbaseCandles {
		high, _ := decimal.NewFromString(*coinbaseCandle.High)
		open, _ := decimal.NewFromString(*coinbaseCandle.Open)
		close, _ := decimal.NewFromString(*coinbaseCandle.Close)
		low, _ := decimal.NewFromString(*coinbaseCandle.Low)
		volume, _ := decimal.NewFromString(*coinbaseCandle.Volume)
		time_unix_num, _ := strconv.ParseInt(*coinbaseCandle.Start, 10, 64)
		time := time.Unix(time_unix_num, 0).UTC()

		ret[i] = environment.CandleStick{
			High:       high,
			Open:       open,
			Close:      close,
			Low:        low,
			Volume:     volume,
			CandleTime: time,
		}
	}

	return ret, nil
}

// GetCandles gets the candle data from the exchange.
func (wrapper *CoinbaseWrapper) GetCandles(market *environment.Market) ([]environment.CandleStick, error) {
	if !wrapper.websocketOn {
		var params = client.ListProductsCandlesParams{
			Product:   MarketNameFor(market, wrapper),
			StartTime: time.Now().Add(-24 * time.Hour),
			EndTime:   time.Now(),
			Interval:  1,
		}

		coinbaseCandles, err := wrapper.api.GetProductCandles(context.Background(), &params)
		if err != nil {
			return nil, err
		}

		ret := make([]environment.CandleStick, len(coinbaseCandles.CandleSticks))

		for i, coinbaseCandle := range coinbaseCandles.GetCandleSticks() {
			high, _ := decimal.NewFromString(*coinbaseCandle.High)
			open, _ := decimal.NewFromString(*coinbaseCandle.Open)
			close, _ := decimal.NewFromString(*coinbaseCandle.Close)
			low, _ := decimal.NewFromString(*coinbaseCandle.Low)
			volume, _ := decimal.NewFromString(*coinbaseCandle.Volume)

			ret[i] = environment.CandleStick{
				High:   high,
				Open:   open,
				Close:  close,
				Low:    low,
				Volume: volume,
			}
		}

		wrapper.candles.Set(market, ret)
	}

	ret, candleLoaded := wrapper.candles.Get(market)
	if !candleLoaded {
		return nil, errors.New("no candle data yet")
	}

	return ret, nil
}

// GetBalance gets the balance of the user of the specified currency.
func (wrapper *CoinbaseWrapper) GetBalance(symbol string) (*decimal.Decimal, error) {

	coinbaseAccount, err := wrapper.api.GetAccount(context.Background(), symbol)
	if err != nil {
		return nil, err
	}

	if *coinbaseAccount.Currency == symbol {
		if coinbaseAccount.AvailableBalance == nil || coinbaseAccount.AvailableBalance.Value == nil {
			return nil, errors.New("available balance not found")
		}

		ret := decimal.NewFromFloat(*coinbaseAccount.AvailableBalance.Value)
		return &ret, nil
	}

	return nil, errors.New("symbol not found")
}

// GetDepositAddress gets the deposit address for the specified coin on the exchange.
func (wrapper *CoinbaseWrapper) GetDepositAddress(coinTicker string) (string, bool) {
	addr, exists := wrapper.depositAddresses[coinTicker]
	return addr, exists
}

func (wrapper *CoinbaseWrapper) GetAllTrades(markets []*environment.Market) (*environment.TradeBook, error) {
	panic("GetAllTrades Not supported on Coinbase wrapper yet")
}
func (wrapper *CoinbaseWrapper) GetAllMarketTrades(market *environment.Market) (*environment.TradeBook, error) {
	panic("GetAllMarketTrades Not supported on Coinbase wrapper yet")
}

func (wrapper *CoinbaseWrapper) GetFilteredTrades(market *environment.Market, symbol string, tradeSide environment.TradeSide, tradeType environment.TradeType, tradeStatus environment.TradeStatus) (*environment.TradeBook, error) {
	panic("GetFilteredTrades Not supported on Coinbase wrapper yet")
}

// NOTE: In Coinbase fees are currently hardcoded.
func (wrapper *CoinbaseWrapper) CalculateTradingFees(market *environment.Market, amount float64, limit float64, orderSide environment.TradeSide) float64 {
	var feePercentage float64
	if orderSide == environment.Sell {
		feePercentage = 0.0025
	} else if orderSide == environment.Buy {
		feePercentage = 0.0025
	} else {
		panic("Unknown trade type")
	}

	return amount * limit * feePercentage
}

// CalculateWithdrawFees calculates the withdrawal fees on a specified market.
func (wrapper *CoinbaseWrapper) CalculateWithdrawFees(market *environment.Market, amount float64) float64 {
	panic("CalculateWithdrawFees Not Implemented for Coinbase wrapper yet")
}

// FeedConnect connects to the feed of the exchange.
func (wrapper *CoinbaseWrapper) FeedConnect(markets []*environment.Market) error {
	panic("FeedConnect Not supported on Coinbase wrapper yet")
}

// Withdraw performs a withdraw operation from the exchange to a destination address.
func (wrapper *CoinbaseWrapper) Withdraw(destinationAddress string, coinTicker string, amount float64) error {
	panic("subscribeOrderbookFeed Not supported on Coinbase wrapper yet")
}

func (wrapper *CoinbaseWrapper) IsHistoricalSimulation() bool {
	return false
}
