package exchanges

import (
	"fmt"
	"sync"
	"time"

	"github.com/mcwarner5/BlockBot8000/environment"
)

// SummaryCache represents a local summary cache for every exchange. To allow dinamic polling from multiple sources (REST + Websocket)
type SummaryCache struct {
	mutex    *sync.RWMutex
	internal map[*environment.Market]*environment.MarketSummary
}

// NewSummaryCache creates a new SummaryCache Object
func NewSummaryCache() *SummaryCache {
	return &SummaryCache{
		mutex:    &sync.RWMutex{},
		internal: make(map[*environment.Market]*environment.MarketSummary),
	}
}

// Set sets a value for the specified key.
func (sc *SummaryCache) Set(market *environment.Market, summary *environment.MarketSummary) *environment.MarketSummary {
	sc.mutex.Lock()
	old := sc.internal[market]
	sc.internal[market] = summary
	sc.mutex.Unlock()
	return old
}

// Get gets the value for the specified key.
func (sc *SummaryCache) Get(market *environment.Market) (*environment.MarketSummary, bool) {
	sc.mutex.RLock()
	ret, isSet := sc.internal[market]
	sc.mutex.RUnlock()
	return ret, isSet
}

// CandlesCache represents a local candles cache for every exchange. To allow dinamic polling from multiple sources (REST + Websocket)
type CandlesCache struct {
	mutex    *sync.RWMutex
	internal map[*environment.Market][]environment.CandleStick
}

// NewCandlesCache creates a new CandlesCache Object
func NewCandlesCache() *CandlesCache {
	return &CandlesCache{
		mutex:    &sync.RWMutex{},
		internal: make(map[*environment.Market][]environment.CandleStick),
	}
}

// Set sets a value for the specified key.
func (cc *CandlesCache) Set(market *environment.Market, candles []environment.CandleStick) []environment.CandleStick {
	cc.mutex.Lock()
	old := cc.internal[market]
	cc.internal[market] = candles
	cc.mutex.Unlock()
	return old
}

// Get gets the value for the specified key.
func (cc *CandlesCache) Get(market *environment.Market) ([]environment.CandleStick, bool) {
	cc.mutex.RLock()
	ret, isSet := cc.internal[market]
	cc.mutex.RUnlock()
	return ret, isSet
}

// OrderbookCache represents a local orderbook cache for every exchange. To allow dinamic polling from multiple sources (REST + Websocket)
type OrderbookCache struct {
	mutex    *sync.RWMutex
	internal map[*environment.Market]*environment.OrderBook
}

// NewOrderbookCache creates a new OrderbookCache Object
func NewOrderbookCache() *OrderbookCache {
	return &OrderbookCache{
		mutex:    &sync.RWMutex{},
		internal: make(map[*environment.Market]*environment.OrderBook),
	}
}

// Set sets a value for the specified key.
func (cc *OrderbookCache) Set(market *environment.Market, book *environment.OrderBook) *environment.OrderBook {
	cc.mutex.Lock()
	old := cc.internal[market]
	cc.internal[market] = book
	cc.mutex.Unlock()
	return old
}

// Get gets the value for the specified key.
func (cc *OrderbookCache) Get(market *environment.Market) (*environment.OrderBook, bool) {
	cc.mutex.RLock()
	ret, isSet := cc.internal[market]
	cc.mutex.RUnlock()
	return ret, isSet
}

type CandleMap struct {
	TimeMap map[string]*environment.CandleStick
}

func NewSizedCandleMap(size int) CandleMap {
	return CandleMap{TimeMap: make(map[string]*environment.CandleStick, size)}
}

type MappedCandlesCache struct {
	mutex    *sync.RWMutex
	internal map[*environment.Market]CandleMap
}

// NewCandlesCache creates a new CandlesCache Object
func NewMappedCandlesCache() *MappedCandlesCache {
	return &MappedCandlesCache{
		mutex:    &sync.RWMutex{},
		internal: make(map[*environment.Market]CandleMap),
	}
}

// Set sets a value for the specified key.
func (mcc *MappedCandlesCache) SetMap(market *environment.Market, candles CandleMap) CandleMap {
	mcc.mutex.Lock()
	old := mcc.internal[market]
	mcc.internal[market] = candles
	mcc.mutex.Unlock()
	return old
}

// Get gets the value for the specified key.
func (mcc *MappedCandlesCache) GetMap(market *environment.Market) (CandleMap, bool) {
	mcc.mutex.RLock()
	ret, isSet := mcc.internal[market]
	mcc.mutex.RUnlock()
	return ret, isSet
}

// Get gets the value for the specified key.
func (mcc *MappedCandlesCache) GetTime(market *environment.Market, time time.Time) (*environment.CandleStick, bool) {
	mcc.mutex.RLock()
	cc, isSet := mcc.internal[market]
	if !isSet {
		mcc.mutex.RUnlock()
		return nil, isSet
	}
	time_key := fmt.Sprint(time.Unix())

	ret, isSet := cc.TimeMap[time_key]
	if !isSet {
		mcc.mutex.RUnlock()
		return nil, isSet
	}

	mcc.mutex.RUnlock()
	return ret, isSet
}

type OrderBookMap struct {
	TimeMap map[string]*environment.OrderBook
}

func NewOrderBookMap(size int) OrderBookMap {
	return OrderBookMap{TimeMap: make(map[string]*environment.OrderBook, size)}
}

type MappedOrdersCache struct {
	mutex    *sync.RWMutex
	internal map[*environment.Market]OrderBookMap
}

// NewCandlesCache creates a new CandlesCache Object
func NewMappedOrdersCache() *MappedOrdersCache {
	return &MappedOrdersCache{
		mutex:    &sync.RWMutex{},
		internal: make(map[*environment.Market]OrderBookMap),
	}
}

func (moc *MappedOrdersCache) SetMap(market *environment.Market, orders OrderBookMap) OrderBookMap {
	moc.mutex.Lock()
	old := moc.internal[market]
	moc.internal[market] = orders
	moc.mutex.Unlock()
	return old
}

// Get gets the value for the specified key.
func (moc *MappedOrdersCache) GetMap(market *environment.Market) (OrderBookMap, bool) {
	moc.mutex.RLock()
	ret, isSet := moc.internal[market]
	moc.mutex.RUnlock()
	return ret, isSet
}

// Get gets the value for the specified key.
func (moc *MappedOrdersCache) GetTime(market *environment.Market, time time.Time) (*environment.OrderBook, bool) {
	moc.mutex.RLock()
	cc, isSet := moc.internal[market]
	if !isSet {
		moc.mutex.RUnlock()
		return nil, isSet
	}
	time_key := fmt.Sprint(time.Unix())

	ret, isSet := cc.TimeMap[time_key]
	if !isSet {
		moc.mutex.RUnlock()
		return nil, isSet
	}

	moc.mutex.RUnlock()
	return ret, isSet
}
