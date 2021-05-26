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
	const testdataPath = "/testdata/samples/"

	samplesDir := filepath.Join(currentPath, testdataPath)
	binPath := filepath.Join(currentPath, "bin", "audit-tool")

	log.Infof("using the path: (%v)", samplesDir)
	log.Infof("using the bin: (%v)", binPath)

	command := exec.Command("rm", "-rf", filepath.Join(samplesDir))
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	command = exec.Command("mkdir", filepath.Join(samplesDir))
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	indexReportKinds := []string{"bundles", "channels", "packages"}
	indexImage := "registry.redhat.io/redhat/redhat-operator-index:v4.7"

	for _, v := range indexReportKinds {
		// create dir
		command = exec.Command("mkdir", filepath.Join(samplesDir, v))
		_, err = pkg.RunCommand(command)
		if err != nil {
			log.Errorf("running command :%s", err)
		}
		// run report
		command = exec.Command(binPath, "index", v,
			fmt.Sprintf("--index-image=%s", indexImage),
			"--limit=5",
			"--output=all",
			fmt.Sprintf("--output-path=%s", filepath.Join(samplesDir, v)),
		)
		_, err = pkg.RunCommand(command)
		if err != nil {
			log.Errorf("running command :%s", err)
		}
	}

	customPath := filepath.Join(samplesDir, "dashboard")
	command = exec.Command("mkdir", customPath)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	jsonBundlesReport := filepath.Join(filepath.Join(samplesDir, "bundles"),
		pkg.GetReportName(indexImage, "bundles", "json"))

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
