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

	// testdata is the path where all samples should be generate
	const testdataPath = "/testdata/"

	reportPath := filepath.Join(currentPath, testdataPath, "reports")
	binPath := filepath.Join(currentPath, "bin", "audit-tool")

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
	images := []string{
		"registry.redhat.io/redhat/certified-operator-index:v4.8",
		"registry.redhat.io/redhat/community-operator-index:v4.8",
		"registry.redhat.io/redhat/redhat-marketplace-index:v4.8",
		"registry.redhat.io/redhat/redhat-operator-index:v4.8",
		"quay.io/operatorhubio/catalog:latest",
	}

	indexReportKinds := []string{"bundles", "channels", "packages"}
	for _, v := range images {
		reportPathName := filepath.Join(reportPath, hack.GetImageNameToCreateDir(v))
		command := exec.Command("mkdir", reportPathName)
		_, err = pkg.RunCommand(command)
		if err != nil {
			log.Errorf("running command :%s", err)
		}

		for _, report := range indexReportKinds {
			// run report
			command := exec.Command(binPath, "index", report,
				fmt.Sprintf("--index-image=%s", v),
				"--output=all",
				fmt.Sprintf("--output-path=%s", reportPathName),
			)
			_, err = pkg.RunCommand(command)
			if err != nil {
				log.Errorf("running command :%s", err)
			}
		}

		customPath := filepath.Join(reportPathName, "dashboards")
		command = exec.Command("mkdir", customPath)
		_, err = pkg.RunCommand(command)
		if err != nil {
			log.Errorf("running command :%s", err)
		}

		jsonBundlesReport := filepath.Join(filepath.Join(reportPathName,
			pkg.GetReportName(v, "bundles", "json")))

		// run report
		command = exec.Command(binPath, "dashboard", "deprecate-apis",
			fmt.Sprintf("--file=%s", jsonBundlesReport),
			fmt.Sprintf("--output-path=%s", customPath),
		)
		_, err = pkg.RunCommand(command)
		if err != nil {
			log.Errorf("running command :%s", err)
		}
	}

	// Gen only bundles for the previous ones >= 4.6+ for we have the deprecated API(s) dashs
	images = []string{
		"registry.redhat.io/redhat/certified-operator-index:v4.7",
		"registry.redhat.io/redhat/community-operator-index:v4.7",
		"registry.redhat.io/redhat/redhat-marketplace-index:v4.7",
		"registry.redhat.io/redhat/redhat-operator-index:v4.7",
		"registry.redhat.io/redhat/certified-operator-index:v4.6",
		"registry.redhat.io/redhat/community-operator-index:v4.6",
		"registry.redhat.io/redhat/redhat-marketplace-index:v4.6",
		"registry.redhat.io/redhat/redhat-operator-index:v4.6",
	}

	indexReportKinds = []string{"bundles"}
	for _, v := range images {
		reportPathName := filepath.Join(reportPath, hack.GetImageNameToCreateDir(v))
		for _, report := range indexReportKinds {
			// run report
			command := exec.Command(binPath, "index", report,
				fmt.Sprintf("--index-image=%s", v),
				"--output=all",
				fmt.Sprintf("--output-path=%s", reportPathName),
			)
			_, err = pkg.RunCommand(command)
			if err != nil {
				log.Errorf("running command :%s", err)
			}
		}

		customPath := filepath.Join(reportPathName, "dashboards")
		jsonBundlesReport := filepath.Join(filepath.Join(reportPathName,
			pkg.GetReportName(v, "bundles", "json")))

		// run report
		command := exec.Command(binPath, "dashboard", "deprecate-apis",
			fmt.Sprintf("--file=%s", jsonBundlesReport),
			fmt.Sprintf("--output-path=%s", customPath),
		)
		_, err = pkg.RunCommand(command)
		if err != nil {
			log.Errorf("running command :%s", err)
		}
	}

}
