package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v3"

	"github.com/andrew-aiken/vmGoat/pkg/logger"
	"github.com/andrew-aiken/vmGoat/pkg/types"
)

func Update(ctx context.Context, cli *cli.Command) error {
	log := logger.Get()

	currentVersion := cli.Root().Version
	log.Info().Msgf("Current version: %s", currentVersion)

	// Check for latest release
	latestVersion, err := getLatestReleaseVersion()
	if err != nil {
		return fmt.Errorf("failed to check for latest version: %v", err)
	}

	log.Debug().Msgf("Latest version: %s", latestVersion)

	// Compare versions
	isNewer, err := isVersionNewer(currentVersion, latestVersion)
	if err != nil {
		return fmt.Errorf("failed to compare versions: %v", err)
	}

	if !isNewer {
		log.Info().Msgf("You are already running the latest version!")
		return nil
	}

	log.Info().Msgf("A newer version (%s) is available!\n", latestVersion)

	// Ask for confirmation unless auto-approve is set
	if !cli.Bool("auto-approve") {
		fmt.Printf("Do you want to update? (y/N): ")
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			log.Warn().Msgf("Update cancelled.")
			return nil
		}
	}

	// Download and install the update
	if err := downloadAndInstallUpdate(latestVersion); err != nil {
		return fmt.Errorf("failed to update: %v", err)
	}

	log.Info().Msgf("Update completed successfully!")

	return nil
}

// getLatestReleaseVersion fetches the latest non-pre-release version from GitHub
func getLatestReleaseVersion() (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	url := "https://api.github.com/repos/andrew-aiken/vmGoat/releases"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	// Set User-Agent header to avoid rate limiting
	req.Header.Set("User-Agent", "vmGoat-updater")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch release info: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []types.GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", fmt.Errorf("failed to parse release info: %v", err)
	}

	// Find the latest non-pre-release
	for _, release := range releases {
		if !release.PreRelease {
			return release.TagName, nil
		}
	}

	return "", fmt.Errorf("no non-pre-release version found")
}

// isVersionNewer compares two version strings and returns true if newVersion is newer than currentVersion
func isVersionNewer(currentVersion, newVersion string) (bool, error) {
	// Handle development versions
	if currentVersion == "0.0.0-undefined" || strings.Contains(currentVersion, "dev") {
		return false, fmt.Errorf("Running a development version, skipping version check")
	}

	// Parse versions using hashicorp/go-version for semantic version comparison
	current, err := version.NewVersion(currentVersion)
	if err != nil {
		return false, fmt.Errorf("failed to parse current version: %v", err)
	}

	latest, err := version.NewVersion(newVersion)
	if err != nil {
		return false, fmt.Errorf("failed to parse latest version: %v", err)
	}

	return latest.GreaterThan(current), nil
}

// downloadAndInstallUpdate downloads and installs the new version
func downloadAndInstallUpdate(version string) error {
	// Get OS and architecture
	osName := runtime.GOOS
	arch := runtime.GOARCH

	// Construct the asset name based on OS and architecture
	var assetName string
	switch osName {
	case "darwin":
		assetName = fmt.Sprintf("vmGoat-%s-%s", osName, arch)
	case "linux":
		assetName = fmt.Sprintf("vmGoat-%s-%s", osName, arch)
	default:
		return fmt.Errorf("unsupported operating system: %s", osName)
	}

	downloadUrl := fmt.Sprintf("https://github.com/andrew-aiken/vmGoat/releases/download/%s/%s", version, assetName)

	log.Debug().Msgf("Downloading binary from: %s", downloadUrl)

	if err := downloadFile(downloadUrl); err != nil {
		return fmt.Errorf("failed to download update: %v", err)
	}

	return nil
}

// downloadFile downloads a file from a URL and returns the path to the temporary file
func downloadFile(url string) error {
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Write the downloaded content directly to ./vmGoat
	outPath := "./vmGoat"
	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		os.Remove(outPath)
		return fmt.Errorf("failed to write downloaded file: %v", err)
	}

	// Make the file executable
	if err := os.Chmod(outPath, 0750); err != nil {
		return fmt.Errorf("failed to set executable permissions: %v", err)
	}

	return nil
}
