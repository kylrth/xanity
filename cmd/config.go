package main

import (
	"strings"

	"github.com/spf13/viper"
)

func setConfig() {
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	viper.SetDefault("secretkey", "devkey")

	viper.SetDefault("poll.enabled", true)
	viper.SetDefault("poll.cron", "*/30 * * * *")
	viper.SetDefault("poll.num", 2000)
	viper.SetDefault("poll.start", 0)
	viper.SetDefault("poll.break", 3)
	viper.SetDefault(
		"poll.query", "cat:cs.CV+OR+cat:cs.LG+OR+cat:cs.CL+OR+cat:cs.AI+OR+cat:cs.NE+OR+cat:cs.RO")

	viper.SetDefault("compute.features", 20000)
	viper.SetDefault("compute.min_df", 5)
	viper.SetDefault("compute.max_df", 0.1)
	viper.SetDefault("compute.max_docs", -1)

	viper.SetDefault("mail.enabled", false)
	viper.SetDefault("mail.cron", "0 5 * * *")

	viper.AutomaticEnv()
}
