package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr         string
	DataDir      string
	IngestSecret string
	Window       time.Duration
	IdemTTL      time.Duration
	Workers      int
	PublicBase   string
	CuttingsPath string
}

func Load() (Config, error) {
	c := Config{
		Addr:         env("MUDLOG_ADDR", ":8080"),
		DataDir:      env("MUDLOG_DATA_DIR", "./data"),
		IngestSecret: env("MUDLOG_INGEST_SECRET", "dev-rig-secret"),
		Window:       durSec("MUDLOG_WINDOW_SEC", 300),
		IdemTTL:      durSec("MUDLOG_IDEM_TTL_SEC", 86400),
		Workers:      envInt("MUDLOG_WORKERS", 4),
		PublicBase:   strings.TrimRight(env("MUDLOG_PUBLIC_BASE", "http://127.0.0.1:8080"), "/"),
		CuttingsPath: "/api/v1/cuttings",
	}
	if c.IngestSecret == "" {
		return c, fmt.Errorf("MUDLOG_INGEST_SECRET is empty")
	}
	if c.Workers < 1 {
		c.Workers = 1
	}
	if c.Workers > 32 {
		c.Workers = 32
	}
	if !strings.HasPrefix(c.Addr, ":") && !strings.Contains(c.Addr, ":") {
		c.Addr = ":" + c.Addr
	}
	return c, nil
}

func env(key, def string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	return v
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func durSec(key string, def int) time.Duration {
	n := envInt(key, def)
	if n < 1 {
		n = def
	}
	return time.Duration(n) * time.Second
}
