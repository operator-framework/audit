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

package np

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	_ "github.com/mattn/go-sqlite3"
	"github.com/operator-framework/audit/cmd/index/bundles"
	auditpkg "github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/actions"
	"github.com/operator-framework/operator-registry/alpha/declcfg"
	"github.com/operator-framework/operator-registry/alpha/model"
)

// flags holds the command-line flags for the np command
var flags struct {
	Indexes         []string
	Package         string
	ContainerEngine string
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

	// container engine (docker or podman)
	cmd.Flags().StringVar(&flags.ContainerEngine, "container-engine", auditpkg.GetContainerToolFromEnvVar(),
		fmt.Sprintf("Container tool to use (options: %s, %s)", auditpkg.Docker, auditpkg.Podman))

	return cmd
}

// validation verifies required flags and flag values
func validation(cmd *cobra.Command, args []string) error {
	// require at least one index
	if len(flags.Indexes) == 0 {
		return fmt.Errorf("invalid value for --indexes: at least one index must be specified")
	}
	// validate container engine
	if flags.ContainerEngine == "" {
		flags.ContainerEngine = auditpkg.GetContainerToolFromEnvVar()
	}
	if flags.ContainerEngine != auditpkg.Docker && flags.ContainerEngine != auditpkg.Podman {
		return fmt.Errorf("invalid value for --container-engine: %s; valid options are %s or %s", flags.ContainerEngine, auditpkg.Docker, auditpkg.Podman)
	}
	return nil
}

