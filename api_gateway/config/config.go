package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Port                      string `mapstructure:"API_GATEWAY_PORT"`
	AuthenticationGRPCAddress string `mapstructure:"AUTH_GRPC_ADDRESS"`
	PropertyGRPCAddress       string `mapstructure:"PROPERTY_GRPC_ADDRESS"`
	MessagingGRPCAddress      string `mapstructure:"MESSAGING_GRPC_ADDRESS"`
	SentryConfig              string `mapstructure:"SENTRY_CONFIG"`
	SentryConst               string `mapstructure:"SENTRY_CONST"`
	TokenSymmetricKey         string `mapstructure:"TOKEN_SYMMETRIC_KEY"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.SetConfigName(".env")

	// Set the path to look for the configuration file
	viper.AddConfigPath(path)

	viper.SetDefault("API_GATEWAY_PORT", "8084")
	viper.SetDefault("AUTH_GRPC_ADDRESS", "localhost:9091")
	viper.SetDefault("PROPERTY_GRPC_ADDRESS", "localhost:9092")
	viper.SetDefault("MESSAGING_GRPC_ADDRESS", "localhost:9093")
	viper.SetDefault("SENTRY_CONFIG", "https://0603a7392023433883071b2243391f9f@o4505462088701952.ingest.sentry.io/4505462093305856")
	viper.SetDefault("SENTRY_CONST", "https://0603a7392023433883071b2243391f9f@o4505462088701952.ingest.sentry.io/4505462093305856")
	viper.SetDefault("TOKEN_SYMMETRIC_KEY", "12345678901234567890123456789012")

	// Automatically read environment variables
	viper.AutomaticEnv()

	// Set the type of the configuration file
	viper.SetConfigType("env")

	// Read the configuration file
	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	// Unmarshal the configuration into the Config struct
	err = viper.Unmarshal(&config)
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
