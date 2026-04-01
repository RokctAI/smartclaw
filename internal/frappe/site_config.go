package frappe

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// SiteConfig represents the relevant fields in a Frappe site_config.json
type SiteConfig struct {
	DBName     string `json:"db_name"`
	DBPassword string `json:"db_password"`
	DBType     string `json:"db_type"`      // mariadb or postgres
	DBHost     string `json:"db_host"`      // defaults to localhost
	DBPort     int    `json:"db_port"`      // defaults based on type
	AppRole    string `json:"app_role"`     // "control" or "tenant"
}

// IngestSiteConfig discovers the local Frappe site_config and returns parsed details.
func IngestSiteConfig() (*SiteConfig, error) {
	// 1. Resolve bench path
	benchPath := os.Getenv("FRAPPE_BENCH")
	if benchPath == "" {
		if cwd, err := os.Getwd(); err == nil {
			benchPath = cwd // default to current dir (likely inside a bench)
		}
	}

	// 2. Resolve site name
	siteName := os.Getenv("SITE")
	if siteName == "" {
		// Fallback to "currentsite.txt" often used by Frappe
		if data, err := os.ReadFile(filepath.Join(benchPath, "sites", "currentsite.txt")); err == nil {
			siteName = strings.TrimSpace(string(data))
		}
	}
	if siteName == "" {
		return nil, fmt.Errorf("SITE environment variable not set and currentsite.txt not found")
	}

	configPath := filepath.Join(benchPath, "sites", siteName, "site_config.json")
	slog.Info("ingesting frappe site config", "path", configPath, "site", siteName)
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read site_config: %w", err)
	}

	var cfg SiteConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse site_config: %w", err)
	}

	// Set defaults
	if cfg.DBHost == "" {
		cfg.DBHost = "localhost"
	}
	if cfg.DBPort == 0 {
		if cfg.DBType == "postgres" {
			cfg.DBPort = 5432
		} else {
			cfg.DBPort = 3306
		}
	}
	if cfg.AppRole == "" {
		cfg.AppRole = "tenant" // default to restricted mode for safety
	}

	return &cfg, nil
}

// IsControlSite returns true if the ingested site is a Control management site.
func (c *SiteConfig) IsControlSite() bool {
	return c.AppRole == "control"
}

// IsTenantSite returns true if the ingested site is a restricted Tenant site.
func (c *SiteConfig) IsTenantSite() bool {
	return c.AppRole == "tenant"
}

// GetPostgresDSN returns a dsn for GoClaw stores if using postgres.
func (c *SiteConfig) GetPostgresDSN() string {
	if c.DBType != "postgres" {
		return ""
	}
	// Note: We might need to handle specific Postgres schema/search_path for Frappe
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", 
		c.DBName, c.DBPassword, c.DBHost, c.DBPort, c.DBName)
}
