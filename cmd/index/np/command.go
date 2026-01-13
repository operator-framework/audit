// Copyright 2021 The Audit Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package np implements NetworkPolicy audit functionality.
//
// This implementation uses external `opm render` for efficient bundle extraction
// combined with internal `declcfg.LoadReader` for pure Go YAML parsing.
// This approach avoids CGO dependencies while leveraging the efficiency of opm render.
package np

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	auditpkg "github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/operator-registry/alpha/declcfg"
)

// flags holds the command-line flags for the np command
var flags struct {
	Indexes []string
	Package string
	Workers int
}

// bundleJob represents a bundle processing job
type bundleJob struct {
	image       string
	reportMutex *sync.Mutex
	reportFile  *os.File
	cacheDir    string
}

// bundleResult represents the result of processing a bundle
type bundleResult struct {
	image         string
	filesScanned  int
	binarySkipped int
	foundPaths    []string
	err           error
	fromCache     bool
}

// NewCmd returns the cobra command for the np sub-command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "np",
		Short:   "audit index catalogs for NetworkPolicy resources",
		Long:    `Scan provided catalog index images for any NetworkPolicy resources included in bundles.`,
		PreRunE: validation,
		RunE:    run,
	}

	// required: one or more catalog indexes
	cmd.Flags().StringSliceVarP(&flags.Indexes, "indexes", "i", []string{}, "Catalog index images to audit (required)")
	_ = cmd.MarkFlagRequired("indexes")

	// optional: filter to a single package name
	cmd.Flags().StringVarP(&flags.Package, "package", "p", "", "Limit scan to a specific package name")

	// number of workers for parallel processing
	cmd.Flags().IntVar(&flags.Workers, "workers", runtime.NumCPU(),
		"Number of worker goroutines for parallel bundle processing")

	return cmd
}

// validation verifies required flags and flag values
func validation(cmd *cobra.Command, args []string) error {
	// require at least one index
	if len(flags.Indexes) == 0 {
		return fmt.Errorf("invalid value for --indexes: at least one index must be specified")
	}
	// validate workers count
	if flags.Workers < 1 {
		return fmt.Errorf("invalid value for --workers: %d; must be at least 1", flags.Workers)
	}
	return nil
}

