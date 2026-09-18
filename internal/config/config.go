package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort string
	AppEnv   string

	DatabaseURL string

	TestSourceURL      string
	TestSourcePageSize int
	TestSourceTimeout  time.Duration
	TestSourceCacheTTL time.Duration

	AppScriptWebAppURL string
	AppScriptSecret    string
	AppScriptTimeout   time.Duration

	PublicWebhookURL string
	WebhookSecret    string
}

func Load() (*Config, error) {
	loadDotenv()

	cfg := &Config{
		HTTPPort:           env("HTTP_PORT", "8080"),
		AppEnv:             env("APP_ENV", "dev"),
		DatabaseURL:        env("DATABASE_URL", ""),
		TestSourceURL:      env("TEST_SOURCE_URL", "http://localhost:8081/tests"),
		AppScriptWebAppURL: env("APPSCRIPT_WEBAPP_URL", ""),
		AppScriptSecret:    env("APPSCRIPT_SECRET", ""),
		PublicWebhookURL:   env("PUBLIC_WEBHOOK_URL", ""),
	}

	var err error
	if cfg.TestSourcePageSize, err = envInt("TEST_SOURCE_PAGE_SIZE", 50); err != nil {
		return nil, err
	}
	if cfg.TestSourceTimeout, err = envDuration("TEST_SOURCE_TIMEOUT", 15*time.Second); err != nil {
		return nil, err
	}
	if cfg.TestSourceCacheTTL, err = envDuration("TEST_SOURCE_CACHE_TTL", 30*time.Second); err != nil {
		return nil, err
	}
	if cfg.AppScriptTimeout, err = envDuration("APPSCRIPT_TIMEOUT", 60*time.Second); err != nil {
		return nil, err
	}

	cfg.WebhookSecret = env("WEBHOOK_SECRET", cfg.AppScriptSecret)

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return errors.New("DATABASE_URL обязательна")
	}
	if strings.TrimSpace(c.WebhookSecret) == "" {
		return errors.New("APPSCRIPT_SECRET (или WEBHOOK_SECRET) обязателен: без него вебхук остался бы открытым")
	}
	if c.TestSourcePageSize <= 0 {
		return errors.New("TEST_SOURCE_PAGE_SIZE должен быть положительным")
	}
	return nil
}

func (c *Config) IsDev() bool { return c.AppEnv != "prod" && c.AppEnv != "production" }

func loadDotenv() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, ".env")
		if _, err := os.Stat(candidate); err == nil {
			_ = godotenv.Overload(candidate)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) (int, error) {
	raw := env(key, "")
	if raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: ожидалось число, получено %q", key, raw)
	}
	return v, nil
}

func envDuration(key string, def time.Duration) (time.Duration, error) {
	raw := env(key, "")
	if raw == "" {
		return def, nil
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: ожидалась длительность вида 15s, получено %q", key, raw)
	}
	return v, nil
}
