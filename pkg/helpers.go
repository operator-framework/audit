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

package pkg

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const JSON = "json"
const Xls = "xls"
const All = "all"
const Yes = "YES"
const No = "NO"
const Unknown = "UNKNOWN"
const NotUsed = "NOT USED"

const TableFormat = `{
    "table_name": "table",
    "table_style": "TableStyleMedium2",
    "show_first_column": true,
    "show_last_column": true,
    "show_row_stripes": false,
    "show_column_stripes": false
}`

// PropertiesAnnotation used to Unmarshal the JSON in the CSV annotation
type PropertiesAnnotation struct {
	Type  string
	Value string
}

func (p PropertiesAnnotation) String() string {
	return fmt.Sprintf("{\"type\": \"%s\", \"value\": \"%s\"}", p.Type, p.Value)
}

// GetYesOrNo return the text yes for true values and No for false one.
func GetYesOrNo(value bool) string {
	if value {
		return Yes
	}
	return No
}

// Run executes the provided command within this context
func RunCommand(cmd *exec.Cmd) ([]byte, error) {
	command := strings.Join(cmd.Args, " ")
	log.Infof("running: %s\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s failed with error: (%v) %s", command, err, string(output))
	}
	if len(output) > 0 {
		log.Debugf("command output :%s", output)
	}
	return output, nil
}

// GetFormatArray return the values without duplicates and in a string such as "v","v"...
func GetFormatArrayWithBreakLine(array []string) string {
	var result string
	for _, n := range array {
		if !strings.Contains(result, n) {
			if len(result) > 0 {
				result = fmt.Sprintf("%s\n%s", result, n)
			} else {
				result = n
			}
		}
	}
	return result
}

// GetUniqueValues return the values without duplicates
func GetUniqueValues(array []string) []string {
	var result []string
	for _, n := range array {
		if len(result) == 0 {
			result = append(result, n)
			continue
		}
		found := false
		for _, v := range result {
			if strings.TrimSpace(n) == strings.TrimSpace(v) {
				found = true
				break
			}
		}
		if !found {
			result = append(result, n)
		}

	}
	return result
}

func WriteJSON(data []byte, imageName, outputPath, typeName string) error {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, data, "", "\t")
	if err != nil {
		return err
	}

	path := filepath.Join(outputPath, GetReportName(imageName, typeName, "json"))

	_, err = ioutil.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return ioutil.WriteFile(path, prettyJSON.Bytes(), 0644)
}

func GetReportName(imageName, typeName, typeFile string) string {
	dt := time.Now().Format("2006-01-02")

	//prepare image name to use as name of the file
	name := strings.ReplaceAll(imageName, "/", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "-", "_")

	return fmt.Sprintf("%s_%s_%s.%s", typeName, name, dt, typeFile)
}

func GenerateTemporaryDirs() {
	command := exec.Command("rm", "-rf", "tmp")
	_, _ = RunCommand(command)

	command = exec.Command("rm", "-rf", "./output/")
	_, _ = RunCommand(command)

	command = exec.Command("mkdir", "./output/")
	_, err := RunCommand(command)
	if err != nil {
		log.Fatal(err)
	}

	command = exec.Command("mkdir", "tmp")
	_, err = RunCommand(command)
	if err != nil {
		log.Fatal(err)
	}
}

func CleanupTemporaryDirs() {
	command := exec.Command("rm", "-rf", "tmp")
	_, _ = RunCommand(command)

	command = exec.Command("rm", "-rf", "./output/")
	_, _ = RunCommand(command)
}

type DockerInspectManifest struct {
	ID           string       `json:"ID"`
	RepoDigests  []string     `json:"RepoDigests"`
	Created      string       `json:"Created"`
	DockerConfig DockerConfig `json:"Config"`
}

type DockerConfig struct {
	Labels map[string]string `json:"Labels"`
}

func RunDockerInspect(image string) (DockerInspectManifest, error) {
	cmd := exec.Command("docker", "inspect", image)
	output, err := RunCommand(cmd)
	if err != nil || len(output) < 1 {
		return DockerInspectManifest{}, err
	}

	var dockerInspect []DockerInspectManifest
	if err := json.Unmarshal(output, &dockerInspect); err != nil {
		return DockerInspectManifest{}, err
	}
	return dockerInspect[0], nil
}

// HasClusterRunning will return true when is possible to check that the env has a cluster running
func HasClusterRunning() bool {
	command := exec.Command("kubectl", "cluster-info")
	output, err := RunCommand(command)
	if err != nil || !strings.Contains(string(output), "is running at") {
		return false
	}
	return true
}

// HasSDKInstalled will return true when find an SDK version installed
func HasSDKInstalled() bool {
	command := exec.Command("operator-sdk", "version")
	_, err := RunCommand(command)
	return err == nil
}

// ReadFile will return the bites of file
func ReadFile(file string) ([]byte, error) {
	jsonFile, err := os.Open(file)
	if err != nil {
		return []byte{}, err
	}
	defer jsonFile.Close()

	var byteValue []byte
	byteValue, err = ioutil.ReadAll(jsonFile)
	if err != nil {
		return []byte{}, err
	}
	return byteValue, err
}

// IsFollowingChannelNameConventional will check the channels.
func IsFollowingChannelNameConventional(channel string) bool {
	const candidate = "candidate"
	const stable = "stable"
	const fast = "fast"

	if !strings.HasPrefix(channel, candidate) &&
		!strings.HasPrefix(channel, stable) &&
		!strings.HasPrefix(channel, fast) {
		return false
	}

	return true
}
