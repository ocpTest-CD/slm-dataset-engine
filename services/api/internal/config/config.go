package config

import "os"

type Config struct {
	ListenAddr    string
	DatabaseURL   string
	ArtifactsDir  string
	MigrationsDir string
}

func Load() Config {
	return Config{
		ListenAddr:    env("LISTEN_ADDR", ":8080"),
		DatabaseURL:   env("DATABASE_URL", "postgres://slm_dataset_engine:slm_dataset_engine@localhost:5432/slm_dataset_engine?sslmode=disable"),
		ArtifactsDir:  env("ARTIFACTS_DIR", "./artifacts"),
		MigrationsDir: env("MIGRATIONS_DIR", "../../migrations"),
	}
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
