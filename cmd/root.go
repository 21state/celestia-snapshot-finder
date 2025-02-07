package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/21state/celestia-snapshot-finder/internal/config"
	"github.com/21state/celestia-snapshot-finder/internal/downloader"
	"github.com/21state/celestia-snapshot-finder/internal/provider"
	"github.com/21state/celestia-snapshot-finder/internal/speedtest"
	"github.com/21state/celestia-snapshot-finder/internal/version"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const providersURL = "https://raw.githubusercontent.com/21state/celestia-snapshots/refs/heads/main/providers.yaml"

var (
	chainID  string
	manual   bool
	debug    bool
	rootCmd  = &cobra.Command{
		Use:   "celestia-snapshot-finder [node-type] [snapshot-type]",
		Short: "Download Celestia node snapshots",
		Long: `A CLI tool for downloading Celestia node snapshots with direct URLs.
Supports different node types and snapshot types with automatic or manual selection.`,
		Args:    cobra.ExactArgs(2),
		RunE:    runRoot,
		Version: version.Version,
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&chainID, "chain-id", "n", "celestia", "Chain ID")
	rootCmd.PersistentFlags().BoolVarP(&manual, "manual", "m", false, "Enable manual selection")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug mode with extra information")
}

func debugPrint(format string, a ...interface{}) {
	if debug {
		debugColor := color.New(color.FgYellow).SprintFunc()
		timestamp := time.Now().Format("15:04:05.000")
		fmt.Printf("%s %s %s\n", debugColor("[DEBUG]"), timestamp, fmt.Sprintf(format, a...))
	}
}

func printInfo(format string, a ...interface{}) {
	infoColor := color.New(color.FgCyan).SprintFunc()
	fmt.Printf("%s %s\n", infoColor("[INFO]"), fmt.Sprintf(format, a...))
}

func Execute() error {
	return rootCmd.Execute()
}

func fetchProviders() (*config.Config, error) {
	debugPrint("Fetching providers from %s", providersURL)
	
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	resp, err := client.Get(providersURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch providers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch providers: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read providers data: %w", err)
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse providers data: %w", err)
	}

	debugPrint("Successfully fetched providers configuration")
	return &cfg, nil
}

