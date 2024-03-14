package exchanges

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"github.com/juju/errors"
	"github.com/mcwarner5/BlockBot8000/environment"
	"github.com/mcwarner5/BlockBot8000/libraries/coinbase-adv/client"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

var CoinbaseMaxLimit int = 300

// ExchangeWrapperSimulator wraps another wrapper and returns simulated balances and orders.
type ExchangeWrapperSimulator struct {
	innerWrapper ExchangeWrapper
	candles      *MappedCandlesCache
	orders       *MappedOrdersCache
	coinbase     client.CoinbaseClient
	balances     map[string]decimal.Decimal
	interval     int
	startDate    *time.Time
	endDate      time.Time
	currDate     *time.Time
}

// NewExchangeWrapperSimulator creates a new simulated wrapper from another wrapper and an initial balance.
func NewExchangeWrapperSimulator(mockedWrapper ExchangeWrapper, simConfigs environment.SimulationConfig) *ExchangeWrapperSimulator {

	start_date, err := time.Parse(time.DateOnly, simConfigs.SimStartDate)
	curr_date, _ := time.Parse(time.DateOnly, simConfigs.SimStartDate)

	if err != nil {
		panic("no start date")
	}

	end_date, err := time.Parse(time.DateOnly, simConfigs.SimEndDate)

	if err != nil {
		panic("no end date")
	}

	creds := client.Credentials{
		ApiKey:      simConfigs.SimPublicKey,
		ApiSKey:     simConfigs.SimSecretKey,
		AccessToken: "",
	}

	return &ExchangeWrapperSimulator{
		innerWrapper: mockedWrapper,
		candles:      NewMappedCandlesCache(),
		orders:       NewMappedOrdersCache(),
		coinbase:     client.NewClient(&creds),
		balances:     simConfigs.SimFakeBalances,
		interval:     simConfigs.SimInterval,
		startDate:    &start_date,
		endDate:      end_date,
		currDate:     &curr_date,
	}
}

// String returns a string representation of the exchange simulator.
func (wrapper *ExchangeWrapperSimulator) String() string {
	return fmt.Sprint(wrapper.innerWrapper.Name(), "_simulator")
}

// Name gets the name of the exchange.
func (wrapper *ExchangeWrapperSimulator) Name() string {
	return "simulator"
}

func (wrapper *ExchangeWrapperSimulator) IncrementCurrDate() error {
	logrus.Info("End of Interval:" + wrapper.currDate.String())
	var interval_len = time.Duration(wrapper.interval) * time.Minute
	*wrapper.currDate = wrapper.currDate.Add(interval_len)

	if wrapper.currDate.After(wrapper.endDate) {
		logrus.Info("End of Simulation:" + wrapper.currDate.String())
		return errors.New("End of Simulation Date has been reached")
	}

	return nil
}

// GetCandles gets the candle data from the exchange.
func (wrapper *ExchangeWrapperSimulator) UpdateMappedCandles(market *environment.Market, from_time time.Time) (*environment.CandleStick, error) {
	ctx := context.Background()
	var one_interval = time.Duration(wrapper.interval) * time.Minute
	var api_call_len = time.Duration(wrapper.interval*300) * time.Minute
	var api_end_date = from_time.Add(-one_interval + api_call_len)
	var from_time_key = fmt.Sprint(from_time.Add(-one_interval).Unix())

	var params = client.ListProductsCandlesParams{
		Product:   MarketNameFor(market, wrapper),
		StartTime: from_time_key,
		EndTime:   fmt.Sprint(api_end_date.Unix()),
		Interval:  wrapper.interval,
	}
	response, _ := wrapper.coinbase.GetProductCandles(ctx, &params)
	new_map := NewSizedCandleMap(len(response.CandleSticks))

	for _, candle := range response.GetCandleSticks() {
		u_num, _ := strconv.ParseInt(*candle.Start, 10, 64)

		c_high, _ := decimal.NewFromString(*candle.High)
		c_open, _ := decimal.NewFromString(*candle.Open)
		c_low, _ := decimal.NewFromString(*candle.Low)
		c_close, _ := decimal.NewFromString(*candle.Close)
		c_volume, _ := decimal.NewFromString(*candle.Volume)
		c_time := time.Unix(u_num, 0)
		c_time_key := fmt.Sprint(c_time.Unix())

		new_map.TimeMap[c_time_key] = &environment.CandleStick{
			High:       c_high,
			Open:       c_open,
			Close:      c_close,
			Low:        c_low,
			Volume:     c_volume,
			CandleTime: c_time,
		}
	}
	wrapper.candles.SetMap(market, new_map)
	candle, isSet := wrapper.candles.GetTime(market, from_time)
	if !isSet {
		logrus.Info("no value found for time" + fmt.Sprint(from_time.Unix()))
		for i := 1; i < 7; i++ {
			//try to cover an outage with value from previous interval
			candle, isSet = wrapper.candles.GetTime(market, from_time.Add(-one_interval*time.Duration(i)))
			if isSet {
				return candle, nil
			}

			//try to cover with the next interval if previous is also empty
			candle, isSet = wrapper.candles.GetTime(market, from_time.Add(one_interval*time.Duration(i)))
			if isSet {
				return candle, nil
			}
		}
		return nil, errors.New("no data for that time  set panic")
	}

	return candle, nil
}

