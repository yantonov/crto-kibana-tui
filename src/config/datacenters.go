package config

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	dcFileName   = "datacenters.yaml"
	dcFetchTimeout = 15 * time.Second
)

var envAPIURLs = map[string]string{
	"prod":    "https://lb-control-plane.prod.crto.in/api/v1/datacenters",
	"preprod": "https://lb-control-plane.preprod.crto.in/api/v1/datacenters",
}

// defaultDCFilePath returns the datacenters.yaml path relative to configPath.
func defaultDCFilePath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), dcFileName)
}

// loadOrFetchAllDatacenters reads datacenters.yaml from dcPath.
// If the file does not exist it fetches each environment from its control-plane
// API, sorts the datacenter codes, writes the file, and returns the result.
func loadOrFetchAllDatacenters(dcPath string) (map[string][]string, error) {
	envs, err := loadDCFile(dcPath)
	if err == nil {
		return envs, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read %s: %w", dcPath, err)
	}

	// File doesn't exist — fetch from APIs.
	envs = make(map[string][]string, len(envAPIURLs))
	for env, apiURL := range envAPIURLs {
		dcs, fetchErr := fetchDatacenters(apiURL)
		if fetchErr != nil {
			return nil, fmt.Errorf("fetch datacenters for %s: %w", env, fetchErr)
		}
		envs[env] = dcs
	}

	if saveErr := saveDCFile(dcPath, envs); saveErr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save %s: %v\n", dcPath, saveErr)
	}

	return envs, nil
}

func loadDCFile(path string) (map[string][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var data map[string][]string
	if err := yaml.NewDecoder(f).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return data, nil
}

func saveDCFile(path string, envs map[string][]string) error {
	data, err := yaml.Marshal(envs)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func fetchDatacenters(apiURL string) ([]string, error) {
	client := &http.Client{Timeout: dcFetchTimeout}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", apiURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %s", apiURL, resp.Status)
	}

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode response from %s: %w", apiURL, err)
	}

	dcs := make([]string, 0, len(raw))
	for k := range raw {
		dcs = append(dcs, k)
	}
	sort.Strings(dcs)
	return dcs, nil
}
