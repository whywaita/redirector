package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strconv"
)

// Config holds the application configuration.
type Config struct {
	Destination string
	StatusCode  int
	Port        int
}

// ParseConfig reads configuration from CLI flags and environment variables.
// CLI flags take precedence over environment variables.
func ParseConfig() (*Config, error) {
	cfg := &Config{}

	// CLI flags
	flag.StringVar(&cfg.Destination, "destination", "", "redirect destination base URL")
	flag.StringVar(&cfg.Destination, "d", "", "redirect destination base URL (short)")
	flag.IntVar(&cfg.StatusCode, "status", 0, "redirect status code (301/302/307/308)")
	flag.IntVar(&cfg.StatusCode, "s", 0, "redirect status code (short)")
	flag.IntVar(&cfg.Port, "port", 0, "listen port")
	flag.IntVar(&cfg.Port, "p", 0, "listen port (short)")
	flag.Parse()

	// Apply environment variables for unset values (flag default is zero value)
	if cfg.Destination == "" {
		cfg.Destination = os.Getenv("REDIRECT_DESTINATION")
	}
	if cfg.StatusCode == 0 {
		if v := os.Getenv("REDIRECT_STATUS"); v != "" {
			code, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("invalid REDIRECT_STATUS: %w", err)
			}
			cfg.StatusCode = code
		}
	}
	if cfg.Port == 0 {
		if v := os.Getenv("PORT"); v != "" {
			port, err := strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("invalid PORT: %w", err)
			}
			cfg.Port = port
		}
	}

	// Apply defaults
	if cfg.StatusCode == 0 {
		cfg.StatusCode = 302
	}
	if cfg.Port == 0 {
		cfg.Port = 8080
	}

	// Validation
	if cfg.Destination == "" {
		return nil, fmt.Errorf("REDIRECT_DESTINATION is required (set via --destination flag or REDIRECT_DESTINATION env var)")
	}
	if _, err := url.Parse(cfg.Destination); err != nil {
		return nil, fmt.Errorf("invalid REDIRECT_DESTINATION %q: %w", cfg.Destination, err)
	}

	switch cfg.StatusCode {
	case 301, 302, 307, 308:
		// valid
	default:
		return nil, fmt.Errorf("invalid REDIRECT_STATUS %d (must be 301, 302, 307, or 308)", cfg.StatusCode)
	}

	if cfg.Port < 1 || cfg.Port > 65535 {
		return nil, fmt.Errorf("invalid PORT %d (must be between 1 and 65535)", cfg.Port)
	}

	return cfg, nil
}