// run executes the np audit logic
func run(cmd *cobra.Command, args []string) error {
	startTime := time.Now()
	log.Info("Starting NetworkPolicy audit...")
	// create report file
	reportName := fmt.Sprintf("np_report_%s.txt", time.Now().Format("20060102_150405"))
	reportFile, err := os.Create(reportName)
	if err != nil {
		return fmt.Errorf("unable to create report file %s: %v", reportName, err)
	}
	defer reportFile.Close()
	auditpkg.GenerateTemporaryDirs()

	// Create cache directory for processed bundles
	cacheDir := "cache"
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("unable to create cache directory: %v", err)
	}

	// Create a mutex for thread-safe report writing
	var reportMutex sync.Mutex

	// Process each index using hybrid opm+declcfg extraction approach
	for _, index := range flags.Indexes {
		log.Infof("Processing index %s...", index)

		// Extract ALL unique bundle images at once
		bundleImages, err := getAllBundleImages(index)
		if err != nil {
			log.Errorf("unable to extract bundle images from index %s: %v", index, err)
			continue
		}

		log.Infof("Found %d unique bundle images in index %s", len(bundleImages), index)

		// Note: Package filtering with render approach would require additional metadata extraction
		// For now, we process all bundles and note the limitation
		if flags.Package != "" {
			log.Warnf("Package filtering not yet implemented with render approach - processing all bundles")
		}

		// write index header
		reportFile.WriteString(fmt.Sprintf("%s (hybrid opm+declcfg)\n", index))
		reportFile.WriteString(fmt.Sprintf("Processing %d unique bundle images\n", len(bundleImages)))

		// Create channels for worker communication
		jobs := make(chan bundleJob, len(bundleImages))
		results := make(chan bundleResult, len(bundleImages))

		// Start worker goroutines
		var wg sync.WaitGroup
		for i := 0; i < flags.Workers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				processBundleWorker(jobs, results)
			}()
		}

		// Send all bundle images to workers
		go func() {
			defer close(jobs)
			for _, image := range bundleImages {
				jobs <- bundleJob{
					image:       image,
					reportMutex: &reportMutex,
					reportFile:  reportFile,
					cacheDir:    cacheDir,
				}
			}
		}()

		// Collect results
		bundleResults := make(map[string]bundleResult)
		for i := 0; i < len(bundleImages); i++ {
			result := <-results
			bundleResults[result.image] = result
		}

		// Wait for all workers to finish
		wg.Wait()
		close(results)

		// Track statistics for summary report
		var successCount, errorCount, cacheHitCount int
		var errorDetails []string

		// Write results (maintain order by sorting images)
		sort.Strings(bundleImages)
		for _, image := range bundleImages {
			result, exists := bundleResults[image]
			if !exists {
				errorCount++
				errorDetails = append(errorDetails, fmt.Sprintf("No result for %s", getBundleNameFromImage(image)))
				continue
			}

			if result.err != nil {
				errorCount++
				log.Errorf("Error processing bundle image %s: %v", image, result.err)
				errorDetails = append(errorDetails, fmt.Sprintf("%s: %v", getBundleNameFromImage(image), result.err))
				continue
			}

			successCount++
			if result.fromCache {
				cacheHitCount++
			}

			// write report entries, including skipped binary count
			reportMutex.Lock()
			bundleName := getBundleNameFromImage(image)
			reportFile.WriteString(fmt.Sprintf("    %s: %d files scanned, skipped %d binary files\n",
				bundleName, result.filesScanned, result.binarySkipped))
			for _, rel := range result.foundPaths {
				reportFile.WriteString(fmt.Sprintf("        Found NetworkPolicy resource in %s: %s\n",
					bundleName, rel))
			}
			reportMutex.Unlock()
		}

		// Write summary statistics for this index
		reportMutex.Lock()
		reportFile.WriteString(fmt.Sprintf("\nSummary for %s:\n", index))
		reportFile.WriteString(fmt.Sprintf("  Total bundles: %d\n", len(bundleImages)))
		reportFile.WriteString(fmt.Sprintf("  Successful: %d\n", successCount))
		reportFile.WriteString(fmt.Sprintf("  Failed: %d\n", errorCount))
		reportFile.WriteString(fmt.Sprintf("  Cache hits: %d\n", cacheHitCount))
		reportFile.WriteString(fmt.Sprintf("  Success rate: %.1f%%\n", float64(successCount)/float64(len(bundleImages))*100))

		// Include error details if there were failures
		if errorCount > 0 {
			reportFile.WriteString("\nError details:\n")
			for _, errDetail := range errorDetails {
				reportFile.WriteString(fmt.Sprintf("  - %s\n", errDetail))
			}
		}
		reportFile.WriteString("\n")
		reportMutex.Unlock()

		// Log summary to console as well
		log.Infof("Index %s summary: %d successful, %d failed, %d cache hits out of %d total bundles",
			index, successCount, errorCount, cacheHitCount, len(bundleImages))
	}
	auditpkg.CleanupTemporaryDirs()

	// Calculate and log execution time
	duration := time.Since(startTime)
	log.Infof("Operation completed in %v", duration)

	// Write timing information to report
	reportMutex.Lock()
	reportFile.WriteString(fmt.Sprintf("\nAudit completed in %v\n", duration))
	reportMutex.Unlock()

	return nil
}

// getAllBundleImages extracts all unique bundle images using hybrid approach (external opm + internal parsing)
func getAllBundleImages(indexImage string) ([]string, error) {
	log.Infof("Extracting bundle images from index %s using hybrid opm+declcfg approach...", indexImage)

	// Use external opm render (pre-compiled, works everywhere) + internal declcfg parsing (pure Go)
	// This avoids image registry dependencies (gpgme) while giving us structured data

	// Execute opm render command as external process
	cmd := exec.Command("opm", "render", "-o", "yaml", indexImage)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run opm render for index %s: %v", indexImage, err)
	}

	// Parse the YAML output using declcfg.LoadReader
	reader := bytes.NewReader(output)
	cfg, err := declcfg.LoadReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse render output for index %s: %v", indexImage, err)
	}

	// Extract bundle images from the declarative config
	bundleImages := extractBundleImagesFromDeclarativeConfig(cfg)

	log.Infof("Extracted %d unique bundle images from index %s using hybrid approach", len(bundleImages), indexImage)
	return bundleImages, nil
}

