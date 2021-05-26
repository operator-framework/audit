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

package channels

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
	// To allow create connection to query the index database
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/models"
	"github.com/operator-framework/audit/pkg/reports/channels"
)

const catalogIndex = "audit-catalog-index"

var flags = channels.BindFlags{}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "channels",
		Short: "audit all channels of an index catalog image",
		Long: "Provides reports with the details based on the channels and their operator bundle versions " +
			"which are ship in the index image informed " +
			"according to the criteria defined via the flags.\n\n " +
			"**When this report is useful?** \n\n" +
			"This report is useful when is required to audit the channels" +
			" to check issues that can affect the upgrade graphs.",
		PreRunE: validation,
		RunE:    run,
	}

	currentPath, err := os.Getwd()
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	cmd.Flags().StringVar(&flags.IndexImage, "index-image", "",
		"index image and tag which will be audit")
	if err := cmd.MarkFlagRequired("index-image"); err != nil {
		log.Fatalf("Failed to mark `index-image` flag for `index` sub-command as required")
	}

	cmd.Flags().StringVar(&flags.Filter, "filter", "",
		"filter by the packages names which are like *filter*")
	cmd.Flags().Int32Var(&flags.Limit, "limit", 0,
		"limit the num of packages to be audit")
	cmd.Flags().StringVar(&flags.OutputFormat, "output", pkg.Xls,
		fmt.Sprintf("inform the output format. [Flags: %s, %s, %s]", pkg.JSON,
			pkg.Xls, pkg.All))
	cmd.Flags().StringVar(&flags.OutputPath, "output-path", currentPath,
		"inform the path of the directory to output the report. (Default: current directory)")

	return cmd
}

func validation(cmd *cobra.Command, args []string) error {

	if flags.Limit < 0 {
		return fmt.Errorf("invalid value informed via the --limit flag :%v", flags.Limit)
	}

	if len(flags.OutputFormat) > 0 && flags.OutputFormat != pkg.JSON &&
		flags.OutputFormat != pkg.Xls && flags.OutputFormat != pkg.All {
		return fmt.Errorf("invalid value informed via the --output flag :%v. "+
			"The available options are: %s, %s and %s", flags.OutputFormat, pkg.JSON, pkg.Xls, pkg.All)
	}

	if len(flags.OutputPath) > 0 {
		if _, err := os.Stat(flags.OutputPath); os.IsNotExist(err) {
			return fmt.Errorf("invalid directory path informed via the flag output-path (%s) : %s ",
				flags.OutputPath, err)
		}
	}

	return nil
}

func run(cmd *cobra.Command, args []string) error {
	log.Info("Starting audit...")
	reportData := channels.Data{}
	reportData.Flags = flags
	pkg.GenerateTemporaryDirs()

	// to fix common possible typo issue
	reportData.Flags.Filter = strings.ReplaceAll(reportData.Flags.Filter, "”", "")

	if err := extractIndexDB(); err != nil {
		return err
	}

	// Inspect the OLM index image
	var err error
	reportData.IndexImageInspect, err = pkg.RunDockerInspect(flags.IndexImage)
	if err != nil {
		log.Errorf("unable to inspect the index image: %s", err)
	}

	reportData, err = getDataFromIndexDB(reportData)
	if err != nil {
		return err
	}

	log.Infof("Start to generate the report")
	if err := reportData.OutputReport(); err != nil {
		return err
	}

	pkg.CleanupTemporaryDirs()
	log.Infof("Operation completed.")

	return nil
}

func extractIndexDB() error {
	// Remove image if exists already
	command := exec.Command("docker", "rm", catalogIndex)
	_, _ = pkg.RunCommand(command)

	// Download the image
	command = exec.Command("docker", "create", "--name", catalogIndex, flags.IndexImage, "\"yes\"")
	_, err := pkg.RunCommand(command)
	if err != nil {
		return fmt.Errorf("unable to create container image %s : %s", flags.IndexImage, err)
	}

	// Extract
	command = exec.Command("docker", "cp", fmt.Sprintf("%s:/database/index.db", catalogIndex), "./output/")
	_, err = pkg.RunCommand(command)
	if err != nil {
		return fmt.Errorf("unable to extract the image for index.db %s : %s", flags.IndexImage, err)
	}
	return nil
}

func getDataFromIndexDB(report channels.Data) (channels.Data, error) {
	// Connect to the database
	db, err := sql.Open("sqlite3", "./output/index.db")
	if err != nil {
		return report, fmt.Errorf("unable to connect in to the database : %s", err)
	}

	sql, err := report.BuildChannelsQuery()
	if err != nil {
		return report, err
	}

	row, err := db.Query(sql)
	if err != nil {
		return report, fmt.Errorf("unable to query the index db : %s", err)
	}

	defer row.Close()
	for row.Next() {
		var name string
		var packageName string
		var headOperatorBundle string

		if err := row.Scan(&name, &packageName, &headOperatorBundle); err != nil {
			log.Errorf("unable to scan data from index %s\n", err.Error())
		}

		col := models.NewAuditChannels(packageName, name, headOperatorBundle)
		report.AuditChannel = append(report.AuditChannel, *col)
	}

	for k, v := range report.AuditChannel {
		sqlString := fmt.Sprintf("SELECT o.name, o.version, o.skiprange, o.replaces, "+
			"o.skips from channel_entry ce, operatorbundle o "+
			"WHERE ce.operatorbundle_name = o.name AND ce.channel_name = \"%s\"", v.ChannelName)

		row, err = db.Query(sqlString)
		if err != nil {
			log.Errorf("error to query the bundles for the channel : %s", err)
		}

		defer row.Close()

		for row.Next() {
			var bundleName string
			var skipRange string
			var version string
			var replaces string
			var skips string

			err = row.Scan(&bundleName, &version, &skipRange, &replaces, &skips)
			if err != nil {
				log.Errorf("unable to scan data from index %s\n", err.Error())
			}

			auditBundle := models.NewAuditBundle(bundleName, "")
			auditBundle.VersionDB = version
			auditBundle.SkipRangeDB = skipRange
			auditBundle.ReplacesDB = replaces
			auditBundle.SkipsDB = skips
			report.AuditChannel[k].AuditBundles = append(report.AuditChannel[k].AuditBundles, *auditBundle)
		}
	}
	return report, nil
}