func runRoot(cmd *cobra.Command, args []string) error {
	nodeType := args[0]
	snapshotType := args[1]

	debugPrint("Starting with raw arguments: nodeType=%s, snapshotType=%s, chainID=%s, manual=%v", 
		nodeType, snapshotType, chainID, manual)

	nodeType, snapshotType, err := validateArgs(nodeType, snapshotType)
	if err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	printInfo("Searching for %s-%s snapshots [chain-id: %s, mode: %s]", 
		nodeType, snapshotType, chainID, map[bool]string{true: "manual", false: "auto"}[manual])

	cfg, err := fetchProviders()
	if err != nil {
		return fmt.Errorf("failed to load providers configuration: %w", err)
	}

	debugPrint("Configuration loaded successfully with %d providers", len(cfg.Providers))
	if debug {
		for _, p := range cfg.Providers {
			debugPrint("Provider %s has %d snapshots", p.Name, len(p.Snapshots))
			for _, s := range p.Snapshots {
				debugPrint("  - Type: %s, Chain ID: %s", s.Type, s.ChainID)
				debugPrint("    URL: %s", s.URL)
			}
		}
	}

	debugPrint("Initializing managers")
	providerMgr := provider.NewManager(cfg.Providers, debugPrint)
	speedTester := speedtest.NewSpeedTester(debugPrint)
	downloadMgr := downloader.NewManager()

	debugPrint("Filtering snapshots for type=%s-%s and chainID=%s", nodeType, snapshotType, chainID)
	providers := providerMgr.FilterProviders(nodeType, snapshotType, chainID)
	if len(providers) == 0 {
		return fmt.Errorf("no snapshots found for node type '%s' and snapshot type '%s'", nodeType, snapshotType)
	}

	printInfo("Found %d matching snapshots from %d providers", len(providers), len(cfg.Providers))
	debugPrint("Found %d matching snapshots", len(providers))
	if debug {
		for _, p := range providers {
			debugPrint("Matched snapshot from %s: %s", p.Name, p.URL)
		}
	}

	debugPrint("Running health checks on snapshots")
	providers = providerMgr.CheckHealth(providers)
	if len(providers) == 0 {
		return fmt.Errorf("no healthy snapshots found")
	}

	printInfo("%d snapshots are healthy and ready for download", len(providers))
	debugPrint("%d snapshots passed health check", len(providers))
	if debug {
		for _, p := range providers {
			debugPrint("Healthy snapshot from %s", p.Name)
		}
	}

	printInfo("Testing download speeds...")
	debugPrint("Starting speed tests")
	providers = speedTester.TestProviders(providers)

	sort.Slice(providers, func(i, j int) bool {
		return providers[i].Speed > providers[j].Speed
	})

	var selectedProvider provider.ProviderInfo
	if len(providers) == 0 {
		return fmt.Errorf("no snapshots available")
	} else if len(providers) == 1 {
		selectedProvider = providers[0]
		debugPrint("Only one snapshot available from %s", selectedProvider.Name)
	} else {
		if manual {
			fmt.Println("\nAvailable snapshots:")
			for i, p := range providers {
				fmt.Printf("%d. %s (%.2f MB/s)\n", i+1, p.Name, p.Speed)
			}

			var choice int
			for {
				fmt.Printf("\nSelect snapshot (1-%d): ", len(providers))
				fmt.Scanf("%d", &choice)
				if choice > 0 && choice <= len(providers) {
					break
				}
				fmt.Printf("Invalid choice. Please enter a number between 1 and %d\n", len(providers))
			}
			selectedProvider = providers[choice-1]
		} else {
			selectedProvider = providers[0]
			debugPrint("Automatically selected fastest snapshot from %s", selectedProvider.Name)
		}
	}

	printInfo("Selected snapshot from %s (%.2f MB/s)", selectedProvider.Name, selectedProvider.Speed)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	downloadDir := filepath.Join(homeDir, "celestia-snapshots")
	debugPrint("Download directory: %s", downloadDir)

	printInfo("Starting download from %s", selectedProvider.Name)
	debugPrint("Starting download from URL: %s", selectedProvider.URL)
	result, err := downloadMgr.Download(selectedProvider.URL, downloadDir)
	if err != nil {
		debugPrint("Download failed: %v", err)
		return fmt.Errorf("failed to download snapshot: %w", err)
	}

	success := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("\n%s Download completed!\n", success("✓"))
	fmt.Printf("Snapshot saved to: %s\n", result.Path)
	
	sizeGB := float64(result.Size) / 1000 / 1000 / 1000
	fmt.Printf("Size: %.2f GB\n", sizeGB)
	debugPrint("Download completed successfully. File size: %d bytes (%.2f GB)", 
		result.Size, sizeGB)

	return nil
}

func validateArgs(nodeType, snapshotType string) (string, string, error) {
	nodeTypeMap := map[string]string{
		"c": "consensus",
		"b": "bridge",
		"consensus": "consensus",
		"bridge":    "bridge",
	}

	snapshotTypeMap := map[string]string{
		"p": "pruned",
		"a": "archive",
		"pruned":  "pruned",
		"archive": "archive",
	}

	fullNodeType, validNode := nodeTypeMap[nodeType]
	if !validNode {
		return "", "", fmt.Errorf("invalid node type: %s. Must be one of: consensus (c), bridge (b)", nodeType)
	}

	fullSnapshotType, validSnapshot := snapshotTypeMap[snapshotType]
	if !validSnapshot {
		return "", "", fmt.Errorf("invalid snapshot type: %s. Must be one of: pruned (p), archive (a)", snapshotType)
	}

	return fullNodeType, fullSnapshotType, nil
}
