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

package bot

import (
	"reflect"
	"strings"

	helpers "github.com/mcwarner5/BlockBot8000/bot_helpers"
	"github.com/mcwarner5/BlockBot8000/environment"
	"github.com/mcwarner5/BlockBot8000/exchanges"
	"github.com/mcwarner5/BlockBot8000/strategies"
	"github.com/mitchellh/mapstructure"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	viper "github.com/spf13/viper"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts trading using saved configs",
	Long:  `Starts trading using saved configs`,
	Run:   executeStartCommand,
}

var botConfig environment.BotConfig

func init() {
	RootCmd.AddCommand(startCmd)
	startCmd.Flags().BoolVarP(&startFlags.Simulate, "simulate", "s", false, "Simulates the trades instead of actually doing them")
}

func DecimalHookFunction(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t != reflect.TypeOf(decimal.Decimal{}) {
		return data, nil
	}

	switch f.Kind() {
	case reflect.Int:
		return decimal.NewFromInt(int64(data.(int))), nil
	case reflect.Int64:
		return decimal.NewFromInt(data.(int64)), nil
	case reflect.Float64:
		return decimal.NewFromFloat(data.(float64)), nil
	case reflect.String:
		return decimal.NewFromString(data.(string))
	default:
		return data, nil
	}

}

func initConfigs() error {

	hooks := mapstructure.ComposeDecodeHookFunc(
		DecimalHookFunction,
	)

	viper.SetConfigType("yaml")
	viper.SetConfigFile(GlobalFlags.ConfigFile)
	viper.AutomaticEnv()
	err := viper.ReadInConfig()

	if err != nil {
		return err
	}

	err = viper.Unmarshal(&botConfig, viper.DecodeHook(hooks))

	if err != nil {
		return err
	}

	logrus.Info("DONE")

	return nil
}

func executeStartCommand(cmd *cobra.Command, args []string) {
	logrus.Info("Getting configurations ... ")
	if err := initConfigs(); err != nil {
		logrus.Info("Cannot read from configuration file, please create or replace the current one using gobot init")
		return
	}
	logrus.Info("DONE")

	logrus.Info("Getting exchange info ... ")
	wrappers := make([]exchanges.ExchangeWrapper, len(botConfig.ExchangeConfigs))
	for i, config := range botConfig.ExchangeConfigs {
		wrappers[i] = helpers.InitExchange(config, botConfig.SimulationConfigs, config.DepositAddresses)
	}
	logrus.Info("DONE")

	logrus.Info("Getting markets cold info ... ")
	for _, strategyConf := range botConfig.Strategies {
		mkts := make([]*environment.Market, len(strategyConf.Markets))
		for i, mkt := range strategyConf.Markets {
			currencies := strings.SplitN(mkt.Name, "-", 2)
			mkts[i] = &environment.Market{
				Name:           mkt.Name,
				BaseCurrency:   currencies[0],
				MarketCurrency: currencies[1],
			}

			mkts[i].ExchangeNames = make(map[string]string, len(wrappers))

			for _, exName := range mkt.Exchanges {
				mkts[i].ExchangeNames[exName.Name] = exName.MarketName
			}
		}

		err := strategies.MatchWithMarkets(strategies.AddCustomStrategy(helpers.InitStrategy(strategyConf)), mkts)
		if err != nil {
			logrus.Info("Cannot add tactic : ", err)
		}
	}
	logrus.Info("DONE")

	logrus.Info("Starting bot ... ")
	executeBotLoop(wrappers)
	logrus.Info("EXIT, good bye :)")
}

func executeBotLoop(wrappers []exchanges.ExchangeWrapper) {
	strategies.ApplyAllStrategies(wrappers)
}
