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

package main

import (
	"log"

	"github.com/operator-framework/audit/cmd/custom"
	"github.com/operator-framework/audit/cmd/index"

	"github.com/spf13/cobra"
)

func main() {

	rootCmd := &cobra.Command{
		Use:   "audit-tool",
		Short: "An analytic tool to audit operator bundles and index catalogs",
		Long: "The audit is an analytic tool which uses the Operator Framework solutions. " +
			"Its purpose is to obtain and report and aggregate data provided by checks and analyses done in " +
			"the operator bundles, packages and channels from an index catalog image.\n\n" +
			"Note that the latest version of the reports generated for all images can be checked in its repository, " +
			"inside of `testdata/report`. \n **NOTE** The file names has been created by using the kind/type " +
			"of the report, image name and date. " +
			"(E.g. `testdata/report/bundles_quay.io_operatorhubio_catalog_latest_2021-04-22.xlsx`)" +
			"For further information use the --help and check : https://github.com/operator-framework/audit",
	}

	rootCmd.AddCommand(index.NewCmd())
	rootCmd.AddCommand(custom.NewCmd())

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
