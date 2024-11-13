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

package actions

import (
	"encoding/json"
	"errors" //nolint: typecheck
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	// "strings"

	goyaml "github.com/goccy/go-yaml"
	log "github.com/sirupsen/logrus"

	apimanifests "github.com/operator-framework/api/pkg/manifests"
	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/models"
)

// Manifest define the manifest.json which is  required to read the bundle
type Manifest struct {
	Config string
	Layers []string
}

// GetDataFromBundleImage returns the bundle from the image
func GetDataFromBundleImage(auditBundle *models.AuditBundle,
	disableScorecard,
	disableValidators,
	serverMode bool,
	label,
	labelValue string,
	containerEngine string,
	indexImage string) *models.AuditBundle {

	if len(auditBundle.OperatorBundleImagePath) < 1 {
		log.Errorf("not found bundle path stored in the index.db")
		auditBundle.Errors = append(auditBundle.Errors,
			errors.New("not found bundle path stored in the index.db").Error())
		return auditBundle
	}

	err := DownloadImage(auditBundle.OperatorBundleImagePath, containerEngine)
	if err != nil {
		log.Errorf("unable to download container image (%s): %s", auditBundle.OperatorBundleImagePath, err)
		auditBundle.Errors = append(auditBundle.Errors,
			fmt.Errorf("unable to download container image (%s): %s", auditBundle.OperatorBundleImagePath, err).Error())
		return auditBundle
	}

	bundleDir := createBundleDir(auditBundle)
	extractBundleFromImage(auditBundle, bundleDir, containerEngine)

	inspectManifest, err := pkg.RunDockerInspect(auditBundle.OperatorBundleImagePath, containerEngine)
	if err != nil {
		log.Errorf("unable to inspace: %s", err)
		auditBundle.Errors = append(auditBundle.Errors, err.Error())
	} else {
		// Gathering data by inspecting the operator bundle image
		if len(label) > 0 {
			value := inspectManifest.DockerConfig.Labels[label]
			if value == labelValue {
				auditBundle.FoundLabel = true
			}
		}
		auditBundle.BundleImageLabels = inspectManifest.DockerConfig.Labels
	}

	// Ensure the image path has the 'docker://' prefix
	formattedImagePath := auditBundle.OperatorBundleImagePath
	if !strings.HasPrefix(formattedImagePath, "docker://") {
		formattedImagePath = "docker://" + formattedImagePath
	}

	dockerfiles, err := pkg.RunSkopeoLayerExtract(formattedImagePath)
	if err != nil {
		log.Printf("Error extracting Dockerfiles: %s", err)
		// Handle the error, e.g., by returning or continuing with other logic
	} else {
		// Store the extracted Dockerfiles in the auditBundle
		auditBundle.BundleDockerfiles = dockerfiles
	}

	// Read the bundle
	auditBundle.Bundle, err = apimanifests.GetBundleFromDir(filepath.Join(bundleDir, "bundle"))
	if err != nil {
		log.Errorf("unable to load bundle: %s", err)
		auditBundle.Errors = append(auditBundle.Errors, fmt.Errorf("unable to get the bundle: %s", err).Error())
		return auditBundle
	}

	annotationsPath := filepath.Join(bundleDir, "bundle/metadata/annotations.yaml")

	// If find the annotations file then, check for the scorecard path on it.
	if _, err := os.Stat(annotationsPath); err == nil && !os.IsNotExist(err) {
		annFile, err := pkg.ReadFile(annotationsPath)
		if err != nil {
			msg := fmt.Errorf("unable to read annotations.yaml to check scorecard path: %s", err)
			log.Error(msg)
			auditBundle.Errors = append(auditBundle.Errors, msg.Error())
		}
		var bundleAnnotations BundleAnnotations

		if err := goyaml.Unmarshal(annFile, &bundleAnnotations); err != nil {
			msg := fmt.Errorf("unable to Unmarshal annotations.yaml to check ocp label path: %s", err)
			log.Error(msg)
			auditBundle.Errors = append(auditBundle.Errors, msg.Error())
		}
		if len(bundleAnnotations.Annotations) > 0 {
			auditBundle.BundleAnnotations = bundleAnnotations.Annotations
		}
	}

	// Gathering data from scorecard
	if !disableScorecard {
		auditBundle = RunScorecard(filepath.Join(bundleDir, "bundle"), auditBundle)
	}

	// Run validators
	if !disableValidators {
		auditBundle = RunValidators(filepath.Join(bundleDir, "bundle"), auditBundle, indexImage)

	}

	cleanupBundleDir(auditBundle, bundleDir, serverMode, containerEngine)

	return auditBundle
}

func createBundleDir(auditBundle *models.AuditBundle) string {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	dir := fmt.Sprintf("%s/tmp/%s", currentPath, auditBundle.OperatorBundleName)
	cmd := exec.Command("mkdir", dir)
	_, err = pkg.RunCommand(cmd)
	if err != nil {
		log.Error(err)
		auditBundle.Errors = append(auditBundle.Errors,
			fmt.Errorf("unable to create the dir for the bundle: %s", err).Error())
	}
	return dir
}

