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

	command = exec.Command("mkdir", filepath.Join(samplesDir, "bundles"))
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	command = exec.Command("mkdir", filepath.Join(samplesDir, "bundles", "xls"))
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	command = exec.Command("mkdir", filepath.Join(samplesDir, "bundles", "json"))
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	command = exec.Command("mkdir", filepath.Join(samplesDir, "channels"))
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	command = exec.Command("mkdir", filepath.Join(samplesDir, "channels", "xls"))
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	command = exec.Command("mkdir", filepath.Join(samplesDir, "channels", "json"))
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	command = exec.Command("mkdir", filepath.Join(samplesDir, "packages"))
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	command = exec.Command("mkdir", filepath.Join(samplesDir, "packages", "xls"))
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	command = exec.Command("mkdir", filepath.Join(samplesDir, "packages", "json"))
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	log.Infof("creating bundles testdata Sample XLS")
	command = exec.Command(binPath, "bundles",
		"--index-image=registry.redhat.io/redhat/certified-operator-index:v4.8",
		"--limit=5",
		"--head-only",
		fmt.Sprintf("--output-path=%s", filepath.Join(samplesDir, "bundles", "xls")),
	)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	log.Infof("creating bundles testdata Sample json")
	command = exec.Command(binPath, "bundles",
		"--index-image=registry.redhat.io/redhat/certified-operator-index:v4.8",
		"--limit=2",
		"--output=json",
		"--head-only",
		fmt.Sprintf("--output-path=%s", filepath.Join(samplesDir, "bundles", "json")),
	)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	log.Infof("creating packages testdata Sample XLS")
	command = exec.Command(binPath, "packages",
		"--index-image=registry.redhat.io/redhat/certified-operator-index:v4.8",
		"--limit=2",
		fmt.Sprintf("--output-path=%s", filepath.Join(samplesDir, "packages", "xls")),
	)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	log.Infof("creating packages testdata Sample json")
	command = exec.Command(binPath, "packages",
		"--index-image=registry.redhat.io/redhat/certified-operator-index:v4.8",
		"--limit=2",
		"--output=json",
		fmt.Sprintf("--output-path=%s", filepath.Join(samplesDir, "packages", "json")),
	)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	log.Infof("creating channels testdata Sample XLS")
	command = exec.Command(binPath, "channels",
		"--index-image=registry.redhat.io/redhat/certified-operator-index:v4.8",
		"--limit=5",
		fmt.Sprintf("--output-path=%s", filepath.Join(samplesDir, "channels", "xls")),
	)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}

	log.Infof("creating channels testdata Sample json")
	command = exec.Command(binPath, "channels",
		"--index-image=registry.redhat.io/redhat/certified-operator-index:v4.8",
		"--limit=5",
		"--output=json",
		fmt.Sprintf("--output-path=%s", filepath.Join(samplesDir, "channels", "json")),
	)
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Errorf("running command :%s", err)
	}
}
