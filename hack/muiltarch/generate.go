// Copyright 2021 The Audit Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this File except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This module is used to generate the Deprecated API(s) custom dashboards
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/operator-framework/audit/hack"
	"github.com/operator-framework/audit/pkg"
	log "github.com/sirupsen/logrus"
)

//nolint:gocyclo
func main() {

	currentPath, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	fullReportsPath := filepath.Join(currentPath, hack.ReportsPath)

	dirs := map[string]string{
		"redhat_certified_operator_index": "registry.redhat.io/redhat/certified-operator-index",
		"redhat_community_operator_index": "registry.redhat.io/redhat/community-operator-index",
		"redhat_redhat_marketplace_index": "registry.redhat.io/redhat/redhat-marketplace-index",
		"redhat_redhat_operator_index":    "registry.redhat.io/redhat/redhat-operator-index",
	}

	binPath := filepath.Join(currentPath, hack.BinPath)

	// nolint:scopelint
	for dir := range dirs {
		pathToWalk := filepath.Join(fullReportsPath, dir)
		dashboardPath := filepath.Join(fullReportsPath, dir, "dashboards")

		command := exec.Command("mkdir", dashboardPath)
		_, err = pkg.RunCommand(command)
		if err != nil {
			log.Warnf("running command :%s", err)
		}

		if _, err := os.Stat(pathToWalk); err != nil && os.IsNotExist(err) {
			continue
		}

		// Walk in the testdata dir and generates the deprecated-api custom dashboard for
		// all bundles JSON reports available there
		err := filepath.Walk(pathToWalk, func(path string, info os.FileInfo, err error) error {
			if info != nil && !info.IsDir() && strings.HasPrefix(info.Name(), "bundles") &&
				strings.HasSuffix(info.Name(), "json") {

				// Ignore the tag images 4.6 and 4.7
				if strings.Contains(info.Name(), "v4.7") ||
					strings.Contains(info.Name(), "v4.6") ||
					strings.Contains(info.Name(), "v4.8") {
					return nil
				}

				// run report
				command := exec.Command(binPath, "dashboard", "multiarch",
					fmt.Sprintf("--file=%s", path),
					fmt.Sprintf("--output-path=%s", dashboardPath),
				)
				if _, errC := pkg.RunCommand(command); errC != nil {
					log.Errorf("running command :%s", errC)
				}
			}
			return nil
		})
		if err != nil {
			log.Fatal(err)
		}
	}
}
