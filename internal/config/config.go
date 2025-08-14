package config

import (
	"os"
)

type Config struct {
	ProcessorDefaultURL  string
	ProcessorFallbackURL string
	SocketPath           string
	OtherSocketPath      string
	SummaryUrl           string
}

func Load() Config {
	return Config{
		ProcessorDefaultURL:  getenv("PROCESSOR_DEFAULT_URL", "http://payment-processor-default:8080"),
		ProcessorFallbackURL: getenv("PROCESSOR_FALLBACK_URL", "http://payment-processor-fallback:8080"),
		SocketPath:           getenv("SOCKET_PATH", ""),
		OtherSocketPath:      getenv("OTHER_SOCKET_PATH", ""),
		SummaryUrl:           getenv("SUMMARY_URL", ""),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
