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
	"fmt"
	"os/exec"

	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/audit/pkg"
)

const catalogIndex = "audit-catalog-index"

func ExtractIndexDB(image string, containerEngine string) error {
	log.Info("Extracting database...")
	// Remove image if exists already
	command := exec.Command(containerEngine, "rm", catalogIndex)
	_, _ = pkg.RunCommand(command)

	// Download the image
	command = exec.Command(containerEngine, "create", "--name", catalogIndex, image, "\"yes\"")
	_, err := pkg.RunCommand(command)
	if err != nil {
		return fmt.Errorf("unable to create container image %s : %s", image, err)
	}

	// Extract
	command = exec.Command(containerEngine, "cp", fmt.Sprintf("%s:/database/index.db", catalogIndex), "./output/")
	_, err = pkg.RunCommand(command)
	if err != nil {

		command = exec.Command(containerEngine, "cp", fmt.Sprintf("%s:/var/lib/iib/_hidden/do.not.edit.db", catalogIndex), "./output/")
		_, err = pkg.RunCommand(command)
		if err != nil {
			return fmt.Errorf("unable to extract the image for index.db %s : %s", image, err)
		}

		command = exec.Command("cp", "./output/do.not.edit.db", "./output/index.db")
		_, err = pkg.RunCommand(command)
		if err != nil {
			return fmt.Errorf("renaming do.not.edit.db to index.db %s : %s", image, err)
		}

	}
	return nil
}
