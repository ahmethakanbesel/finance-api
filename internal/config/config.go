package config

import "os"

type Config struct {
	Port    string
	DBPath  string
	Workers int
}

func Load() Config {
	return Config{
		Port:    getEnv("PORT", "8080"),
		DBPath:  getEnv("DB_PATH", "finance.db"),
		Workers: getEnvInt("WORKERS", 5),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n := 0
	for _, c := range v {
		if c < '0' || c > '9' {
			return fallback
		}
		n = n*10 + int(c-'0')
	}
	return n
}
