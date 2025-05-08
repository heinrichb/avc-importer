// cmd/avcimporter/main.go
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/heinrichb/avcimporter/pkg/config"
	"github.com/heinrichb/avcimporter/pkg/utils"
)

/*
Global variables for storing command-line arguments.

- configPath: The path to the configuration file.
- verbose: Enables verbose output.
*/
var (
	configPath string
	verbose    bool
)

func init() {
	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.StringVar(&configPath, "c", "", "Path to config file (shorthand)")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&verbose, "v", false, "Enable verbose output (shorthand)")
}

/*
main is the entry point of AVC Importer CLI.

It parses command-line flags, prints a welcome message, loads the configuration,
and then either runs the EDI/SFTP flow or the SP‑API flow based on the config.
*/
func main() {
	flag.Parse()
	config.Verbose = verbose

	utils.PrintColored("Starting AVC Importer!", "", "#00FFFF")

	if configPath == "" {
		configPath = "configs/default.json"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		utils.PrintColored("Failed to load config: ", err.Error(), "#FF0000")
		os.Exit(1)
	}

	// If neither EDI nor API is active, abort
	if !cfg.EDI.Active && !cfg.API.Active {
		utils.PrintColored("No valid API or EDI configuration found.", "", "#FF0000")
		os.Exit(1)
	}

	// EDI / SFTP flow: download & delete only
	if cfg.EDI.Active {
		files, err := utils.FetchFilesOverSFTP(
			cfg.EDI.Host,
			cfg.EDI.Port,
			cfg.EDI.Username,
			cfg.EDI.PrivateKeyPath,
			cfg.EDI.InboundDir,
			cfg.Storage.SavePath,
		)
		if err != nil {
			utils.PrintColored("SFTP download failed: ", err.Error(), "#FF0000")
			os.Exit(1)
		}
		for _, f := range files {
			utils.PrintColored("Downloaded and removed remote file: ", f, "#00FFFF")
		}
	}

	// SP‑API flow
	if cfg.API.Active {
		token, err := fetchOAuthToken(cfg)
		if err != nil {
			utils.PrintColored("Error fetching OAuth2 token: ", err.Error(), "#FF0000")
			os.Exit(1)
		}
		if err := fetchFromAPI(cfg, token); err != nil {
			utils.PrintColored("Error fetching data from API: ", err.Error(), "#FF0000")
			os.Exit(1)
		}
	}

	utils.PrintColored("AVC Importer CLI completed successfully.", "", "#32CD32")
}

/*
fetchOAuthToken requests an OAuth2 token from Amazon SP‑API.
*/
func fetchOAuthToken(cfg *config.Config) (string, error) {
	requestBody := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": cfg.API.Auth.RefreshToken,
		"client_id":     cfg.API.Auth.ClientID,
		"client_secret": cfg.API.Auth.ClientSecret,
	}

	jsonData, _ := json.Marshal(requestBody)
	req, _ := http.NewRequest("POST", cfg.API.TokenURL, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to fetch token: %s", string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if verbose {
		utils.PrintColored("OAuth2 Token Response: ", fmt.Sprintf("%v", result), "#00FFFF")
	}

	return result["access_token"].(string), nil
}

/*
fetchFromAPI requests data from the configured SP‑API endpoint.
It uses the `endpointUrl` field in the config to determine which endpoint to call.

Parameters:
  - cfg:   The application configuration, containing API details.
  - token: The OAuth2 bearer token for authentication.
*/
func fetchFromAPI(cfg *config.Config, token string) error {
	fullURL := cfg.API.BaseURL + cfg.API.EndpointURL

	utils.PrintColored("Fetching data from: ", fullURL, "#32CD32")

	req, _ := http.NewRequest("GET", fullURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to fetch data from API: %s", string(body))
	}

	body, _ := io.ReadAll(resp.Body)
	if verbose {
		utils.PrintColored("API Response: ", string(body), "#00FFFF")
	} else {
		utils.PrintColored("Data fetched successfully. Use -v for details.", "", "#00FFFF")
	}

	return nil
}
