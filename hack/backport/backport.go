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
	"strings"

	"path/filepath"

	"github.com/operator-framework/audit/pkg"
	log "github.com/sirupsen/logrus"
)

func main() {

	command := exec.Command("make", "install")
	_, err := pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	// testdata is the path where all samples should be generate
	const testdataPath = "/testdata/"

	reportPath := filepath.Join(currentPath, testdataPath, "backport")
	binPath := filepath.Join(currentPath, "bin", "audit")

	command = exec.Command("rm", "-rf", reportPath)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	command = exec.Command("mkdir", reportPath)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	allimages := []string{
		"registry.redhat.io/redhat/redhat-operator-index:v4.5",
		"registry.redhat.io/redhat/certified-operator-index:v4.5",
	}

	command = exec.Command("docker", "login", "https://registry.connect.redhat.com")
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	command = exec.Command("docker", "login", "https://registry.redhat.io")
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	for _, v := range allimages {

		// create dir name with the image name only
		name := strings.Split(v, ":")[0]
		name = strings.ReplaceAll(name, "registry.redhat.io/redhat/", "redhat_")
		name = strings.ReplaceAll(name, "/", "_")
		name = strings.ReplaceAll(name, ":", "_")
		name = strings.ReplaceAll(name, "-", "_")

		reportPathName := filepath.Join(reportPath, name)
		command = exec.Command("mkdir", reportPathName)
		_, err = pkg.RunCommand(command)
		if err != nil {
			log.Errorf("running command :%s", err)
		}

		log.Infof("creating report bundles with XLS format for %s", v)
		command = exec.Command(binPath, "bundles",
			fmt.Sprintf("--index-image=%s", v),
			fmt.Sprintf("--output-path=%s", reportPathName),
			"--label=com.redhat.delivery.backport",
			"--label-value=true",
		)

		_, err = pkg.RunCommand(command)
		if err != nil {
			log.Errorf("running command :%s", err)
		}

		log.Infof("creating report packages with XLS format for %s", v)
		command = exec.Command(binPath, "packages",
			fmt.Sprintf("--index-image=%s", v),
			fmt.Sprintf("--output-path=%s", reportPathName),
			"--label=com.redhat.delivery.backport",
			"--label-value=true",
		)
		_, err = pkg.RunCommand(command)
		if err != nil {
			log.Errorf("running command :%s", err)
		}
	}
}
