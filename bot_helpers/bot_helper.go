package helpers

import (
	"github.com/mcwarner5/BlockBot8000/environment"
	"github.com/mcwarner5/BlockBot8000/exchanges"
	"github.com/mcwarner5/BlockBot8000/intervalstrategies"
	"github.com/mcwarner5/BlockBot8000/strategies"
)

// InitExchange initialize a new ExchangeWrapper binded to the specified exchange provided.
func InitExchange(exchangeConfig environment.ExchangeConfig, simulatedConfigs environment.SimulationConfig, depositAddresses map[string]string) exchanges.ExchangeWrapper {
	if depositAddresses == nil && !simulatedConfigs.SimModeOn {
		return nil
	}

	var exch exchanges.ExchangeWrapper
	switch exchangeConfig.ExchangeName {
	case "kucoin":
		exch = exchanges.NewKucoinWrapper(exchangeConfig.PublicKey, exchangeConfig.SecretKey, depositAddresses)
	case "kraken":
		exch = exchanges.NewKrakenWrapper(exchangeConfig.PublicKey, exchangeConfig.SecretKey, depositAddresses)
	case "coinbase":
		exch = exchanges.NewCoinbaseWrapper(exchangeConfig.PublicKey, exchangeConfig.SecretKey, depositAddresses)
	default:
		return nil
	}

	if simulatedConfigs.SimModeOn {
		if simulatedConfigs.SimFakeBalances == nil {
			return nil
		}
		exch = exchanges.NewExchangeWrapperSimulator(exch, simulatedConfigs)
	}

	return exch
}

func InitStrategy(rawStrategy environment.StrategyConfig) strategies.Strategy {
	switch rawStrategy.Strategy {
	case "PullMarketData":
		return intervalstrategies.NewPullMarketData(rawStrategy)
	case "RebalancerStrategy":
		return intervalstrategies.NewRebalancerStrategy(rawStrategy)
	default:
		return nil
	}
}