func (wrapper *ExchangeWrapperSimulator) GetCandle(market *environment.Market, time time.Time) (*environment.CandleStick, error) {
	candle, isSet := wrapper.candles.GetTime(market, time)

	if !isSet {
		new_candle, err := wrapper.UpdateMappedCandles(market, time)
		if err != nil {
			return nil, err
		}
		return new_candle, nil
	}

	return candle, nil
}

// GetCandles gets the candle data from the exchange.
func (wrapper *ExchangeWrapperSimulator) GetCandles(market *environment.Market) ([]environment.CandleStick, error) {
	return wrapper.innerWrapper.GetCandles(market)
}

func (wrapper *ExchangeWrapperSimulator) GetProducts() {
	ctx := context.Background()
	var start int32 = 1

	for i := start; i < 4; i++ {
		var params = client.ListProductsParams{
			Limit:  int32(CoinbaseMaxLimit),
			Offset: int32(CoinbaseMaxLimit)*(i-1) + 1,
			//ProductType: "SPOT",
		}

		res_products, _ := wrapper.coinbase.ListProducts(ctx, &params)

		var products string
		for _, product := range res_products.Products {
			products = products + fmt.Sprintln(*product.ProductId)
		}

		logrus.Info(products)
		var sleep_len = time.Duration(10) * time.Second
		time.Sleep(sleep_len)
	}
}

// GetMarketSummary gets the current market summary.
func (wrapper *ExchangeWrapperSimulator) GetMarketSummary(market *environment.Market) (*environment.MarketSummary, error) {
	var c_time *time.Time = &time.Time{}
	*c_time = *wrapper.currDate

	candle, err := wrapper.GetCandle(market, *c_time)

	if err != nil {
		return nil, err
	}

	return &environment.MarketSummary{
		High:   candle.High,
		Low:    candle.Low,
		Volume: candle.Volume,
		Last:   candle.Open,
		Ask:    candle.Open,
		Bid:    candle.Close,
	}, nil
}

