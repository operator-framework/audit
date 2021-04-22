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
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/operator-framework/api/pkg/apis/scorecard/v1alpha3"
	"github.com/operator-framework/audit/pkg"
	log "github.com/sirupsen/logrus"
)

func RunScorecard(bundleDir string) (v1alpha3.TestList, error) {
	scorecardConfigPath := filepath.Join(bundleDir, "tests", "scorecard")
	// Check if has scorecard manifests
	if _, err := os.Stat(scorecardConfigPath); os.IsNotExist(err) {
		// Write scorecard config when that does not exist
		if err := writeScorecardConfig(scorecardConfigPath); err != nil {
			return v1alpha3.TestList{}, err
		}
	}

	// run scorecard against bundle
	cmd := exec.Command("operator-sdk", "scorecard", bundleDir, "--wait-time=120s", "--output=json")
	output, _ := pkg.RunCommand(cmd)
	if len(output) < 1 {
		log.Errorf("unable to get scorecard output: %s", output)
		return v1alpha3.TestList{}, errors.New("unable get scorecard output")
	}

	var scorecardResults v1alpha3.TestList
	err := json.Unmarshal(output, &scorecardResults)
	if err != nil {
		return v1alpha3.TestList{}, err
	}

	return scorecardResults, nil
}

func writeScorecardConfig(scorecardConfigPath string) error {
	if err := os.MkdirAll(scorecardConfigPath, os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(scorecardConfigPath, "config.yaml"))
	if err != nil {
		return err
	}

	defer f.Close()

	_, err = f.WriteString(scorecardConfigFragment)
	if err != nil {
		return err
	}
	return nil
}

const scorecardConfigFragment = `apiVersion: scorecard.operatorframework.io/v1alpha3
kind: Configuration
metadata:
  name: config
stages:
- parallel: true
  tests:
  - entrypoint:
    - scorecard-test
    - basic-check-spec
    image: quay.io/operator-framework/scorecard-test:v1.5.0
    labels:
      suite: basic
      test: basic-check-spec-test
  - entrypoint:
    - scorecard-test
    - olm-bundle-validation
    image: quay.io/operator-framework/scorecard-test:v1.5.0
    labels:
      suite: olm
      test: olm-bundle-validation-test
  - entrypoint:
    - scorecard-test
    - olm-crds-have-validation
    image: quay.io/operator-framework/scorecard-test:v1.5.0
    labels:
      suite: olm
      test: olm-crds-have-validation-test
  - entrypoint:
    - scorecard-test
    - olm-crds-have-resources
    image: quay.io/operator-framework/scorecard-test:v1.5.0
    labels:
      suite: olm
      test: olm-crds-have-resources-test
  - entrypoint:
    - scorecard-test
    - olm-spec-descriptors
    image: quay.io/operator-framework/scorecard-test:v1.5.0
    labels:
      suite: olm
      test: olm-spec-descriptors-test
  - entrypoint:
    - scorecard-test
    - olm-status-descriptors
    image: quay.io/operator-framework/scorecard-test:v1.5.0
    labels:
      suite: olm
      test: olm-status-descriptors-test`
