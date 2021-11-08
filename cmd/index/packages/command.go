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

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/operator-framework/audit/pkg/actions"

	log "github.com/sirupsen/logrus"

	"database/sql"
	"fmt"

	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	// To allow create connection to query the index database
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/models"
	"github.com/operator-framework/audit/pkg/reports/packages"
)

var flags = packages.BindFlags{}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "packages",
		Short: "audit all packages of an index catalog image (only use head bundles)",
		Long: "Provides reports with the details based on the packages and their latest operator bundle versions (head) " +
			"which are ship in the index image informed " +
			"according to the criteria defined via the flags.\n\n " +
			"**When this report is useful?** \n\n" +
			"This report is useful when you need to audit the packages as their latest state.",
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
		fmt.Sprintf("inform the output format. [Options: %s, %s, %s]", pkg.JSON,
			pkg.Xls, pkg.All))
	cmd.Flags().StringVar(&flags.OutputPath, "output-path", currentPath,
		"inform the path of the directory to output the report. (Default: current directory)")
	cmd.Flags().BoolVar(&flags.DisableScorecard, "disable-scorecard", false,
		"if set, will disable the scorecard tests")
	cmd.Flags().BoolVar(&flags.DisableValidators, "disable-validators", false,
		"if set, will disable the validators tests")
	cmd.Flags().StringVar(&flags.Label, "label", "",
		"filter by packages which has bundles with index images where contains *label*")
	cmd.Flags().StringVar(&flags.LabelValue, "label-value", "",
		"filter by packages which has bundles with index images where contains *label=label-value*. "+
			"This option can only be used with the --label flag.")
	cmd.Flags().BoolVar(&flags.ServerMode, "server-mode", false,
		"if set, the images which are downloaded will not be removed. This flag should be used on dedicated "+
			"environments and reduce the cost to generate the reports periodically")
	cmd.Flags().StringVar(&flags.ContainerEngine, "container-engine", pkg.Docker,
		fmt.Sprintf("specifies the container tool to use. If not set, the default value is docker. "+
			"Note that you can use the environment variable CONTAINER_ENGINE to inform this option. "+
			"[Options: %s and %s]", pkg.Docker, pkg.Podman))

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

	if len(flags.LabelValue) > 0 && len(flags.Label) < 0 {
		return fmt.Errorf("inform the label via the --label flag")
	}

	if !flags.DisableScorecard {
		if !pkg.HasClusterRunning() {
			return errors.New("this report is configured to run the Scorecard tests which requires a cluster up " +
				"and running. Please, startup your cluster or use the flag --disable-scorecard")
		}
		if !pkg.HasSDKInstalled() {
			return errors.New("this report is configured to run the Scorecard tests which requires the " +
				"SDK CLI version >= 1.5 installed locally.\n" +
				"Please, see ensure that you have SDK installed or use the flag --disable-scorecard.\n" +
				"More info: https://github.com/operator-framework/operator-sdk")
		}
	}

	if len(flags.ContainerEngine) == 0 {
		flags.ContainerEngine = pkg.GetContainerToolFromEnvVar()
	}
	if flags.ContainerEngine != pkg.Docker && flags.ContainerEngine != pkg.Podman {
		return fmt.Errorf("invalid value for the flag --container-engine (%s). "+
			"The valid options are %s and %s", flags.ContainerEngine, pkg.Docker, pkg.Podman)
	}

	return nil
}

