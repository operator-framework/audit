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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	goyaml "github.com/goccy/go-yaml"
	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/models"
	log "github.com/sirupsen/logrus"
)

const defaultSDKScorecardImageName = "quay.io/operator-framework/scorecard-test"
const scorecardAnnotation = "operators.operatorframework.io.test.config.v1"

type BundleAnnotations struct {
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

func RunScorecard(bundleDir string, auditBundle *models.AuditBundle) *models.AuditBundle {
	log.Info("\n----bundleDir----\n", bundleDir)
	scorecardTestsPath := filepath.Join(bundleDir, "tests", "scorecard")
	log.Info("\n----scorecardTestsPath----\n", scorecardTestsPath)
	annotationsPath := filepath.Join(bundleDir, "metadata", "annotations.yaml")
	log.Info("\n----annotationsPath----\n", annotationsPath)

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
			msg := fmt.Errorf("unable to Unmarshal annotations.yaml to check scorecard path: %s", err)
			log.Error(msg)
			auditBundle.Errors = append(auditBundle.Errors, msg.Error())
		}
		if len(bundleAnnotations.Annotations) > 0 {
			path := bundleAnnotations.Annotations[scorecardAnnotation]
			if len(path) > 0 {
				scorecardTestsPath = filepath.Join(bundleDir, path)
			}
		}
	}

	// Check if has scorecard manifests
	if _, err := os.Stat(scorecardTestsPath); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(scorecardTestsPath, os.ModePerm); err != nil {
				auditBundle.Errors = append(auditBundle.Errors,
					fmt.Errorf("unable to create scorecard dir test: %s", err).Error())
				return auditBundle
			}
		} else {
			auditBundle.Errors = append(auditBundle.Errors,
				fmt.Errorf("unexpected error to run scorecard: %s", err).Error())
			return auditBundle
		}
	} else {
		err := filepath.Walk(scorecardTestsPath, func(path string, info os.FileInfo, err error) error {
			if info != nil && !info.IsDir() && strings.HasSuffix(info.Name(), "yaml") {
				scorecardFilePath := filepath.Join(scorecardTestsPath, info.Name())
				if existingFile, err := ioutil.ReadFile(scorecardFilePath); err == nil {
					var scorecardConfig v1alpha3.Configuration
					if err := goyaml.Unmarshal(existingFile, &scorecardConfig); err != nil {
						msg := fmt.Errorf("unable to Unmarshal scorecard file %s: %s", info.Name(), err)
						log.Error(msg)
						auditBundle.Errors = append(auditBundle.Errors, msg.Error())
					}

					for _, k := range scorecardConfig.Stages {
						for _, t := range k.Tests {
							if !strings.Contains(t.Image, defaultSDKScorecardImageName) {
								auditBundle.HasCustomScorecardTests = true
								break
							}
						}
					}
				}
			}

			return nil
		})
		if err != nil {
			msg := fmt.Errorf("unable to walk in scorecard filse: %s", err)
			log.Error(msg)
			auditBundle.Errors = append(auditBundle.Errors, msg.Error())
		}
	}

	if err := writeScorecardConfig(scorecardTestsPath); err != nil {
		msg := fmt.Errorf("unable to write scorecard default tests: %s", err)
		log.Error(msg)
		auditBundle.Errors = append(auditBundle.Errors, msg.Error())
		return auditBundle
	}

	scorecardConfig := false
	scorecardFilePath := "github.com/operator-framework/audit/pkg/actions/scorecardDefaultConfigFragment.yaml"
	// Add Logic to update scorecardConfig

	// run scorecard against bundle
	cmd := exec.Command("operator-sdk", "scorecard", bundleDir, "--wait-time=120s", "--output=json", "--scorecard-config", scorecardFilePath, "--scorecard-config", scorecardConfig)
	output, _ := pkg.RunCommand(cmd)
	if len(output) < 1 {
		log.Errorf("unable to get scorecard output: %s", output)
		auditBundle.Errors = append(auditBundle.Errors,
			fmt.Errorf("unable to run scorecard: %s", errors.New("unable get scorecard output")).Error())
		return auditBundle
	}

	var scorecardResults v1alpha3.TestList
	err := json.Unmarshal(output, &scorecardResults)
	if err != nil {
		auditBundle.Errors = append(auditBundle.Errors,
			fmt.Errorf("unable to run scorecard: %s", err).Error())
		return auditBundle
	}
	auditBundle.ScorecardResults = scorecardResults
	return auditBundle
}

// writeScorecardConfig always the config file for audit
func writeScorecardConfig(scorecardConfigPath string) error {
	auditScorecardConfig := filepath.Join(scorecardConfigPath, "config.yaml")
	cmd := exec.Command("rm", "-rf", auditScorecardConfig)
	_, _ = pkg.RunCommand(cmd)

	f, err := os.Create(auditScorecardConfig)
	if err != nil {
		log.Error(err)
		return err
	}

	defer f.Close()

	_, err = f.WriteString(scorecardDefaultConfigFragment)
	if err != nil {
		return err
	}
	return nil
}

// const scorecardDefaultConfigFragment = `apiVersion: scorecard.operatorframework.io/v1alpha3
// kind: Configuration
// metadata:
//   name: config
// stages:
// - parallel: true
//   tests:
//   - entrypoint:
//     - scorecard-test
//     - basic-check-spec
//     image: quay.io/operator-framework/scorecard-test:v1.22.0
//     labels:
//       suite: basic
//       test: basic-check-spec-test
//   - entrypoint:
//     - scorecard-test
//     - olm-bundle-validation
//     image: quay.io/operator-framework/scorecard-test:v1.22.0
//     labels:
//       suite: olm
//       test: olm-bundle-validation-test
//   - entrypoint:
//     - scorecard-test
//     - olm-crds-have-validation
//     image: quay.io/operator-framework/scorecard-test:v1.22.0
//     labels:
//       suite: olm
//       test: olm-crds-have-validation-test
//   - entrypoint:
//     - scorecard-test
//     - olm-crds-have-resources
//     image: quay.io/operator-framework/scorecard-test:v1.22.0
//     labels:
//       suite: olm
//       test: olm-crds-have-resources-test
//   - entrypoint:
//     - scorecard-test
//     - olm-spec-descriptors
//     image: quay.io/operator-framework/scorecard-test:v1.22.0
//     labels:
//       suite: olm
//       test: olm-spec-descriptors-test
//   - entrypoint:
//     - scorecard-test
//     - olm-status-descriptors
//     image: quay.io/operator-framework/scorecard-test:v1.22.0
//     labels:
//       suite: olm
//       test: olm-status-descriptors-test`
