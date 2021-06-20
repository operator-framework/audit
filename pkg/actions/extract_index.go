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

	"github.com/operator-framework/audit/pkg"
)

const catalogIndex = "audit-catalog-index"

func ExtractIndexDB(image string) error {
	// Remove image if exists already
	command := exec.Command("docker", "rm", catalogIndex)
	_, _ = pkg.RunCommand(command)

	// Download the image
	command = exec.Command("docker", "create", "--name", catalogIndex, image, "\"yes\"")
	_, err := pkg.RunCommand(command)
	if err != nil {
		return fmt.Errorf("unable to create container image %s : %s", image, err)
	}

	// Extract
	command = exec.Command("docker", "cp", fmt.Sprintf("%s:/database/index.db", catalogIndex), "./output/")
	_, err = pkg.RunCommand(command)
	if err != nil {
		return fmt.Errorf("unable to extract the image for index.db %s : %s", image, err)
	}
	return nil
}
