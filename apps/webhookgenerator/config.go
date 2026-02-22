package main

type config struct {
	BaseURL      string   `mapstructure:"base_url"`
	Token        string   `mapstructure:"token"`
	Secret       string   `mapstructure:"secret"`
	Service      string   `mapstructure:"service"`
	Services     []string `mapstructure:"services"`
	Environment  string   `mapstructure:"environment"`
	Environments []string `mapstructure:"environments"`
	Interval     string   `mapstructure:"interval"`
}