// run executes the np audit logic
func run(cmd *cobra.Command, args []string) error {
	log.Info("Starting NetworkPolicy audit...")
	// create report file
	reportName := fmt.Sprintf("np_report_%s.txt", time.Now().Format("20060102_150405"))
	reportFile, err := os.Create(reportName)
	if err != nil {
		return fmt.Errorf("unable to create report file %s: %v", reportName, err)
	}
	defer reportFile.Close()
	auditpkg.GenerateTemporaryDirs()
	// load models or databases for each index
	modelOrDBs := getModelsOrDB(flags.Indexes)
	for idx, modelOrDB := range modelOrDBs {
		index := flags.Indexes[idx]
		log.Infof("Preparing Data for NetworkPolicy audit for index %s...", index)
		// write index header
		reportFile.WriteString(fmt.Sprintf("%s\n", index))
		// get package names
		pkgs, err := getPackageNames(modelOrDB)
		if err != nil {
			log.Errorf("unable to list packages for index %s: %v", index, err)
			continue
		}
		for _, pkgName := range pkgs {
			// write package header
			reportFile.WriteString(fmt.Sprintf("    %s\n", pkgName))
			if flags.Package != "" && pkgName != flags.Package {
				continue
			}
			// list bundles for the package
			bundlesList, err := getBundleNames(modelOrDB, pkgName)
			if err != nil {
				log.Errorf("unable to list bundles for package %s: %v", pkgName, err)
				continue
			}
			for _, bundleName := range bundlesList {
				// find image reference
				img, err := getBundleImagePath(modelOrDB, pkgName, bundleName)
				if err != nil {
					log.Errorf("unable to find image for bundle %s: %v", bundleName, err)
					continue
				}
				// download bundle image
				log.Infof("Downloading bundle image %s", img)
				if err := actions.DownloadImage(img, flags.ContainerEngine); err != nil {
					log.Errorf("unable to download image %s: %v", img, err)
					continue
				}
				// extract bundle tar
				bundleDir := filepath.Join("tmp", bundleName)
				if err := os.MkdirAll(bundleDir, 0755); err != nil {
					log.Errorf("unable to create tmp dir %s: %v", bundleDir, err)
					continue
				}
				tarPath := filepath.Join(bundleDir, bundleName+".tar")
				if _, err := auditpkg.RunCommand(exec.Command(flags.ContainerEngine, "save", img, "-o", tarPath)); err != nil {
					log.Errorf("unable to save bundle image %s: %v", img, err)
					cleanupBundle(bundleDir, img)
					continue
				}
				if _, err := auditpkg.RunCommand(exec.Command("tar", "-xvf", tarPath, "-C", bundleDir)); err != nil {
					log.Errorf("unable to untar bundle image %s: %v", img, err)
					cleanupBundle(bundleDir, img)
					continue
				}
				// read manifest.json to get layers
				manifestFile := filepath.Join(bundleDir, "manifest.json")
				mf, err := os.ReadFile(manifestFile)
				if err != nil {
					log.Errorf("unable to read manifest.json for bundle %s: %v", bundleName, err)
					cleanupBundle(bundleDir, img)
					continue
				}
				var manifest []struct{ Layers []string }
				if err := json.Unmarshal(mf, &manifest); err != nil {
					log.Errorf("unable to parse manifest.json for bundle %s: %v", bundleName, err)
					cleanupBundle(bundleDir, img)
					continue
				}
				// extract layers into bundleDir/bundle
				bundleRoot := filepath.Join(bundleDir, "bundle")
				_ = os.MkdirAll(bundleRoot, 0755)
				for _, layer := range manifest[0].Layers {
					layerPath := filepath.Join(bundleDir, layer)
					if _, err := auditpkg.RunCommand(exec.Command("tar", "-xvf", layerPath, "-C", bundleRoot)); err != nil {
						log.Warnf("unable to untar layer %s: %v", layer, err)
					}
				}
				// scan for NetworkPolicy across all text files
				filesScanned := 0
				binarySkipped := 0
				foundPaths := []string{}
				filepath.Walk(bundleRoot, func(filePath string, info os.FileInfo, err error) error {
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
					if isBinary(data) {
						binarySkipped++
						return nil
					}
					filesScanned++
					// list each file when filtering by package
					relPath, _ := filepath.Rel(bundleRoot, filePath)
					if flags.Package != "" {
						reportFile.WriteString(fmt.Sprintf("            %s\n", relPath))
					}
					// search for keyword
					if strings.Contains(string(data), "NetworkPolicy") {
						rel, _ := filepath.Rel(bundleRoot, filePath)
						foundPaths = append(foundPaths, rel)
						log.Infof("Found NetworkPolicy resource in bundle %s of package %s", bundleName, pkgName)
					}
					return nil
				})
				// write report entries, including skipped binary count
				reportFile.WriteString(fmt.Sprintf("        %s: %d files scanned, skipped %d binary files\n", bundleName, filesScanned, binarySkipped))
				for _, rel := range foundPaths {
					reportFile.WriteString(fmt.Sprintf("            Found NetworkPolicy resource in bundle %s of package %s: %s\n", bundleName, pkgName, rel))
				}
				// cleanup extracted bundle and remove image
				cleanupBundle(bundleDir, img)
			}
		}
		// remove temporary index container and image
		_, _ = auditpkg.RunCommand(exec.Command(flags.ContainerEngine, "rm", actions.CatalogIndex))
		_, _ = auditpkg.RunCommand(exec.Command(flags.ContainerEngine, "rmi", index))
	}
	auditpkg.CleanupTemporaryDirs()
	log.Info("Operation completed.")
	return nil
}

// getModelsOrDB extracts each index and loads either a file-based catalog or sqlite DB
func getModelsOrDB(indexes []string) []interface{} {
	var modelsOrDBs []interface{}
	for _, index := range indexes {
		if err := actions.ExtractIndexDBorCatalogs(index, flags.ContainerEngine); err != nil {
			log.Errorf("error extracting index %s: %v", index, err)
			return modelsOrDBs
		}
		log.Infof("Preparing data for index %s...", index)
		var db *sql.DB
		var modelData model.Model
		var err error
		if bundles.IsFBC(index) {
			root := filepath.Join("./output", actions.GetVersionTagFromImage(index), "configs")
			fs := os.DirFS(root)
			fbc, err := declcfg.LoadFS(context.Background(), fs)
			if err != nil {
				log.Errorf("unable to load file-based catalog for index %s: %v", index, err)
				return modelsOrDBs
			}
			modelData, _ = declcfg.ConvertToModel(*fbc)
		} else {
			db, err = sql.Open("sqlite3", filepath.Join("./output", actions.GetVersionTagFromImage(index), "index.db"))
			if err != nil {
				log.Errorf("unable to open index.db for index %s: %v", index, err)
				return modelsOrDBs
			}
		}
		if modelData != nil {
			modelsOrDBs = append(modelsOrDBs, modelData)
		} else {
			modelsOrDBs = append(modelsOrDBs, db)
		}
	}
	return modelsOrDBs
}

