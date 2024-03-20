<p align="center"><img src="https://lumiere-a.akamaihd.net/v1/images/bb-8-main_72775463.jpeg" width="360"></p>
<p align="center">
  <a href="https://github.com/avelino/awesome-go#other-software"><img src="https://cdn.rawgit.com/sindresorhus/awesome/d7305f38d29fed78fa85652e3a63e154dd8e8829/media/badge.svg" alt="Listed on Awesome Go"></img></a>
  <a href="https://github.com/mcwarner5/BlockBot8000/actions"><img src="https://github.com/mcwarner5/BlockBot8000/workflows/Go%20Build/badge.svg?branch=develop" alt="Develop Branch Build Status"></img></a>
  <a href="https://godoc.org/github.com/mcwarner5/BlockBot8000"><img src="https://godoc.org/github.com/mcwarner5/BlockBot8000?status.svg" alt="Godoc reference"></a>
  <a href="https://github.com/mcwarner5/BlockBot8000/releases"><img src="https://img.shields.io/github/release/saniales/golang-crypto-trading-bot.svg" alt="Last Release"></a>
  <a href="https://github.com/mcwarner5/BlockBot8000/LICENSE"><img src="https://img.shields.io/github/license/saniales/golang-crypto-trading-bot.svg?maxAge=2592000" alt="License"></a>
  <a href="https://goreportcard.com/report/github.com/mcwarner5/BlockBot8000"><img src="https://goreportcard.com/badge/github.com/mcwarner5/BlockBot8000" alt="Goreportcard" /></a>
</p>

# BlockBot8000

A golang implementation of a console-based trading bot for cryptocurrency exchanges.

## Usage

Download a release or directly build the code from this repository.

``` bash
go get github.com/mcwarner5/BlockBot8000
```

If you need to, you can create a strategy and bind it to the bot:

``` go
import bot "github.com/mcwarner5/BlockBot8000/cmd"

func main() {
    bot.Execute()
}
```

For strategy reference see the [Godoc documentation](https://godoc.org/github.com/mcwarner5/BlockBot8000).

## Simulation Mode

If enabled, the bot will do paper trading, as it will execute fake orders in a sandbox environment.

A Fake balance for each coin must be specified for each exchange if simulation mode is enabled.

Get coinbase API Keys/Secrets at: coinbase.com/settings/api

## Supported Exchanges

| Exchange Name | REST Supported    | Websocket Support | API Keys Website                |
| ------------- |------------------ | ----------------- | ------------------------------- |
| Bittrex       | Yes               | No                |                                 |
| Poloniex      | Yes               | Yes               |                                 |
| Kraken        | Yes (recommneded) | No                | pro.kraken.com/app/settings/api |
| Bitfinex      | Yes               | Yes               |                                 |
| Binance       | Yes               | Yes               |                                 |
| Kucoin        | Yes               | No                |                                 |
| HitBtc        | Yes               | Yes               |                                 |

## Configuration file template

Create a configuration file from this example or run the `init` command of the compiled executable.

``` yaml
simulation_configs:
  enabled: true
  start_date: '2023-01-01'
  end_date: '2024-03-15'
  public_key: ''
  secret_key: ''
  interval: 1440
  fake_balances:
    btc: 0.0
    eth: 50.0
    link: 0.0
    avax: 0.0
    sol: 0.0
    dot: 0.0
    ada: 0.0
    matic: 0.0
    algo: 0.0
    atom: 0.0
    usdt: 0.0
exchange_configs:
  - exchange: kraken
    public_key: 
    secret_key: 
    deposit_addresses:
      'BTC': BTC_wallet
      'ETH': ETH_wallet
      'SOL': SOL_wallet
      'DOT': DOT_wallet
      'ADA': ADA_wallet
      'XYZ': XYZ_wallet
      'EOS': EOS_wallet
      'ATOM': ATOM_wallet
      'XLM': XLM_wallet
strategies:
  - strategy: RebalancerStrategy
    spec:
      name: MyFirstRebalancer
      interval: 1440
      allowance_threshold: 0.25
      market_cap_multiplier: 1.25
      min_trade_size: 0.0075
      static_coin: usdt
      nuetral_coin: eth
      portfolio_ratio_percent: 
        eth: 0.3
        dot: 0.125
        link: 0.125
        sol: 0.2
        ada: 0.2
        usdt: 0.05
    markets:
      - market: eth-usdt
        bindings:
        - exchange: kraken
          market_name: ETHUSDT
        - exchange: simulator
          market_name: "ETH-USDC"
      - market: sol-usdt
        bindings:
        - exchange: kraken
          market_name: SOLUSDT
        - exchange: simulator
          market_name: "SOL-USDC"
      - market: link-usdt
        bindings:
        - exchange: kraken
          market_name: LINKUSDT
        - exchange: simulator
          market_name: "LINK-USDC"
      - market:  dot-usdt
        bindings:
        - exchange: kraken
          market_name: DOTUSDT
        - exchange: simulator
          market_name: "DOT-USDC"
      - market: ada-usdt
        bindings:
        - exchange: kraken
          market_name: ADAUSDT
        - exchange: simulator
          market_name: "ADA-USDC"          
      - market: usdt-usd
        bindings:
        - exchange: kraken
          market_name: USDTZUSD
        - exchange: simulator
          market_name: "USDT-USD"
```
