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

package alpha

import (
	"strings"

	"github.com/operator-framework/audit/cmd/alpha/catalogs"
	"github.com/operator-framework/audit/cmd/alpha/maxocp"
	"github.com/spf13/cobra"
)

func NewCmd() *cobra.Command {
	indexCmd := &cobra.Command{
		Use:        "alpha",
		SuggestFor: []string{"experimental"},
		Short:      "Alpha-stage subcommands",
		Long: strings.TrimSpace(`
Alpha subcommands are for unstable features.
- Alpha subcommands are exploratory and may be removed without warning.
- No backwards compatibility is provided for any alpha subcommands.
`),
	}

	indexCmd.AddCommand(
		catalogs.NewCmd(),
		maxocp.NewCmd(),
	)

	return indexCmd

}
