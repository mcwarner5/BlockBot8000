package environment

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type TradeType int16

const (
	MarketPrice TradeType = iota
	LimitOrder  TradeType = iota
)

func (w TradeType) String() string {
	return [...]string{"Market", "Limit"}[w-1]
}

func (w TradeType) EnumIndex() int {
	return int(w)
}

type TradeStatus int16

const (
	Complete TradeStatus = iota
	Pending  TradeStatus = iota
	Canceled TradeStatus = iota
)

func (w TradeStatus) String() string {
	return [...]string{"Complete", "Pending", "Cancled"}[w]
}

func (w TradeStatus) EnumIndex() int {
	return int(w)
}

type TradeSide int16

const (
	Buy  TradeSide = iota
	Sell TradeSide = iota
)

func TradeSideFromString(s string) (TradeSide, error) {
	switch strings.ToUpper(s) {
	case "BUY":
		return Buy, nil
	case "SELL":
		return Sell, nil
	default:
		return -1, errors.New("invalid TradeSide")
	}
}

func (w TradeSide) String() string {
	return [...]string{"Buy", "Sell"}[w]
}

func (w TradeSide) EnumIndex() int {
	return int(w)
}

type TradeBook struct {
	Trades []Trade `json:"trades"`
}

func (book TradeBook) String() string {
	one := decimal.NewFromInt(1)
	var pa_string string = "$$$ TradeBook Summary $$$\n"
	var TotalBuyCount, TotalBuyAmount, TotalSellCount, TotalSellAmount,
		TotalTrades, TotalFessAmount decimal.Decimal

	for _, trade := range book.Trades {
		TotalTrades = TotalTrades.Add(one)
		TotalFessAmount = TotalFessAmount.Add(trade.Fees)

		if trade.Side == Buy {
			TotalBuyCount = TotalBuyCount.Add(one)
			TotalBuyAmount = TotalBuyAmount.Add(trade.Total())

		} else if trade.Side == Sell {
			TotalSellCount = TotalSellCount.Add(one)
			TotalSellAmount = TotalSellAmount.Add(trade.Total())
		}
	}
	pa_string += fmt.Sprintln("TotalTrades: " + TotalTrades.String())
	pa_string += fmt.Sprintln("TotalBuyCount: " + TotalBuyCount.String())
	pa_string += fmt.Sprintln("TotalSellCount: " + TotalSellCount.String())
	pa_string += fmt.Sprintln("TotalBuyAmount: $" + TotalBuyAmount.Round(2).String())
	pa_string += fmt.Sprintln("TotalSellAmount: $" + TotalSellAmount.Round(2).String())
	pa_string += fmt.Sprintln("TotalFeesAmount: $" + TotalFessAmount.Round(2).String())

	return pa_string
}

func NewTradeBook() *TradeBook {
	return &TradeBook{
		Trades: make([]Trade, 0),
	}
}

func NewSizedTradeBook(size int) *TradeBook {
	return &TradeBook{
		Trades: make([]Trade, size),
	}
}

// Order represents a single order in the Order Book for a market.
type Trade struct {
	Price        decimal.Decimal //Value of the trade : e.g. in a BTC ETH is the value of a single ETH in BTC.
	AskQuantity  decimal.Decimal
	FillQuantity decimal.Decimal //Quantity of Coins of this order.
	Fees         decimal.Decimal
	Market       string
	Side         TradeSide
	Status       TradeStatus
	Type         TradeType
	TradeNumber  string    //[optional] Order number as seen in echange archives.
	Timestamp    time.Time //[optional] The timestamp of the order (as got from the exchange).
}

func (trade Trade) String() string {
	tr_str := "#" + trade.TradeNumber + " "
	tr_str += trade.Market + "@ " + trade.Price.String()
	tr_str += " for " + trade.FillQuantity.String() + " coins\n"

	return tr_str
}

// Total returns order total in base currency.
func (trade Trade) Total() decimal.Decimal {
	return trade.FillQuantity.Mul(trade.Price)
}
