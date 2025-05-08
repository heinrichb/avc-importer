// pkg/utils/config.go
package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/heinrichb/avcimporter/pkg/utils"
)

/*
Global Verbose flag.

This flag determines whether verbose output is enabled.
It is set in `main.go` and used throughout the application.
*/
var Verbose bool

// Config holds configuration data used by AVC Importer CLI.

// Fields:
//   - Version:         The current version of the configuration.
//   - API:             SP‑API credentials and endpoints.
//       - Active:         Enable the SP‑API flow when true.
//       - Auth:
//           - ClientID:      The client ID provided by Amazon SP‑API.
//           - ClientSecret:  The client secret associated with the ClientID.
//           - ApplicationID: The unique identifier for your registered application.
//           - RefreshToken:  The OAuth2 refresh token for renewing access tokens.
//       - BaseURL:        The base URL for SP‑API requests.
//       - TokenURL:       The URL to retrieve OAuth2 tokens.
//       - EndpointURL:    The SP‑API path to fetch data (e.g. purchase orders).
//   - EDI:             SFTP credentials and directories for EDI integration.
//       - Active:            Enable the EDI/SFTP flow when true.
//       - Host:              The SFTP server hostname.
//       - Port:              The SFTP port (usually 22).
//       - DownloadUsername:  Username for downloading (receiving) files.
//       - UploadUsername:    Username for sending (uploading) files.
//       - PrivateKeyPath:    Path to your SSH private key for authentication.
//       - InboundDir:        Directory where PO files land (e.g. "download").
//       - OutboundDir:       Directory for ACKs or feed uploads (e.g. "upload").
//       - SenderID:          Your Amazon‑assigned SFTP ID (the “YOURID” in 997).
//   - Storage:         Settings for where and how to save fetched data.
//       - OutputFormat:     The format to save data (e.g. json).
//       - SavePath:         Directory path for saving files.
//       - FileName:         Base name for saved files.

type Config struct {
	Version string `json:"version"`

	API struct {
		Active bool `json:"active"`
		Auth   struct {
			ClientID      string `json:"clientId"`
			ClientSecret  string `json:"clientSecret"`
			ApplicationID string `json:"applicationId"`
			RefreshToken  string `json:"refreshToken"`
		} `json:"auth"`
		BaseURL     string `json:"baseUrl"`
		TokenURL    string `json:"tokenUrl"`
		EndpointURL string `json:"endpointUrl"`
	} `json:"api"`

	EDI struct {
		Active           bool   `json:"active"`
		Host             string `json:"host"`
		Port             int    `json:"port"`
		DownloadUsername string `json:"downloadUsername"`
		UploadUsername   string `json:"uploadUsername"`
		PrivateKeyPath   string `json:"privateKeyPath"`
		InboundDir       string `json:"inboundDir"`
		OutboundDir      string `json:"outboundDir"`
		SenderID         string `json:"senderId"`
	} `json:"edi"`

	Storage struct {
		OutputFormat string `json:"outputFormat"`
		SavePath     string `json:"savePath"`
		FileName     string `json:"fileName"`
	} `json:"storage"`
}

/*
ConfigOverride represents a partial configuration used for overriding values.
All fields are pointers, so that nil indicates "no override" while non-nil
values replace existing configuration.
*/
type ConfigOverride struct {
	Version *string `json:"version"`

	API *struct {
		Auth *struct {
			ClientID      *string `json:"clientId"`
			ClientSecret  *string `json:"clientSecret"`
			ApplicationID *string `json:"applicationId"`
			RefreshToken  *string `json:"refreshToken"`
		} `json:"auth"`
		BaseURL     *string `json:"baseUrl"`
		TokenURL    *string `json:"tokenUrl"`
		EndpointURL *string `json:"endpointUrl"`
	} `json:"api"`

	EDI *struct {
		Host             *string `json:"host"`
		Port             *int    `json:"port"`
		DownloadUsername *string `json:"downloadUsername"`
		UploadUsername   *string `json:"uploadUsername"`
		PrivateKeyPath   *string `json:"privateKeyPath"`
		InboundDir       *string `json:"inboundDir"`
		OutboundDir      *string `json:"outboundDir"`
		SenderID         *string `json:"senderId"`
	} `json:"edi"`

	Storage *struct {
		OutputFormat *string `json:"outputFormat"`
		SavePath     *string `json:"savePath"`
		FileName     *string `json:"fileName"`
	} `json:"storage"`
}