// extractBundleImagesFromDeclarativeConfig extracts bundle images from a DeclarativeConfig
func extractBundleImagesFromDeclarativeConfig(cfg *declcfg.DeclarativeConfig) []string {
	var bundleImages []string
	imageSet := make(map[string]struct{})

	// Extract images from bundles
	for _, bundle := range cfg.Bundles {
		if bundle.Image != "" {
			imageSet[bundle.Image] = struct{}{}
		}
	}

	// Convert set to slice (sorting will be done later for report order)
	for image := range imageSet {
		bundleImages = append(bundleImages, image)
	}

	return bundleImages
}

// getBundleNameFromImage extracts a bundle name from an image path for directory naming
func getBundleNameFromImage(image string) string {
	// Extract the last part of the image path, replacing special characters for filesystem safety
	parts := strings.Split(image, "/")
	name := parts[len(parts)-1]
	// Replace problematic characters with underscores
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "@", "_")
	return name
}

// processBundleWorker processes bundle jobs from the jobs channel and sends results to the results channel
func processBundleWorker(jobs <-chan bundleJob, results chan<- bundleResult) {
	for job := range jobs {
		result := bundleResult{
			image: job.image,
		}

		// extract bundle name from image path for directory creation
		bundleName := getBundleNameFromImage(job.image)

		// Check cache first - if bundle already processed, use cached results
		cacheFile := filepath.Join(job.cacheDir, bundleName+".json")
		if cachedResult, err := loadCachedResult(cacheFile); err == nil {
			log.Infof("Using cached result for bundle %s", job.image)
			result = *cachedResult
			result.image = job.image // ensure image field is correct
			result.fromCache = true  // mark as cache hit
			results <- result
			continue
		}

		// Use opm alpha bundle unpack for faster extraction (only operator manifests)
		bundleDir := filepath.Join(job.cacheDir, "extracts", bundleName)

		// Check if bundle manifests are already extracted (manifest extraction caching!)
		var extractionErr error
		if _, err := os.Stat(bundleDir); err == nil {
			log.Infof("Using previously extracted manifests for bundle %s", job.image)
		} else {
			log.Infof("Extracting bundle manifests %s using opm alpha bundle unpack", job.image)

			// Use opm alpha bundle unpack to extract only operator manifests with retry logic
			maxRetries := 3

			// Create unique cache directory per bundle to prevent race conditions
			// Use both PID and bundle name to ensure complete isolation
			workerCacheDir := filepath.Join(job.cacheDir, "opm", fmt.Sprintf("worker-%d-%s", os.Getpid(), bundleName))
			// Clean any existing corrupted cache first
			os.RemoveAll(workerCacheDir)
			os.MkdirAll(workerCacheDir, 0755)

			for attempt := 1; attempt <= maxRetries; attempt++ {
				cmd := exec.Command("opm", "alpha", "bundle", "unpack", job.image, "-o", bundleDir)
				// Set isolated environment to prevent cache conflicts
				cmd.Env = append(os.Environ(),
					"OPM_CACHE_DIR="+workerCacheDir,
					"TMPDIR="+workerCacheDir, // Force temp files to isolated location
					"HOME="+workerCacheDir,   // Isolate any home-based caches
				)
				cmd.Dir = workerCacheDir // Set working directory to isolated cache

				if _, extractionErr = auditpkg.RunCommand(cmd); extractionErr == nil {
					break // Success, exit retry loop
				}

				if attempt < maxRetries {
					log.Warnf("opm unpack attempt %d failed for %s, retrying: %v", attempt, job.image, extractionErr)
					// More aggressive backoff for cache corruption issues
					sleepDuration := time.Duration(attempt*attempt) * time.Second // Quadratic backoff: 1s, 4s, 9s
					time.Sleep(sleepDuration)

					// Aggressively clean up all potential corruption
					os.RemoveAll(bundleDir)      // Clean target directory
					os.RemoveAll(workerCacheDir) // Clean worker cache

					// Wait a bit more and recreate clean cache
					time.Sleep(500 * time.Millisecond)
					os.MkdirAll(workerCacheDir, 0755) // Recreate clean cache
				}
			}

			if extractionErr != nil {
				log.Errorf("unable to unpack bundle %s using opm after %d attempts: %v", job.image, maxRetries, extractionErr)
				result.err = extractionErr
				results <- result
				continue
			}

			// Clean up opm cache after successful extraction to prevent accumulation
			if err := os.RemoveAll(workerCacheDir); err != nil {
				log.Errorf("could not remove cache dir, err: %v", err)
			}
		}

		// scan for NetworkPolicy across all text files
		filesScanned := 0
		binarySkipped := 0
		foundPaths := []string{}
		filepath.Walk(bundleDir, func(filePath string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			// read first chunk to detect binary
			f, err := os.Open(filePath)
			if err != nil {
				return nil
			}
			defer f.Close()
			buf := make([]byte, 8000)
			n, _ := f.Read(buf)
			data := buf[:n]
			// skip binary files
			if isBinary(filePath, data) {
				binarySkipped++
				return nil
			}
			filesScanned++
			// Check if this file contains actual NetworkPolicy resources (not just mentions)
			if hasNetworkPolicyResource(data, filePath) {
				rel, _ := filepath.Rel(bundleDir, filePath)
				foundPaths = append(foundPaths, rel)
				log.Infof("Found NetworkPolicy resource in bundle image %s", job.image)
			}
			return nil
		})

		result.filesScanned = filesScanned
		result.binarySkipped = binarySkipped
		result.foundPaths = foundPaths

		// Save successful result to cache for future runs
		if result.err == nil {
			if err := saveCachedResult(cacheFile, &result); err != nil {
				log.Warnf("Failed to cache result for %s: %v", bundleName, err)
			}
		}

		results <- result
	}
}