func (wrapper *ExchangeWrapperSimulator) UpdateMappedOrders(market *environment.Market, from_time time.Time) (*environment.OrderBook, error) {
	ctx := context.Background()
	//var to_include_first = time.Duration(wrapper.interval) * time.Minute
	//var api_call_len = time.Duration(wrapper.interval*300) * time.Minute
	var thirty_min = time.Duration(30) * time.Minute

	var api_end_date = from_time.Add(thirty_min)

	var params = client.ListProductsTickerHistoryParams{
		ProductId: MarketNameFor(market, wrapper),
		StartTime: from_time,
		EndTime:   api_end_date,
	}
	response, _ := wrapper.coinbase.ListProductsTickerHistory(ctx, &params)

	new_map := OrderBookMap{TimeMap: make(map[string]*environment.OrderBook)}
	curr_map_time := from_time

	var c_asks []environment.Order = make([]environment.Order, 0)
	var c_bids []environment.Order = make([]environment.Order, 0)

	for i := len(response.Trades) - 1; i >= 0; i-- {
		curr_trade := response.Trades[i]
		trade_time, err := time.Parse(time.RFC3339, *curr_trade.Time)
		if err != nil {
			return nil, err
		}

		if trade_time.Before(curr_map_time) {
			continue
		}

		if trade_time.After(curr_map_time) {
			new_order := environment.Order{
				Value:       decimal.NewFromFloat(*curr_trade.Price), //Value of the trade : e.g. in a BTC ETH is the value of a single ETH in BTC.
				Quantity:    decimal.NewFromFloat(*curr_trade.Size),  //Quantity of Coins of this order.
				OrderNumber: *curr_trade.TradeId,                     //[optional] Order number as seen in echange archives.
				Timestamp:   trade_time,
			}

			if *curr_trade.Side == "BUY" {
				c_asks = append(c_asks, new_order)
			}

			if *curr_trade.Side == "SELL" {
				c_bids = append(c_bids, new_order)
			}
			continue
		}

	}

	curr_map_time_key := fmt.Sprint(curr_map_time.Unix())
	new_order_book := environment.OrderBook{
		Asks: c_asks,
		Bids: c_bids,
	}

	new_map.TimeMap[curr_map_time_key] = &new_order_book
	wrapper.orders.SetMap(market, new_map)

	order, isSet := wrapper.orders.GetTime(market, from_time)
	if !isSet {
		return nil, errors.New("no data for that time  set panic")
	}

	return order, nil
}

// GetOrderBook gets the order(ASK + BID) book of a market.
func (wrapper *ExchangeWrapperSimulator) GetOrderBook(market *environment.Market) (*environment.OrderBook, error) {
	order, isSet := wrapper.orders.GetTime(market, *wrapper.currDate)

	if !isSet {
		order, err := wrapper.UpdateMappedOrders(market, *wrapper.currDate)

		if err != nil {
			return nil, err
		}
		return order, nil
	}

	return order, nil
}

// BuyLimit here is just to implement the ExchangeWrapper Interface, do not use, use BuyMarket instead.
func (wrapper *ExchangeWrapperSimulator) BuyLimit(market *environment.Market, amount float64, limit float64) (string, error) {
	return "", errors.New("BuyLimit operation is not mockable")
}

// SellLimit here is just to implement the ExchangeWrapper Interface, do not use, use SellMarket instead.
func (wrapper *ExchangeWrapperSimulator) SellLimit(market *environment.Market, amount float64, limit float64) (string, error) {
	return "", errors.New("sellLimit operation is not mockable")
}

// BuyMarket performs a FAKE market buy action.
func (wrapper *ExchangeWrapperSimulator) BuyMarket(market *environment.Market, amount float64) (string, error) {
	baseBalance, _ := wrapper.GetBalance(market.BaseCurrency)
	quoteBalance, _ := wrapper.GetBalance(market.MarketCurrency)

	orderbook, err := wrapper.GetOrderBook(market)
	if err != nil {
		return "", errors.Annotate(err, "Cannot market buy without orderbook knowledge")
	}

	totalQuote := decimal.Zero
	remainingAmount := decimal.NewFromFloat(amount)
	expense := decimal.Zero

	for _, ask := range orderbook.Asks {
		if remainingAmount.LessThanOrEqual(ask.Quantity) {
			totalQuote = totalQuote.Add(remainingAmount)
			expense = expense.Add(remainingAmount.Mul(ask.Value))
			if expense.GreaterThan(*quoteBalance) {
				return "", fmt.Errorf("cannot Buy not enough %s balance", market.BaseCurrency)
			}
			break
		}
		totalQuote = totalQuote.Add(ask.Quantity)
		remainingAmount = remainingAmount.Sub(ask.Quantity)

		expense = expense.Add(ask.Quantity.Mul(ask.Value))
		if expense.GreaterThan(*quoteBalance) {
			return "", fmt.Errorf("cannot Buy not enough %s balance", market.BaseCurrency)
		}
	}

	wrapper.balances[market.BaseCurrency] = baseBalance.Add(totalQuote)
	wrapper.balances[market.MarketCurrency] = quoteBalance.Sub(expense)

	orderFakeID, err := uuid.NewV4()
	if err != nil {
		return "", errors.Annotate(err, "UUID Generation")
	}
	return fmt.Sprintf("FAKE_BUY-%s", orderFakeID), nil
}

