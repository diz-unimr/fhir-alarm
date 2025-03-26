package config

import (
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"strings"
)

type AppConfig struct {
	App          App          `mapstructure:"app"`
	Fhir         Fhir         `mapstructure:"fhir"`
	Notification Notification `mapstructure:"notification"`
}

type App struct {
	Name     string `mapstructure:"name"`
	LogLevel string `mapstructure:"log-level"`
}

type Fhir struct {
	Server Server `mapstructure:"server"`
}

type Email struct {
	Recipients string `mapstructure:"recipients"`
	Sender     string `mapstructure:"sender"`
	Smtp       Smtp   `mapstructure:"smtp"`
}

type Smtp struct {
	Server   string `mapstructure:"server"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

type Notification struct {
	Email Email `mapstructure:"email"`
}

type Server struct {
	Host           string `mapstructure:"host"`
	Auth           *Auth  `mapstructure:"auth"`
	SubscriptionId string `mapstructure:"subscription-id"`
}

type Auth struct {
	CertLocation string `mapstructure:"cert-location"`
	KeyLocation  string `mapstructure:"key-location"`
}

func LoadConfig() AppConfig {
	c, err := parseConfig(".")
	if err != nil {
		slog.Error("Unable to load config file", "error", err)
		os.Exit(1)
	}

	return *c
}

func parseConfig(path string) (config *AppConfig, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`, `-`, `_`))

	err = viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	parseSecretFiles()

	err = viper.Unmarshal(&config)
	return config, err
}

func ConfigureLogger(c AppConfig) {
	lvl := new(slog.LevelVar)
	lvl.Set(slog.LevelInfo)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	}))
	slog.SetDefault(logger)

	// set configured log level
	err := lvl.UnmarshalText([]byte(c.App.LogLevel))
	if err != nil {
		slog.Error("Unable to set Log level from application properties", "level", c.App.LogLevel, "error", err)
	}
}

func parseSecretFiles() {
	keys := viper.AllKeys()
	for _, key := range keys {

		value := getSecretFromFile(key)
		if value != nil {
			viper.Set(strings.TrimSuffix(key, ".file"), value)
		}
	}
}

func getSecretFromFile(configKey string) *string {
	if !strings.HasSuffix(configKey, ".file") {
		configKey += ".file"
	}
	var valuePath = viper.GetString(configKey)
	if valuePath == "" {
		return nil
	}

	contents, err := os.ReadFile(valuePath)
	if err == nil {
		value := strings.Trim(string(contents), "\n")
		return &value
	}

	slog.Error("Failed to read secrets file", "path", valuePath, "configKey", configKey, "error", err)
	return nil
}
