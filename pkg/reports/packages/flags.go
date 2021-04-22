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

package packages

type BindFlags struct {
	IndexImage        string `json:"index-image"`
	Limit             int32  `json:"limit"`
	Filter            string `json:"filter"`
	Label             string `json:"label"`
	LabelValue        string `json:"labelValue"`
	OutputPath        string `json:"outputPath"`
	OutputFormat      string `json:"outputFormat"`
	DisableScorecard  bool   `json:"disableScorecard"`
	DisableValidators bool   `json:"disableValidators"`
}
