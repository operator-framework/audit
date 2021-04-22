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
