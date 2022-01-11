// Copyright 2021 The Audit Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this File except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package custom

// BindFlags define the Flags used to generate the bundle report
type BindFlags struct {
	Files          string            `json:"files,omitempty"`
	File           string            `json:"file,omitempty"`
	Template       string            `json:"template,omitempty"`
	OutputPath     string            `json:"outputPath,omitempty"`
	Filter         string            `json:"filter,omitempty"`
	OptionalValues map[string]string `json:"optionalValues,omitempty"`
}

var Flags = BindFlags{}
