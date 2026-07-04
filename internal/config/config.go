// Package config carrega a configuração do serviço a partir de variáveis de ambiente.
package config

import "os"

// Config reúne os parâmetros de execução do data-transform-service.
type Config struct {
	GRPCPort      string
	MetricsPort   string
	PseudonymSalt string
	LogLevel      string
}

// Load lê as variáveis de ambiente aplicando valores padrão.
func Load() Config {
	return Config{
		GRPCPort:      getenv("GRPC_PORT", "50053"),
		MetricsPort:   getenv("METRICS_PORT", "9091"),
		PseudonymSalt: getenv("PSEUDONYM_SALT", "pspd-troque-este-salt"),
		LogLevel:      getenv("LOG_LEVEL", "info"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
