// pkg/config/config_test.go
package config

import (
	"os"
	"testing"

	"github.com/heinrichb/awsimporter/pkg/utils"
)

const configPath = "test_config.json"

/*
TestApplyDefaults checks that defaults are correctly applied when values are missing.
*/
func TestApplyDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.ApplyDefaults()

	if cfg.API.BaseURL != "https://sellingpartnerapi-na.amazon.com" {
		t.Fatalf("Expected default BaseURL, got %s", cfg.API.BaseURL)
	}
	if cfg.API.TokenURL != "https://api.amazon.com/auth/o2/token" {
		t.Fatalf("Expected default TokenURL, got %s", cfg.API.TokenURL)
	}
	if cfg.API.EndpointURL != "/vendor/orders/v1/purchaseOrders" {
		t.Fatalf("Expected default EndpointURL, got %s", cfg.API.EndpointURL)
	}
	if cfg.Storage.OutputFormat != "json" {
		t.Fatalf("Expected default OutputFormat, got %s", cfg.Storage.OutputFormat)
	}
	if cfg.Storage.SavePath != "output/" {
		t.Fatalf("Expected default SavePath, got %s", cfg.Storage.SavePath)
	}
	if cfg.Storage.FileName != "data_dump" {
		t.Fatalf("Expected default FileName, got %s", cfg.Storage.FileName)
	}
}

/*
TestLoad tests the Load function for:
  - Successful load of valid config
  - Error for non-existing file
*/
func TestLoad(t *testing.T) {
	defer os.Remove(configPath)
	data := `{
		"api": {
			"auth": {
				"clientId":"cid",
				"clientSecret":"cs",
				"applicationId":"aid",
				"refreshToken":"rt"
			},
			"baseUrl":"https://ex.com",
			"tokenUrl":"https://ex.com/t",
			"endpointUrl":"/ex/e"
		},
		"edi": {
			"host":"h","port":22,"username":"u",
			"privateKeyPath":"k","inboundDir":"i","outboundDir":"o"
		},
		"storage":{"outputFormat":"j","savePath":"p/","fileName":"f"}
	}`
	utils.SaveToFile(".", configPath, data)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if cfg.API.Auth.ClientID != "cid" {
		t.Fatalf("Loaded api.auth.clientId, got %s", cfg.API.Auth.ClientID)
	}
	if cfg.EDI.Host != "h" {
		t.Fatalf("Loaded edi.host, got %s", cfg.EDI.Host)
	}
	if cfg.Storage.FileName != "f" {
		t.Fatalf("Loaded storage.fileName, got %s", cfg.Storage.FileName)
	}
}

/*
TestOverrideConfig tests the OverrideConfig function for:
  - Correctly overriding existing values
*/
func TestOverrideConfig(t *testing.T) {
	cfg := &Config{}
	cfg.ApplyDefaults()

	override := ConfigOverride{
		API: &struct {
			Auth        *struct {
				ClientID      *string `json:"clientId"`
				ClientSecret  *string `json:"clientSecret"`
				ApplicationID *string `json:"applicationId"`
				RefreshToken  *string `json:"refreshToken"`
			} `json:"auth"`
			BaseURL     *string `json:"baseUrl"`
			TokenURL    *string `json:"tokenUrl"`
			EndpointURL *string `json:"endpointUrl"`
		}{
			Auth:        &struct{ ClientID *string }{ClientID: utils.PtrString("newCid")},
			EndpointURL: utils.PtrString("/new"),
		},
	}
	cfg.OverrideConfig(override)

	if cfg.API.Auth.ClientID != "newCid" {
		t.Fatalf("Expected overridden auth.clientId, got %s", cfg.API.Auth.ClientID)
	}
	if cfg.API.EndpointURL != "/new" {
		t.Fatalf("Expected overridden endpointUrl, got %s", cfg.API.EndpointURL)
	}
}