func extractBundleFromImage(auditBundle *models.AuditBundle, bundleDir string, containerEngine string) {
	// imageName := strings.Split(auditBundle.OperatorBundleImagePath, "@")[0]
	imageName := auditBundle.OperatorBundleImagePath
	tarPath := fmt.Sprintf("%s/%s.tar", bundleDir, auditBundle.OperatorBundleName)
	cmd := exec.Command(containerEngine, "save", imageName, "-o", tarPath)
	_, err := pkg.RunCommand(cmd)
	if err != nil {
		log.Errorf("unable to save the bundle image : %s", err)
		auditBundle.Errors = append(auditBundle.Errors,
			fmt.Errorf("unable to save the bundle image : %s", err).Error())
	}

	cmd = exec.Command("tar", "-xvf", tarPath, "-C", bundleDir)
	_, err = pkg.RunCommand(cmd)
	if err != nil {
		log.Errorf("unable to untar the bundle image: %s", err)
		auditBundle.Errors = append(auditBundle.Errors,
			fmt.Errorf("unable to untar the bundle image : %s", err).Error())
	}

	cmd = exec.Command("mkdir", filepath.Join(bundleDir, "bundle"))
	_, err = pkg.RunCommand(cmd)
	if err != nil {
		log.Errorf("error to create the bundle bundleDir: %s", err)
		auditBundle.Errors = append(auditBundle.Errors,
			fmt.Errorf("error to create the bundle bundleDir : %s", err).Error())
	}

	bundleConfigFilePath := filepath.Join(bundleDir, "manifest.json")
	existingFile, err := os.ReadFile(bundleConfigFilePath)
	if err == nil {
		var bundleLayerConfig []Manifest
		if err := json.Unmarshal(existingFile, &bundleLayerConfig); err != nil {
			log.Errorf("unable to Unmarshal manifest.json: %s", err)
			auditBundle.Errors = append(auditBundle.Errors,
				fmt.Errorf("unable to Unmarshal manifest.json: %s", err).Error())
		}
		if bundleLayerConfig == nil {
			log.Errorf("error to untar layers")
			auditBundle.Errors = append(auditBundle.Errors,
				fmt.Errorf("error to untar layers: %s", err).Error())
		}

		for _, layer := range bundleLayerConfig[0].Layers {
			cmd = exec.Command("tar", "-xvf", filepath.Join(bundleDir, layer), "-C", filepath.Join(bundleDir, "bundle"))
			_, err = pkg.RunCommand(cmd)
			if err != nil {
				log.Errorf("unable to untar layer : %s", err)
				auditBundle.Errors = append(auditBundle.Errors,
					fmt.Errorf("error to untar layers : %s", err).Error())
			}
		}
	} else {
		// If the docker manifest was not found then check if has just one layer
		cmd = exec.Command("tar", "-xvf", fmt.Sprintf("%s/layer.tar", bundleDir), "-C", filepath.Join(bundleDir, "bundle"))
		_, err = pkg.RunCommand(cmd)
		if err != nil {
			log.Errorf("unable to untar layer : %s", err)
			auditBundle.Errors = append(auditBundle.Errors,
				fmt.Errorf("unable to untar layer: %s", err).Error())
		}
	}

	// Remove files in the image to allow load the bundle
	cmd = exec.Command("rm", "-rf", fmt.Sprintf("%s/bundle/manifests/.wh..wh..opq", bundleDir))
	_, _ = pkg.RunCommand(cmd)

	cmd = exec.Command("rm", "-rf", fmt.Sprintf("%s/bundle/metadata/.wh..wh..opq", bundleDir))
	_, _ = pkg.RunCommand(cmd)

	cmd = exec.Command("rm", "-rf", fmt.Sprintf("%s/bundle/root/", bundleDir))
	_, _ = pkg.RunCommand(cmd)

	cmd = exec.Command("rm", "-rf", fmt.Sprintf("%s/bundle/manifests/.DS_Store", bundleDir))
	_, _ = pkg.RunCommand(cmd)
}

func cleanupBundleDir(auditBundle *models.AuditBundle, dir string, serverMode bool, containerEngine string) {
	cmd := exec.Command("rm", "-rf", dir)
	_, _ = pkg.RunCommand(cmd)

	if !serverMode {
		cmd = exec.Command(containerEngine, "rmi", auditBundle.OperatorBundleImagePath)
		_, _ = pkg.RunCommand(cmd)
	}
}

func DownloadImage(image string, containerEngine string) error {
	log.Infof("Downloading image %s to audit...", image)
	cmd := exec.Command(containerEngine, "pull", image)
	_, err := pkg.RunCommand(cmd)
	// if found an error try again
	// Sometimes it faces issues to download the image
	if err != nil {
		log.Warnf("%s failed to downlad the image. Let's try more one time.", err)
		cmd := exec.Command(containerEngine, "pull", image)
		_, err = pkg.RunCommand(cmd)
	}
	return err
}