// isBinary reports whether a file is likely binary based on extension and content
func isBinary(filePath string, data []byte) bool {
	// Skip known binary file extensions immediately
	skipExtensions := []string{
		".so", ".dylib", ".exe", ".bin", ".jar", ".tar", ".gz", ".zip",
		".tgz", ".bz2", ".xz", ".rpm", ".deb", ".dmg", ".img", ".iso",
		".a", ".o", ".lib", ".dll", ".class", ".pyc", ".pyo",
		".png", ".jpg", ".jpeg", ".gif", ".bmp", ".svg", ".ico",
		".mp3", ".mp4", ".avi", ".mov", ".pdf", ".doc", ".docx",
		".xls", ".xlsx", ".ppt", ".pptx", ".odt", ".ods", ".odp",
	}

	for _, ext := range skipExtensions {
		if strings.HasSuffix(strings.ToLower(filePath), ext) {
			return true
		}
	}

	// Only check first 512 bytes for null bytes (fast approach)
	sampleSize := 512
	if len(data) < sampleSize {
		sampleSize = len(data)
	}

	for i := 0; i < sampleSize; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

// loadCachedResult loads a cached bundle result from disk
func loadCachedResult(cacheFile string) (*bundleResult, error) {
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil, err
	}

	var result bundleResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// saveCachedResult saves a bundle result to disk cache
func saveCachedResult(cacheFile string, result *bundleResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return os.WriteFile(cacheFile, data, 0644)
}

// hasNetworkPolicyResource checks if the file content contains actual NetworkPolicy resources
// rather than just mentions of "NetworkPolicy" in field names or documentation
func hasNetworkPolicyResource(data []byte, filePath string) bool {
	// Quick string check first - if no mention, definitely no NetworkPolicy
	if !strings.Contains(string(data), "NetworkPolicy") {
		return false
	}

	// Only check YAML/JSON files for actual Kubernetes resources
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".yaml" && ext != ".yml" && ext != ".json" {
		return false
	}

	// Parse as YAML (works for JSON too) and look for kind: NetworkPolicy
	if ext == ".json" {
		return hasNetworkPolicyInJSON(data)
	} else {
		return hasNetworkPolicyInYAML(data)
	}
}

// hasNetworkPolicyInYAML checks for NetworkPolicy resources in YAML content
func hasNetworkPolicyInYAML(data []byte) bool {
	// Split YAML documents (separated by ---)
	documents := bytes.Split(data, []byte("---"))

	for _, doc := range documents {
		doc = bytes.TrimSpace(doc)
		if len(doc) == 0 {
			continue
		}

		var resource struct {
			Kind string `yaml:"kind"`
		}

		if err := yaml.Unmarshal(doc, &resource); err != nil {
			continue // Skip malformed YAML
		}

		if resource.Kind == "NetworkPolicy" {
			return true
		}
	}

	return false
}

// hasNetworkPolicyInJSON checks for NetworkPolicy resources in JSON content
func hasNetworkPolicyInJSON(data []byte) bool {
	var resource struct {
		Kind string `json:"kind"`
	}

	if err := json.Unmarshal(data, &resource); err != nil {
		return false // Skip malformed JSON
	}

	return resource.Kind == "NetworkPolicy"
}
