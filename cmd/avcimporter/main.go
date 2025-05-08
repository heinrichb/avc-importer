// cmd/avcimporter/main.go
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/heinrichb/avcimporter/pkg/config"
	"github.com/heinrichb/avcimporter/pkg/utils"
)

// Global variables for storing command-line arguments.
// - configPath: Path to config file
// - verbose:    Enable verbose output
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

	// EDI / SFTP flow
	if cfg.EDI.Active {
		// Download any inbound files
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
		payload := []byte(
			"ISA*00*          *00*          *ZZ*1691449        *ZZ*AMAZON         *250508*1540*U*00400*900000014*0*T*>~" +
				"GS*PR*1691449*AMAZON*20250508*1540*900000014*X*004010~" +
				"ST*855*0001~" +
				"BAK*00*AC*CONNECTIVITYTEST*20250508~" +
				"PO1*1*23*UP*23.45*PE*EN*1234567891234~" +
				"ACK*IA*23*UP~" +
				"CTT*1*23~" +
				"SE*6*0001~" +
				"GE*1*900000014~" +
				"IEA*1*900000014~",
		)
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
		// 1. Load last checkpoint
		lastCtrl, err := utils.LoadCheckpoint(cfg.Storage.SavePath)
		if err != nil {
			utils.PrintColored("Failed to load checkpoint: ", err.Error(), "#FF0000")
			os.Exit(1)
		}

		// 2. Get LWA token
		token, err := fetchOAuthToken(cfg)
		if err != nil {
			utils.PrintColored("Error fetching OAuth2 token: ", err.Error(), "#FF0000")
			os.Exit(1)
		}

		// 3. Call SP‑API with SigV4 + access token
		raw, err := fetchFromAPI(cfg, token)
		if err != nil {
			utils.PrintColored("Error fetching data from SP‑API: ", err.Error(), "#FF0000")
			os.Exit(1)
		}

		// 4. Unmarshal & filter new orders
		var resp struct {
			Payload struct {
				PurchaseOrders []struct {
					PurchaseOrderNumber string `json:"purchaseOrderNumber"`
					// …other fields…
				} `json:"payload"`
			}
		}
		if err := json.Unmarshal(raw, &resp); err != nil {
			utils.PrintColored("Invalid JSON from SP‑API: ", err.Error(), "#FF0000")
			os.Exit(1)
		}

		highest := lastCtrl
		for _, po := range resp.Payload.PurchaseOrders {
			if po.PurchaseOrderNumber <= lastCtrl {
				continue
			}
			filename := fmt.Sprintf("%s_%s.json", cfg.Storage.FileName, po.PurchaseOrderNumber)
			path := filepath.Join(cfg.Storage.SavePath, filename)
			if err := os.WriteFile(path, raw, 0o644); err != nil {
				utils.PrintColored("Failed to write order file: ", err.Error(), "#FF0000")
				os.Exit(1)
			}
			utils.PrintColored("Saved purchaseOrder: ", po.PurchaseOrderNumber, "#00FFFF")
			if po.PurchaseOrderNumber > highest {
				highest = po.PurchaseOrderNumber
			}
		}

		// 5. Update checkpoint
		if highest != lastCtrl {
			if err := utils.SaveCheckpoint(cfg.Storage.SavePath, highest); err != nil {
				utils.PrintColored("Failed to save checkpoint: ", err.Error(), "#FF0000")
				os.Exit(1)
			}
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
fetchFromAPI requests data from SP‑API with AWS SigV4 & LWA token.
*/
func fetchFromAPI(cfg *config.Config, token string) ([]byte, error) {
	fullURL := cfg.API.BaseURL + cfg.API.EndpointURL

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Amazon SP‑API requires the LWA token in this header:
	req.Header.Set("x-amz-access-token", token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "avcimporter/1.0")

	// Load AWS SDK config for region and credentials
	awsCfg, err := awsconfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("unable to load AWS SDK config: %w", err)
	}

	// Retrieve the actual credentials struct
	creds, err := awsCfg.Credentials.Retrieve(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve AWS credentials: %w", err)
	}

	// Compute the empty‑body SHA‑256 payload hash
	sum := sha256.Sum256(nil)
	payloadHash := hex.EncodeToString(sum[:])

	// Sign the HTTP request with SigV4
	signer := v4.NewSigner()
	if err := signer.SignHTTP(
		context.TODO(),
		creds,
		req,
		payloadHash,
		"execute-api",
		awsCfg.Region,
		time.Now(),
	); err != nil {
		return nil, fmt.Errorf("sigv4 signing failed: %w", err)
	}

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s", string(body))
	}
	return io.ReadAll(resp.Body)
}
