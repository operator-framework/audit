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

package hack

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"
)

const ReportsPath = "testdata/reports/"
const BinPath = "bin/audit-tool"

// GetImageNameToCreateDir returns the name of the image formatted to be used
// as the name of the dir
func GetImageNameToCreateDir(v string) string {
	name := strings.Split(v, ":")[0]
	name = strings.ReplaceAll(name, "registry.redhat.io/redhat/", "redhat_")
	name = strings.ReplaceAll(name, "quay.io/operatorhubio/", "operatorhubio_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "-", "_")
	return name
}

func ReplaceInFile(path, old, new string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	if !strings.Contains(string(b), old) {
		return errors.New("unable to find the content to be replaced")
	}
	s := strings.Replace(string(b), old, new, -1)
	err = ioutil.WriteFile(path, []byte(s), info.Mode())
	if err != nil {
		return err
	}
	return nil
}
