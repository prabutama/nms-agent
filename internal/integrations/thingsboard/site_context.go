package thingsboard

import "fmt"

func ValidateSiteContext(cfg Config) error {
	if cfg.API.BaseURL == "" {
		return fmt.Errorf("thingsboard.api.base_url is required")
	}
	if cfg.API.APIKey == "" {
		return fmt.Errorf("thingsboard.api.api_key is required")
	}
	if cfg.Site.Key == "" {
		return fmt.Errorf("thingsboard.site.key is required")
	}
	if cfg.Site.AssetID == "" {
		return fmt.Errorf("thingsboard.site.asset_id is required")
	}
	return nil
}
