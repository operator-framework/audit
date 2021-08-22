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

// This module generate the bundles reports
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/operator-framework/audit/hack"

	"github.com/operator-framework/audit/pkg"
	log "github.com/sirupsen/logrus"
)

func main() {
	currentPath, err := os.Getwd()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	reportPath := filepath.Join(currentPath, hack.ReportsPath)
	binPath := filepath.Join(currentPath, hack.BinPath)

	command := exec.Command("rm", "-rf", reportPath)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	command = exec.Command("mkdir", reportPath)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	// Gen all Kinds for the latest
	images := map[string]string{
		"registry.redhat.io/redhat/certified-operator-index:v4.9": "https://registry.redhat.io",
		//TODO: Add when created
		//"registry.redhat.io/redhat/community-operator-index:v4.9": "https://registry.redhat.io",
		"registry.redhat.io/redhat/redhat-marketplace-index:v4.9": "https://registry.redhat.io",
		"registry.redhat.io/redhat/redhat-operator-index:v4.9":    "https://registry.redhat.io",
		"registry.redhat.io/redhat/certified-operator-index:v4.8": "https://registry.redhat.io",
		"registry.redhat.io/redhat/community-operator-index:v4.8": "https://registry.redhat.io",
		"registry.redhat.io/redhat/redhat-marketplace-index:v4.8": "https://registry.redhat.io",
		"registry.redhat.io/redhat/redhat-operator-index:v4.8":    "https://registry.redhat.io",
		"quay.io/operatorhubio/catalog:latest":                    "https://registry.connect.redhat.com",
		"registry.redhat.io/redhat/certified-operator-index:v4.7": "https://registry.redhat.io",
		"registry.redhat.io/redhat/community-operator-index:v4.7": "https://registry.redhat.io",
		"registry.redhat.io/redhat/redhat-marketplace-index:v4.7": "https://registry.redhat.io",
		"registry.redhat.io/redhat/redhat-operator-index:v4.7":    "https://registry.redhat.io",
		"registry.redhat.io/redhat/certified-operator-index:v4.6": "https://registry.redhat.io",
		"registry.redhat.io/redhat/community-operator-index:v4.6": "https://registry.redhat.io",
		"registry.redhat.io/redhat/redhat-marketplace-index:v4.6": "https://registry.redhat.io",
		"registry.redhat.io/redhat/redhat-operator-index:v4.6":    "https://registry.redhat.io",
	}

	indexReportKinds := []string{"bundles"}
	for image, registry := range images {
		reportPathName := filepath.Join(reportPath, hack.GetImageNameToCreateDir(image))
		command := exec.Command("mkdir", reportPathName)
		_, err = pkg.RunCommand(command)
		if err != nil {
			log.Warnf("running command :%s", err)
		}

		command = exec.Command("docker", "login", registry)
		_, err = pkg.RunCommand(command)
		if err != nil {
			log.Errorf("running command :%s", err)
		}

		for _, report := range indexReportKinds {
			// run report
			command := exec.Command(binPath, "index", report,
				fmt.Sprintf("--index-image=%s", image),
				"--output=all",
				fmt.Sprintf("--output-path=%s", reportPathName),
			)
			_, err = pkg.RunCommand(command)
			if err != nil {
				log.Errorf("running command :%s", err)
			}
		}
	}
}