func run(cmd *cobra.Command, args []string) error {
	log.Info("Starting audit...")
	reportData := packages.Data{}
	reportData.Flags = flags
	// to fix common possible typo issue
	reportData.Flags.Filter = strings.ReplaceAll(reportData.Flags.Filter, "”", "")
	pkg.GenerateTemporaryDirs()

	if err := actions.DownloadImage(flags.IndexImage, flags.ContainerEngine); err != nil {
		return err
	}

	// Inspect the OLM index image
	var err error
	reportData.IndexImageInspect, err = pkg.RunDockerInspect(flags.IndexImage, flags.ContainerEngine)
	if err != nil {
		log.Errorf("unable to inspect the index image: %s", err)
	}

	if err := actions.ExtractIndexDB(flags.IndexImage, flags.ContainerEngine); err != nil {
		return err
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

func getDataFromIndexDB(report packages.Data) (packages.Data, error) {
	// Connect to the database
	db, err := sql.Open("sqlite3", "./output/index.db")
	if err != nil {
		return report, fmt.Errorf("unable to connect in to the database : %s", err)
	}

	sql, err := report.BuildPackagesQuery()
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
		var defaultChannel string

		if err := row.Scan(&name, &defaultChannel); err != nil {
			log.Errorf("unable to scan data from index %s\n", err.Error())
		}

		col := models.NewAuditPackage(name)
		report.AuditPackage = append(report.AuditPackage, *col)
	}

	for k, v := range report.AuditPackage {
		sqlString := fmt.Sprintf("SELECT count(DISTINCT(name)) FROM channel "+
			"WHERE channel.package_name = \"%s\"", v.PackageName)

		row, err = db.Query(sqlString)
		if err != nil {
			log.Errorf("error to check if is multi-channel: %s", err)
		}

		defer row.Close()

		for row.Next() {
			var count int
			if err := row.Scan(&count); err != nil {
				log.Errorf("unable to scan data from index %s\n", err.Error())
			}

			if count > 1 {
				report.AuditPackage[k].IsMultiChannel = true
			}
		}
	}

	for k, v := range report.AuditPackage {
		sqlString := fmt.Sprintf("SELECT operatorbundle.name, operatorbundle.csv, operatorbundle.bundlepath "+
			"FROM package, channel, operatorbundle "+
			"WHERE channel.head_operatorbundle_name == operatorbundle.name "+
			"AND channel.package_name == package.name  "+
			"AND package.name == \"%s\"", v.PackageName)
		row, err = db.Query(sqlString)
		if err != nil {
			log.Errorf("error to get bundles for the package: %s", err)

		}

		defer row.Close()
		for row.Next() {
			var bundleName string
			var csv *string
			var bundlePath string
			var csvStruct *v1alpha1.ClusterServiceVersion
			if err := row.Scan(&bundleName, &csv, &bundlePath); err != nil {
				log.Errorf("unable to scan data from index %s\n", err.Error())
			}

			auditBundle := models.NewAuditBundle(bundleName, bundlePath)
			// the csv is pruned from the database to save space.
			// See that is store only what is needed to populate the package manifest on cluster, all the extra
			// manifests are pruned to save storage space
			if csv != nil {
				err = json.Unmarshal([]byte(*csv), &csvStruct)
				if err == nil {
					auditBundle.CSVFromIndexDB = csvStruct
				} else {
					auditBundle.Errors = append(auditBundle.Errors,
						fmt.Errorf("unable to parse the csv from the index.db: %s", err).Error())
				}
			}

			auditBundle = actions.GetDataFromBundleImage(auditBundle,
				report.Flags.DisableScorecard, report.Flags.DisableValidators, report.Flags.ServerMode,
				report.Flags.Label, report.Flags.LabelValue, flags.ContainerEngine)

			if len(strings.TrimSpace(auditBundle.PackageName)) == 0 && auditBundle.Bundle != nil {
				auditBundle.PackageName = auditBundle.Bundle.Package
			}

			sqlString = fmt.Sprintf("SELECT type, value FROM properties WHERE operatorbundle_name = '%s'",
				auditBundle.OperatorBundleName)
			row, err = db.Query(sqlString)
			if err != nil {
				return report, fmt.Errorf("unable to query default channel entry in the index db : %s", err)
			}

			defer row.Close()
			var properType string
			var properValue string
			for row.Next() { // Iterate and fetch the records from result cursor
				_ = row.Scan(&properType, &properValue)
				auditBundle.PropertiesDB = append(auditBundle.PropertiesDB,
					pkg.PropertiesAnnotation{Type: properType, Value: properValue})
			}
			report.AuditPackage[k].AuditBundle = append(report.AuditPackage[k].AuditBundle, *auditBundle)
		}

	}

	return report, nil
}