// getPackageNames lists packages in the model or sqlite DB
func getPackageNames(modelOrDB interface{}) ([]string, error) {
	var packages []string
	switch m := modelOrDB.(type) {
	case *sql.DB:
		rows, err := m.Query("SELECT name FROM package")
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var pkgName string
			if err := rows.Scan(&pkgName); err == nil {
				packages = append(packages, pkgName)
			}
		}
		return uniqueStrings(packages), nil
	case model.Model:
		for pkgName := range m {
			packages = append(packages, pkgName)
		}
		return uniqueStrings(packages), nil
	default:
		return nil, fmt.Errorf("unsupported model type %T", modelOrDB)
	}
}

// getBundleNames lists all bundles for a given package
func getBundleNames(modelOrDB interface{}, pkgName string) ([]string, error) {
	var bundlesList []string
	switch m := modelOrDB.(type) {
	case *sql.DB:
		query := `SELECT o.name FROM operatorbundle o JOIN channel_entry c ON o.name=c.operatorbundle_name WHERE c.package_name = ? GROUP BY o.name`
		rows, err := m.Query(query, pkgName)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var b string
			if err := rows.Scan(&b); err == nil {
				bundlesList = append(bundlesList, b)
			}
		}
		return uniqueStrings(bundlesList), nil
	case model.Model:
		if pkgModel, exists := m[pkgName]; exists {
			set := make(map[string]struct{})
			for _, ch := range pkgModel.Channels {
				for _, b := range ch.Bundles {
					set[b.Name] = struct{}{}
				}
			}
			for name := range set {
				bundlesList = append(bundlesList, name)
			}
		}
		return uniqueStrings(bundlesList), nil
	default:
		return nil, fmt.Errorf("unsupported model type %T", modelOrDB)
	}
}

// getBundleImagePath returns the image reference for a bundle
func getBundleImagePath(modelOrDB interface{}, pkgName, bundleName string) (string, error) {
	switch m := modelOrDB.(type) {
	case *sql.DB:
		var path string
		err := m.QueryRow("SELECT bundlepath FROM operatorbundle WHERE name = ?", bundleName).Scan(&path)
		return path, err
	case model.Model:
		if pkgModel, exists := m[pkgName]; exists {
			for _, ch := range pkgModel.Channels {
				for _, b := range ch.Bundles {
					if b.Name == bundleName {
						return b.Image, nil
					}
				}
			}
		}
		return "", fmt.Errorf("bundle %s not found for package %s", bundleName, pkgName)
	default:
		return "", fmt.Errorf("unsupported model type %T", modelOrDB)
	}
}

// isBinary reports whether data contains a null byte, indicating a binary file
func isBinary(data []byte) bool {
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return false
}

// cleanupBundle removes the extracted bundle dir and the image
func cleanupBundle(dir, image string) {
	_ = os.RemoveAll(dir)
	_ = exec.Command(flags.ContainerEngine, "rmi", image).Run()
}

// uniqueStrings returns a deduplicated, sorted list
func uniqueStrings(slice []string) []string {
	set := make(map[string]struct{})
	for _, s := range slice {
		set[s] = struct{}{}
	}
	var list []string
	for s := range set {
		list = append(list, s)
	}
	sort.Strings(list)
	return list
}
