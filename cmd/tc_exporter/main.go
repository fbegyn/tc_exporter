package main

import (
	"os"

	"github.com/go-kit/kit/log"
	"github.com/spf13/viper"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type Config struct {
	Interfaces []string `mapstructure:"interfaces"`
}

const (
	version = "v0.5.3"
)

func main() {
	var ()
	// CLI arguments parsing
	kingpin.Version(version)
	kingpin.Parse()

	// Start up the logger
	var logger log.Logger
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "version", "v0.5.1", "caller", log.DefaultCaller)

	// Read the data from the config file
	// currently the following options can be used in the configuration folder
	// interfaces: array - array holding the dvice names
	logger.Log("msg", "reading config file ...")
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("/etc/tc_exporter/")
	viper.AddConfigPath("$HOME/.config/tc_exporter/")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		} else {
		}
	}
	var cf Config
	err := viper.Unmarshal(&cf)
	if err != nil {
		logger.Log("level", "ERROR", "msg", "failed to read config file", "error", err)
	}
	logger.Log("msg", "succesfully read config file")
}
