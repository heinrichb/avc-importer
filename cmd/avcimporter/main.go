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

- configPath: Path to config file
- verbose:    Enable verbose output
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

	// EDI / SFTP flow: download incoming files, then send ConnectivityTest
	if cfg.EDI.Active {
		// Download any inbound files, controlling deletion via config
		files, err := utils.FetchFilesOverSFTP(
			cfg.EDI.Host,
			cfg.EDI.Port,
			cfg.EDI.DownloadUsername,
			cfg.EDI.PrivateKeyPath,
			cfg.EDI.InboundDir,
			cfg.Storage.SavePath,
			cfg.EDI.DeleteAfterDownload,
		)
		if err != nil {
			utils.PrintColored("SFTP download failed: ", err.Error(), "#FF0000")
			os.Exit(1)
		}
		for _, f := range files {
			if cfg.EDI.DeleteAfterDownload {
				utils.PrintColored("Downloaded and removed remote file: ", f, "#00FFFF")
			} else {
				utils.PrintColored("Downloaded remote file: ", f, "#00FFFF")
			}
		}

		// Send Amazon connectivity test file
		// TODO: Replace with your actual payload
		// This is a sample EDI 855 file for testing purposes.
		payload := []byte("ISA*00*          *00*          *ZZ*1691449        *ZZ*AMAZON         *250508*1540*U*00400*900000014*0*T*>~GS*PR*1691449*AMAZON*20250508*1540*900000014*X*004010~ST*855*0001~BAK*00*AC*CONNECTIVITYTEST*20250508~PO1*1*23*UP*23.45*PE*EN*1234567891234~ACK*IA*23*UP~CTT*1*23~SE*6*0001~GE*1*900000014~IEA*1*900000014~")
		if err := utils.UploadFileOverSFTP(
			cfg.EDI.Host,
			cfg.EDI.Port,
			cfg.EDI.UploadUsername,
			cfg.EDI.PrivateKeyPath,
			cfg.EDI.OutboundDir,
			"ConnectivityTest",
			payload,
		); err != nil {
			utils.PrintColored("ConnectivityTest upload failed: ", err.Error(), "#FF0000")
			os.Exit(1)
		}
		utils.PrintColored("ConnectivityTest uploaded successfully.", "", "#00FFFF")
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
