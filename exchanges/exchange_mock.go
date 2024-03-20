package exchanges

import (
	"context"
	"fmt"
	"sort"
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
	trades       *TradeBookbookCache
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
		trades:       NewTradeBookbookCache(),
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

func (wrapper *ExchangeWrapperSimulator) GetCurrDate() time.Time {
	return *wrapper.currDate
}

func (wrapper *ExchangeWrapperSimulator) IncrementCurrDate() error {
	var interval_len = time.Duration(wrapper.interval) * time.Minute
	*wrapper.currDate = wrapper.currDate.Add(interval_len)

	if wrapper.currDate.After(wrapper.endDate) {
		*wrapper.currDate = wrapper.currDate.Add(-interval_len)

		diff_duration := wrapper.currDate.Sub(*wrapper.startDate)
		iterations := decimal.NewFromFloat(diff_duration.Minutes()).DivRound(decimal.NewFromInt(int64(wrapper.interval)), 2)
		diff_days := decimal.NewFromFloat(diff_duration.Hours()).DivRound(decimal.NewFromInt(24), 3)

		end_str := fmt.Sprintln("End of Simulation")
		end_str += "Simulation Start Date:" + wrapper.startDate.String() + "\n"
		end_str += "Simulation End Date:" + wrapper.currDate.String() + "\n"
		end_str += "Simulation Iterations:" + iterations.String() + "\n"
		end_str += "Simulation Days:" + diff_days.String() + "\n"
		logrus.Info(end_str)
		return errors.New("End of Simulation Date has been reached")
	}

	return nil
}

// GetCandles gets the candle data from the exchange.
func (wrapper *ExchangeWrapperSimulator) UpdateMappedCandles(market *environment.Market, from_time time.Time) (*environment.CandleStick, error) {
	ctx := context.Background()
	one_interval := time.Duration(wrapper.interval) * time.Minute
	api_call_len := time.Duration(wrapper.interval*300) * time.Minute
	api_end_date := from_time.Add(-one_interval + api_call_len)
	api_end_date_key := fmt.Sprint(api_end_date.Unix())
	api_start_date := from_time.Add(-one_interval)
	api_start_date_key := fmt.Sprint(api_start_date.Unix())
	//from_time_key := fmt.Sprint(from_time.Unix())

	var params = client.ListProductsCandlesParams{
		Product:   MarketNameFor(market, wrapper),
		StartTime: api_start_date_key,
		EndTime:   api_end_date_key,
		Interval:  wrapper.interval,
	}
	response, _ := wrapper.coinbase.GetProductCandles(ctx, &params)
	new_map := NewSizedCandleMap(len(response.CandleSticks))
	candles := response.GetCandleSticks()

	sort.Slice(candles, func(i, j int) bool {
		m_t_u_str, _ := strconv.ParseInt(*candles[i].Start, 10, 64)
		o_t_u_str, _ := strconv.ParseInt(*candles[j].Start, 10, 64)
		m_t := time.Unix(m_t_u_str, 0).UTC()
		o_t := time.Unix(o_t_u_str, 0).UTC()
		return m_t.Before(o_t)
	})

	var prev_candle *environment.CandleStick
	pot_prev_candle, isSet := wrapper.candles.GetTime(market, from_time.Add(-one_interval))
	if isSet {
		prev_candle = pot_prev_candle
	}

	next_fill_time := from_time
	for i := 0; i < len(candles); i++ {
		candle := candles[i]
		c_time_u_num, _ := strconv.ParseInt(*candle.Start, 10, 64)
		c_time := time.Unix(c_time_u_num, 0).UTC()
		c_time_key := fmt.Sprint(c_time.Unix())
		c_high, _ := decimal.NewFromString(*candle.High)
		c_open, _ := decimal.NewFromString(*candle.Open)
		c_low, _ := decimal.NewFromString(*candle.Low)
		c_close, _ := decimal.NewFromString(*candle.Close)
		c_volume, _ := decimal.NewFromString(*candle.Volume)

		new_candle := environment.CandleStick{
			High:       c_high,
			Open:       c_open,
			Close:      c_close,
			Low:        c_low,
			Volume:     c_volume,
			CandleTime: c_time,
		}

		if c_time.Equal(api_start_date) {
			prev_candle = &new_candle
			continue
		}

		if c_time.Equal(next_fill_time) {
			next_fill_time = next_fill_time.Add(one_interval)
			new_map.TimeMap[c_time_key] = &new_candle
			prev_candle = &new_candle
			continue
		}

		if c_time.Before(next_fill_time) {
			new_map.TimeMap[c_time_key] = &new_candle
			prev_candle = &new_candle
			continue
		}

		for c_time.After(next_fill_time) {
			if prev_candle != nil {
				copy_candle := *prev_candle
				next_fill_time_str := fmt.Sprint(next_fill_time.Unix())
				new_map.TimeMap[next_fill_time_str] = &copy_candle
				next_fill_time = next_fill_time.Add(one_interval)
			}
		}
	}

	wrapper.candles.SetMap(market, new_map)
	candle, isSet := wrapper.candles.GetTime(market, from_time)
	if !isSet {
		return nil, errors.New("no data for that time set panic")
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
			Offset: int32(CoinbaseMaxLimit) * (i - 1),
			//ProductType: "SPOT",
		}

		res_products, err := wrapper.coinbase.ListProducts(ctx, &params)
		if err != nil {
			return
		}

		var products string
		for _, product := range res_products.Products {
			products = products + fmt.Sprintf("%s \t", *product.ProductId)
		}

		logrus.Info(products)
		var sleep_len = time.Duration(1) * time.Second
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
	var thirty_min = time.Duration(60) * time.Minute

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
	baseBalance, _ := wrapper.GetBalance(market.BaseCurrency)
	quoteBalance, _ := wrapper.GetBalance(market.MarketCurrency)

	orderbook, err := wrapper.GetOrderBook(market)
	if err != nil {
		return "", errors.Annotate(err, "Cannot market buy without orderbook knowledge")
	}

	totalQuote := decimal.Zero
	remainingAmount := decimal.NewFromFloat(amount)
	expense := decimal.Zero
	avg_price := decimal.Zero

	for _, ask := range orderbook.Asks {
		if ask.Value.LessThan(decimal.NewFromFloat(limit)) {
			continue
		}

		if remainingAmount.LessThanOrEqual(ask.Quantity) {
			totalQuote = totalQuote.Add(remainingAmount)
			expense = expense.Add(remainingAmount.Mul(ask.Value))
			avg_price = ask.Value
			if expense.GreaterThan(*quoteBalance) {
				return "", fmt.Errorf("cannot Buy not enough %s balance", market.BaseCurrency)
			}
			break
		}

		old_totalQuote := totalQuote
		new_totalQuote := totalQuote.Add(ask.Quantity)
		avg_price = (old_totalQuote.Mul(avg_price).Add((ask.Quantity.Mul(ask.Value)))).Div(new_totalQuote)

		totalQuote = new_totalQuote
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

	new_trade := environment.Trade{
		Price:        avg_price,
		AskQuantity:  decimal.NewFromFloat(amount),
		FillQuantity: totalQuote,
		Fees:         decimal.Zero,
		Market:       market.Name,
		Side:         environment.Buy,
		Status:       environment.Complete,
		Type:         environment.LimitOrder,
		TradeNumber:  orderFakeID.String(),
		Timestamp:    time.Now(),
	}
	wrapper.AddTrade(market, new_trade)

	return fmt.Sprintf("FAKE_BUY-%s", new_trade.String()), nil
}

// SellLimit here is just to implement the ExchangeWrapper Interface, do not use, use SellMarket instead.
func (wrapper *ExchangeWrapperSimulator) SellLimit(market *environment.Market, amount float64, limit float64) (string, error) {
	baseBalance, _ := wrapper.GetBalance(market.BaseCurrency)
	quoteBalance, _ := wrapper.GetBalance(market.MarketCurrency)

	orderbook, err := wrapper.GetOrderBook(market)
	if err != nil {
		return "", errors.Annotate(err, "cannot market sell without orderbook knowledge")
	}

	totalQuote := decimal.Zero
	remainingAmount := decimal.NewFromFloat(amount)
	gain := decimal.Zero
	avg_price := decimal.Zero

	if baseBalance.LessThan(remainingAmount) {
		return "", fmt.Errorf("cannot Sell: not enough %s balance", market.MarketCurrency)
	}

	for _, bid := range orderbook.Bids {
		if bid.Value.GreaterThan(decimal.NewFromFloat(limit)) {
			continue
		}

		if remainingAmount.LessThanOrEqual(bid.Quantity) {
			totalQuote = totalQuote.Add(remainingAmount)
			gain = gain.Add(remainingAmount.Mul(bid.Value))
			avg_price = bid.Value
			break
		}

		old_totalQuote := totalQuote
		new_totalQuote := totalQuote.Add(bid.Quantity)
		avg_price = (old_totalQuote.Mul(avg_price).Add((bid.Quantity.Mul(bid.Value)))).Div(new_totalQuote)

		totalQuote = new_totalQuote
		remainingAmount = remainingAmount.Sub(bid.Quantity)
		gain = gain.Add(bid.Quantity.Mul(bid.Value))
	}

	wrapper.balances[market.BaseCurrency] = baseBalance.Sub(totalQuote)
	wrapper.balances[market.MarketCurrency] = quoteBalance.Add(gain)

	orderFakeID, err := uuid.NewV4()
	if err != nil {
		return "", errors.Annotate(err, "UUID Generation")
	}

	new_trade := environment.Trade{
		Price:        avg_price,
		AskQuantity:  decimal.NewFromFloat(amount),
		FillQuantity: totalQuote,
		Fees:         decimal.Zero,
		Market:       market.Name,
		Side:         environment.Sell,
		Status:       environment.Complete,
		Type:         environment.LimitOrder,
		TradeNumber:  orderFakeID.String(),
		Timestamp:    time.Now(),
	}
	wrapper.AddTrade(market, new_trade)

	return fmt.Sprintf("FAKE_SELL-%s", new_trade.String()), nil
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
	avg_price := decimal.Zero

	for _, ask := range orderbook.Asks {
		if remainingAmount.LessThanOrEqual(ask.Quantity) {
			totalQuote = totalQuote.Add(remainingAmount)
			expense = expense.Add(remainingAmount.Mul(ask.Value))
			avg_price = ask.Value
			if expense.GreaterThan(*quoteBalance) {
				return "", fmt.Errorf("cannot Buy not enough %s balance", market.BaseCurrency)
			}
			break
		}

		old_totalQuote := totalQuote
		new_totalQuote := totalQuote.Add(ask.Quantity)
		avg_price = (old_totalQuote.Mul(avg_price).Add((ask.Quantity.Mul(ask.Value)))).Div(new_totalQuote)

		totalQuote = new_totalQuote
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
	new_trade := environment.Trade{
		Price:        avg_price,
		AskQuantity:  decimal.NewFromFloat(amount),
		FillQuantity: totalQuote,
		Fees:         decimal.Zero,
		Market:       market.Name,
		Side:         environment.Buy,
		Status:       environment.Complete,
		Type:         environment.MarketPrice,
		TradeNumber:  orderFakeID.String(),
		Timestamp:    time.Now(),
	}
	wrapper.AddTrade(market, new_trade)

	return fmt.Sprintf("FAKE_BUY-%s", new_trade.String()), nil
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
	avg_price := decimal.Zero

	if baseBalance.LessThan(remainingAmount) {
		return "", fmt.Errorf("cannot Sell: not enough %s balance", market.MarketCurrency)
	}

	for _, bid := range orderbook.Bids {
		if remainingAmount.LessThanOrEqual(bid.Quantity) {
			totalQuote = totalQuote.Add(remainingAmount)
			gain = gain.Add(remainingAmount.Mul(bid.Value))
			avg_price = bid.Value
			break
		}

		old_totalQuote := totalQuote
		new_totalQuote := totalQuote.Add(bid.Quantity)
		avg_price = (old_totalQuote.Mul(avg_price).Add((bid.Quantity.Mul(bid.Value)))).Div(new_totalQuote)

		totalQuote = new_totalQuote
		remainingAmount = remainingAmount.Sub(bid.Quantity)
		gain = gain.Add(bid.Quantity.Mul(bid.Value))
	}

	wrapper.balances[market.BaseCurrency] = baseBalance.Sub(totalQuote)
	wrapper.balances[market.MarketCurrency] = quoteBalance.Add(gain)

	orderFakeID, err := uuid.NewV4()
	if err != nil {
		return "", errors.Annotate(err, "UUID Generation")
	}

	new_trade := environment.Trade{
		Price:        avg_price,
		AskQuantity:  decimal.NewFromFloat(amount),
		FillQuantity: totalQuote,
		Fees:         decimal.Zero,
		Market:       market.Name,
		Side:         environment.Sell,
		Status:       environment.Complete,
		Type:         environment.MarketPrice,
		TradeNumber:  orderFakeID.String(),
		Timestamp:    time.Now(),
	}
	wrapper.AddTrade(market, new_trade)

	return fmt.Sprintf("FAKE_SELL-%s", new_trade.String()), nil
}

func (wrapper *ExchangeWrapperSimulator) AddTrade(market *environment.Market, trade environment.Trade) error {
	tradeBook, isSet := wrapper.trades.Get(market)
	if !isSet {
		wrapper.trades.Set(market, &environment.TradeBook{Trades: []environment.Trade{trade}})
	} else {
		wrapper.trades.Set(market, &environment.TradeBook{Trades: append(tradeBook.Trades, trade)})
	}

	return nil
}

func (wrapper *ExchangeWrapperSimulator) UpdateTrades(market *environment.Market, from_time time.Time) (*environment.TradeBook, error) {
	return nil, errors.New("error: UpdateTrades not defined for Simulator")
}

func (wrapper *ExchangeWrapperSimulator) GetAllTrades(markets []*environment.Market) (*environment.TradeBook, error) {
	all_trades := environment.TradeBook{}
	for _, market := range markets {
		new_tradeBook, isSet := wrapper.trades.Get(market)
		if !isSet {
			continue
		}

		all_trades.Trades = append(all_trades.Trades, new_tradeBook.Trades...)
	}

	return &all_trades, nil
}
func (wrapper *ExchangeWrapperSimulator) GetAllMarketTrades(market *environment.Market) (*environment.TradeBook, error) {
	tradeBook, isSet := wrapper.trades.Get(market)
	if !isSet {
		return nil, errors.New("Could not find trades for market " + market.Name)
	}
	return tradeBook, nil
}

func (wrapper *ExchangeWrapperSimulator) GetFilteredTrades(market *environment.Market, symbol string, tradeSide environment.TradeSide, tradeType environment.TradeType, tradeStatus environment.TradeStatus) (*environment.TradeBook, error) {
	tradeBook, isSet := wrapper.trades.Get(market)
	finalTradeBook := environment.NewTradeBook()
	if !isSet {
		var err error
		tradeBook, err = wrapper.UpdateTrades(market, *wrapper.currDate)
		if err != nil {
			return nil, err
		}
	}

	for _, trade := range tradeBook.Trades {
		if trade.Market == symbol && trade.Side == tradeSide && trade.Type == tradeType && trade.Status == tradeStatus {
			finalTradeBook.Trades = append(finalTradeBook.Trades, trade)
		}
	}

	return finalTradeBook, nil
}

// CalculateTradingFees calculates the trading fees for an order on a specified market.
func (wrapper *ExchangeWrapperSimulator) CalculateTradingFees(market *environment.Market, amount float64, limit float64, orderSide environment.TradeSide) float64 {
	return wrapper.innerWrapper.CalculateTradingFees(market, amount, limit, orderSide)
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
