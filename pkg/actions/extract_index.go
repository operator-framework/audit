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
	"strings"

	"github.com/operator-framework/audit/pkg"
	log "github.com/sirupsen/logrus"
)

const catalogIndex = "audit-catalog-index"

func ExtractIndexDBorCatalogs(image string, containerEngine string) error {
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

	versionTag := GetVersionTagFromImage(image)

	// Extract
	command = exec.Command("mkdir", "./output/"+versionTag+"/")
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Fatal(err)
	}
	// sqlite db
	command = exec.Command(containerEngine, "cp", fmt.Sprintf("%s:/database/index.db", catalogIndex),
		"./output/"+versionTag+"/")
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Infof("unable to extract index.db (probably file based config index) %s : %s", image, err)
	}
	// transitional indexes have a hidden sqlite db, copy it, and change the name to just index.db
	command = exec.Command(containerEngine, "cp",
		fmt.Sprintf("%s:/var/lib/iib/_hidden/do.not.edit.db", catalogIndex), "./output/"+versionTag+"/index.db")
	_, err = pkg.RunCommand(command)
	if err != nil {
		log.Infof("unable to extract the image for index.db (transition or file based config index) %s : %s", image, err)
	}
	// For FBC extract they are on the image, in /configs/<package_name>/catalog.json
	command = exec.Command(containerEngine, "cp", fmt.Sprintf("%s:/configs/", catalogIndex),
		"./output/"+versionTag+"/")
	_, errFbc := pkg.RunCommand(command)
	if errFbc != nil {
		log.Infof("copying file-based configs %s : %s", image, err)
	}
	return nil
}

// get the tag from an image URL
func GetVersionTagFromImage(image string) string {
	var versionTag string
	splitImage := strings.SplitN(image, ":", 2)
	if len(splitImage) == 2 {
		versionTag = splitImage[1]
	}
	return versionTag
}