// SellMarket performs a FAKE market buy action.
func (wrapper *ExchangeWrapperSimulator) SellMarket(market *environment.Market, amount float64) (string, error) {
	baseBalance, _ := wrapper.GetBalance(market.BaseCurrency)
	quoteBalance, _ := wrapper.GetBalance(market.MarketCurrency)

	orderbook, err := wrapper.GetOrderBook(market)
	if err != nil {
		return "", errors.Annotate(err, "cannot market sell without orderbook knowledge")
	}

	totalQuote := decimal.Zero
	remainingAmount := decimal.NewFromFloat(amount)
	gain := decimal.Zero

	if baseBalance.LessThan(remainingAmount) {
		return "", fmt.Errorf("cannot Sell: not enough %s balance", market.MarketCurrency)
	}

	for _, bid := range orderbook.Bids {
		if remainingAmount.LessThanOrEqual(bid.Quantity) {
			totalQuote = totalQuote.Add(remainingAmount)
			gain = gain.Add(remainingAmount.Mul(bid.Value))
			break
		}
		totalQuote = totalQuote.Add(bid.Quantity)
		remainingAmount = remainingAmount.Sub(bid.Quantity)
		gain = gain.Add(bid.Quantity.Mul(bid.Value))
	}

	wrapper.balances[market.BaseCurrency] = baseBalance.Sub(totalQuote)
	wrapper.balances[market.MarketCurrency] = quoteBalance.Add(gain)

	orderFakeID, err := uuid.NewV4()
	if err != nil {
		return "", errors.Annotate(err, "UUID Generation")
	}
	return fmt.Sprintf("FAKE_SELL-%s", orderFakeID), nil
}

// CalculateTradingFees calculates the trading fees for an order on a specified market.
func (wrapper *ExchangeWrapperSimulator) CalculateTradingFees(market *environment.Market, amount float64, limit float64, orderType TradeType) float64 {
	return wrapper.innerWrapper.CalculateTradingFees(market, amount, limit, orderType)
}

// CalculateWithdrawFees calculates the withdrawal fees on a specified market.
func (wrapper *ExchangeWrapperSimulator) CalculateWithdrawFees(market *environment.Market, amount float64) float64 {
	return wrapper.innerWrapper.CalculateWithdrawFees(market, amount)
}

// GetBalance gets the balance of the user of the specified currency.
func (wrapper *ExchangeWrapperSimulator) GetBalance(symbol string) (*decimal.Decimal, error) {
	bal, exists := wrapper.balances[symbol]
	if !exists {
		wrapper.balances[symbol] = decimal.Zero
		var bal = decimal.Zero
		return &bal, nil
	}
	return &bal, nil
}

// GetDepositAddress gets the deposit address for the specified coin on the exchange.
func (wrapper *ExchangeWrapperSimulator) GetDepositAddress(coinTicker string) (string, bool) {
	return "", false
}

// FeedConnect connects to the feed of the exchange.
func (wrapper *ExchangeWrapperSimulator) FeedConnect(markets []*environment.Market) error {
	return wrapper.innerWrapper.FeedConnect(markets)
}

// Withdraw performs a FAKE withdraw operation from the exchange to a destination address.
func (wrapper *ExchangeWrapperSimulator) Withdraw(destinationAddress string, coinTicker string, amount float64) error {
	if amount <= 0 {
		return errors.New("Withdraw amount must be > 0")
	}

	bal, exists := wrapper.balances[coinTicker]
	amt := decimal.NewFromFloat(amount)
	if !exists || amt.GreaterThan(bal) {
		return errors.New("not enough balance")
	}

	wrapper.balances[coinTicker] = bal.Sub(amt)

	return nil
}
