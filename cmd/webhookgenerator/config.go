package main

type config struct {
	BaseURL     string `mapstructure:"base_url"`
	Token       string `mapstructure:"token"`
	Secret      string `mapstructure:"secret"`
	Service     string `mapstructure:"service"`
	Environment string `mapstructure:"environment"`
	Interval    string `mapstructure:"interval"`
}