func (cfg *Config) ApplyDefaults() {
	if cfg.API.BaseURL == "" {
		cfg.API.BaseURL = "https://sellingpartnerapi-na.amazon.com"
	}
	if cfg.API.TokenURL == "" {
		cfg.API.TokenURL = "https://api.amazon.com/auth/o2/token"
	}
	if cfg.API.EndpointURL == "" {
		cfg.API.EndpointURL = "/vendor/orders/v1/purchaseOrders"
	}
	if cfg.Storage.OutputFormat == "" {
		cfg.Storage.OutputFormat = "json"
	}
	if cfg.Storage.SavePath == "" {
		cfg.Storage.SavePath = "output/"
	}
	if cfg.Storage.FileName == "" {
		cfg.Storage.FileName = "data_dump"
	}
}

/*
Load reads configuration data from the specified filePath.

Parameters:
  - filePath: The path to the JSON configuration file.

Returns:
  - A Config pointer populated from the file and defaults.
  - An error if the file is missing or invalid JSON.
*/
func Load(filePath string) (*Config, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file %s does not exist", filePath)
	}
	utils.PrintColored("Loaded config from: ", filePath, "#32CD32")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	cfg.ApplyDefaults()
	if Verbose {
		utils.PrintNonEmptyFields("", cfg)
	}

	return &cfg, nil
}

/*
OverrideConfig applies any non-nil values from overrides into cfg.
*/
func (cfg *Config) OverrideConfig(o ConfigOverride) {
	if o.Version != nil {
		cfg.Version = *o.Version
	}
	if o.API != nil {
		if o.API.Auth != nil {
			if o.API.Auth.ClientID != nil {
				cfg.API.Auth.ClientID = *o.API.Auth.ClientID
			}
			if o.API.Auth.ClientSecret != nil {
				cfg.API.Auth.ClientSecret = *o.API.Auth.ClientSecret
			}
			if o.API.Auth.ApplicationID != nil {
				cfg.API.Auth.ApplicationID = *o.API.Auth.ApplicationID
			}
			if o.API.Auth.RefreshToken != nil {
				cfg.API.Auth.RefreshToken = *o.API.Auth.RefreshToken
			}
		}
		if o.API.BaseURL != nil {
			cfg.API.BaseURL = *o.API.BaseURL
		}
		if o.API.TokenURL != nil {
			cfg.API.TokenURL = *o.API.TokenURL
		}
		if o.API.EndpointURL != nil {
			cfg.API.EndpointURL = *o.API.EndpointURL
		}
	}
	if o.EDI != nil {
		if o.EDI.Host != nil {
			cfg.EDI.Host = *o.EDI.Host
		}
		if o.EDI.Port != nil {
			cfg.EDI.Port = *o.EDI.Port
		}
		if o.EDI.DownloadUsername != nil {
			cfg.EDI.DownloadUsername = *o.EDI.DownloadUsername
		}
		if o.EDI.UploadUsername != nil {
			cfg.EDI.UploadUsername = *o.EDI.UploadUsername
		}
		if o.EDI.PrivateKeyPath != nil {
			cfg.EDI.PrivateKeyPath = *o.EDI.PrivateKeyPath
		}
		if o.EDI.InboundDir != nil {
			cfg.EDI.InboundDir = *o.EDI.InboundDir
		}
		if o.EDI.OutboundDir != nil {
			cfg.EDI.OutboundDir = *o.EDI.OutboundDir
		}
		if o.EDI.SenderID != nil {
			cfg.EDI.SenderID = *o.EDI.SenderID
		}
	}
	if o.Storage != nil {
		if o.Storage.OutputFormat != nil {
			cfg.Storage.OutputFormat = *o.Storage.OutputFormat
		}
		if o.Storage.SavePath != nil {
			cfg.Storage.SavePath = *o.Storage.SavePath
		}
		if o.Storage.FileName != nil {
			cfg.Storage.FileName = *o.Storage.FileName
		}
	}
}
