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

package custom

import (
	"github.com/operator-framework/audit/cmd/custom/catalogs"
	"github.com/operator-framework/audit/cmd/custom/maxocp"
	"github.com/operator-framework/audit/cmd/custom/multiarch"
	"github.com/operator-framework/audit/cmd/custom/validator"
	"github.com/spf13/cobra"

	"github.com/operator-framework/audit/cmd/custom/deprecate"
	"github.com/operator-framework/audit/cmd/custom/grade"
)

func NewCmd() *cobra.Command {
	indexCmd := &cobra.Command{
		Use:   "dashboard",
		Short: "generate specific custom reports based on the audit JSONs output",
	}

	indexCmd.AddCommand(
		deprecate.NewCmd(),
		grade.NewCmd(),
		maxocp.NewCmd(),
		multiarch.NewCmd(),
		validator.NewCmd(),
		catalogs.NewCmd(),
	)

	return indexCmd

}
