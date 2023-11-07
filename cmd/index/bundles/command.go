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

package bundles

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/operator-framework/audit/pkg/actions"
	"github.com/operator-framework/operator-registry/alpha/declcfg"

	"github.com/spf13/cobra"

	// For connecting to query the legacy index database
	_ "github.com/mattn/go-sqlite3"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	log "github.com/sirupsen/logrus"

	"github.com/operator-framework/audit/pkg"
	"github.com/operator-framework/audit/pkg/models"
	index "github.com/operator-framework/audit/pkg/reports/bundles"
)

var flags = index.BindFlags{}

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundles",
		Short: "audit all operator bundles of an index catalog image",
		Long: `Provides reports with the details of all bundles operators ship in the index image informed. 

## When should I use it?

This command is used to extract the data required for audit tool be able to parse.
By running this command audit tool will:

- Extract the database from the image informed
- Perform SQL queries to obtain the data from the index db
- Download and extract all bundles files by using the operator bundle path which is stored in the index db  
- Get the required data for the report from the operator bundle manifest files 
- Use the [operator-framework/api][of-api] to execute the bundle validator checks
- Use SDK tool to execute the Scorecard bundle checks
- Output a report providing the information obtained and processed in JSON format.

`,

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
	cmd.Flags().BoolVar(&flags.StaticCheckFIPSCompliance, "static-check-fips-compliance", false,
		"If set, the tool will perform a static check for FIPS compliance on all bundle images.")
	cmd.Flags().StringVar(&flags.Filter, "filter", "",
		"filter by the packages names which are like *filter*")
	cmd.Flags().StringVar(&flags.OutputFormat, "output", pkg.JSON,
		fmt.Sprintf("inform the output format. [Options: %s]", pkg.JSON))
	cmd.Flags().StringVar(&flags.OutputPath, "output-path", currentPath,
		"inform the path of the directory to output the report. (Default: current directory)")
	cmd.Flags().Int32Var(&flags.Limit, "limit", 0,
		"limit the num of operator bundles to be audit")
	cmd.Flags().BoolVar(&flags.HeadOnly, "head-only", false,
		"if set, will just check the operator bundle which are head of the channels")
	cmd.Flags().BoolVar(&flags.DisableScorecard, "disable-scorecard", false,
		"if set, will disable the scorecard tests")
	cmd.Flags().BoolVar(&flags.DisableValidators, "disable-validators", false,
		"if set, will disable the validators tests")
	cmd.Flags().StringVar(&flags.Label, "label", "",
		"filter by bundles which has index images where contains *label*")
	cmd.Flags().StringVar(&flags.LabelValue, "label-value", "",
		"filter by bundles which has index images where contains *label=label-value*. "+
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

// CheckFIPSAnnotations searches for variants of the FIPS annotations.
func CheckFIPSAnnotations(csv *v1alpha1.ClusterServiceVersion) (bool, error) {
	fipsAnnotationPatterns := []string{
		"features.operators.openshift.io/fips-compliant",
		"operators.openshift.io/infrastructure-features",
	}

	for _, pattern := range fipsAnnotationPatterns {
		if value, exists := csv.Annotations[pattern]; exists &&
			(strings.Contains(value, "fips") || strings.Contains(value, "true")) {
			return true, nil
		}
	}
	return false, nil
}

// ExtractUniqueImageReferences get a unique list of operator image and related images
func ExtractUniqueImageReferences(operatorBundlePath string, csv *v1alpha1.ClusterServiceVersion) ([]string, error) {
	var imageRefs []string
	// Extract image references from RelatedImages slice
	for _, relatedImage := range csv.Spec.RelatedImages {
		imageRefs = append(imageRefs, relatedImage.Image)
	}
	imageRefs = append(imageRefs, operatorBundlePath)
	// Remove duplicates
	uniqueRefs := removeDuplicates(imageRefs)
	return uniqueRefs, nil
}

func removeDuplicates(elements []string) []string {
	encountered := map[string]bool{}
	result := []string{}

	for v := range elements {
		if !encountered[elements[v]] {
			encountered[elements[v]] = true
			result = append(result, elements[v])
		}
	}

	return result
}

// ExecuteExternalValidator runs the external validator on the provided image reference.
func ExecuteExternalValidator(imageRef string) (bool, []string, []string, error) {
	extValidatorCmd := "sudo check-payload scan operator --spec " + imageRef + " --log_file=/dev/null --output-format=csv"
	cmd := exec.Command("bash", "-c", extValidatorCmd)

	// Log the command being executed for debugging purposes
	log.Infof("Executing external validator with command: %s", extValidatorCmd)

	// Remove the image that check-payload has downloaded using the rmi command
	log.Infof("Removing image with command: %s rmi %s", flags.ContainerEngine, imageRef)
	rmiCmd := exec.Command(flags.ContainerEngine, "rmi", imageRef)
	_, _ = pkg.RunCommand(rmiCmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, nil, nil, err
	}

	lines := strings.Split(string(output), "\n")
	processingMode := "" // can be "warning", "error", or empty
	hasReports := false

	var warnings, errors []string
	for _, line := range lines {
		if line == "---- Warning Report" {
			processingMode = "warning"
			hasReports = true
			continue
		} else if line == "---- Error Report" {
			processingMode = "error"
			hasReports = true
			continue
		} else if strings.HasPrefix(line, "Operator Name,Executable Name,Status,Image") {
			continue
		}

		if processingMode == "" {
			continue
		}

		columns := strings.Split(line, ",")
		if len(columns) < 4 {
			continue
		}
		operatorName, executableName, status, image := columns[0], columns[1], columns[2], columns[3]
		if processingMode == "warning" {
			warnings = append(warnings, fmt.Sprintf("Warning for Operator '%s', Executable '%s': %s (Image: %s)",
				operatorName, executableName, status, image))
		} else if processingMode == "error" {
			errors = append(errors, fmt.Sprintf("Error for Operator '%s', Executable '%s': %s (Image: %s)",
				operatorName, executableName, status, image))
		}
	}

	if !hasReports {
		successMessage := fmt.Sprintf("FIPS compliance check passed successfully for image: %s", imageRef)
		warnings = append(warnings, successMessage) // or choose a different way to report this success
	}

	return true, warnings, errors, nil
}

// ProcessValidatorResults takes the results from the external validator and appends them to the report data.
func ProcessValidatorResults(success bool, warnings, errors []string, report *index.Data) {
	// Create a slice to hold combined errors and warnings
	combinedErrors := make([]string, 0)

	// If the external validator fails, append the errors
	if !success {
		combinedErrors = append(combinedErrors, errors...)
	}

	// Prepend warnings with "WARNING:" and append to combinedErrors
	for _, warning := range warnings {
		combinedErrors = append(combinedErrors, "WARNING: "+warning)
	}

	// Assuming there's a mechanism to identify which bundle is being processed
	// Here, I'm just using the last bundle in the report as an example
	if len(report.AuditBundle) > 0 {
		report.AuditBundle[len(report.AuditBundle)-1].Errors = combinedErrors
	}
}

func validation(cmd *cobra.Command, args []string) error {

	if flags.Limit < 0 {
		return fmt.Errorf("invalid value informed via the --limit flag :%v", flags.Limit)
	}

	if len(flags.OutputFormat) > 0 && flags.OutputFormat != pkg.JSON {
		return fmt.Errorf("invalid value informed via the --output flag :%v. "+
			"The available option is: %s", flags.OutputFormat, pkg.JSON)
	}

	if len(flags.OutputPath) > 0 {
		if _, err := os.Stat(flags.OutputPath); os.IsNotExist(err) {
			return err
		}
	}

	if len(flags.LabelValue) > 0 && len(flags.Label) == 0 {
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
		return fmt.Errorf("invalid value for the flag --container-engine (%s)."+
			" The valid options are %s and %s", flags.ContainerEngine, pkg.Docker, pkg.Podman)
	}

	return nil
}

func run(cmd *cobra.Command, args []string) error {
	log.Info("Starting audit...")

	reportData := index.Data{}
	reportData.Flags = flags
	pkg.GenerateTemporaryDirs()

	// to fix common possible typo issue
	reportData.Flags.Filter = strings.ReplaceAll(reportData.Flags.Filter, "”", "")

	if err := actions.DownloadImage(flags.IndexImage, flags.ContainerEngine); err != nil {
		return err
	}

	// Inspect the OLM index image
	var err error
	reportData.IndexImageInspect, err = pkg.RunDockerInspect(flags.IndexImage, flags.ContainerEngine)
	if err != nil {
		log.Errorf("unable to inspect the index image: %s", err)
	}

	if err := actions.ExtractIndexDBorCatalogs(flags.IndexImage, flags.ContainerEngine); err != nil {
		return err
	}

	log.Info("Gathering data...")

	// check here to see if it's index.db or file-based catalogs
	if IsFBC(flags.IndexImage) {
		reportData, _ = GetDataFromFBC(reportData)
	} else {
		reportData, _ = GetDataFromIndexDB(reportData)
	}
	if err := reportData.OutputReport(); err != nil {
		return err
	}
	pkg.CleanupTemporaryDirs()
	log.Info("Operation completed.")
	return nil
}

func handleFIPS(operatorBundlePath string, csv *v1alpha1.ClusterServiceVersion, reportData index.Data) error {
	isClaimingFIPSCompliant, err := CheckFIPSAnnotations(csv)
	if err != nil {
		return err
	}
	if !isClaimingFIPSCompliant {
		return nil
	}
	uniqueImageRefs, err := ExtractUniqueImageReferences(operatorBundlePath, csv)
	if err != nil {
		return err
	}

	for _, imageRef := range uniqueImageRefs {
		success, warnings, errors, err := ExecuteExternalValidator(imageRef)
		if err != nil {
			log.Errorf("Error while executing FIPS compliance check on image: %s. Error: %s",
				imageRef, err.Error())
			return err
		}
		ProcessValidatorResults(success, warnings, errors, &reportData)
	}
	return nil
}

func IsFBC(indexImage string) bool {
	//check if /output/versiontag/configs is populated to determine if the catalog is file-based
	root := "./output/" + actions.GetVersionTagFromImage(indexImage) + "/configs"
	f, err := os.Open(root)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = f.Readdir(1)
	if err == io.EOF {
		return false
	}
	log.Infof("./output/%s/configs is present & populated so this must be a file-based config catalog",
		actions.GetVersionTagFromImage(indexImage))
	return true
}

func GetDataFromFBC(report index.Data) (index.Data, error) {
	root := "./output/" + actions.GetVersionTagFromImage(report.Flags.IndexImage) + "/configs"
	fileSystem := os.DirFS(root)
	fbc, err := declcfg.LoadFS(fileSystem)

	if err != nil {
		return report, fmt.Errorf("unable to load the file based config : %s", err)
	}
	model, err := declcfg.ConvertToModel(*fbc)
	var auditBundle *models.AuditBundle
	if err != nil {
		return report, fmt.Errorf("unable to file based config to internal model: %s", err)
	}
	// iterate the model by bundle to match up with how code in getDataFromIndexDB() does it
	for packageName, Package := range model {
		for _, Channel := range Package.Channels {
			for _, Bundle := range Channel.Bundles {
				log.Infof("Generating data from the bundle (%s)", Bundle.Name)
				auditBundle = models.NewAuditBundle(Bundle.Name, Bundle.Image)
				var csv *v1alpha1.ClusterServiceVersion
				err := json.Unmarshal([]byte(Bundle.CsvJSON), &csv)
				if err == nil {
					auditBundle.CSVFromIndexDB = csv
				} else {
					auditBundle.Errors = append(auditBundle.Errors,
						fmt.Errorf("unable to parse the csv from the index.db: %s", err).Error())
				}
				auditBundle = actions.GetDataFromBundleImage(auditBundle, report.Flags.DisableScorecard,
					report.Flags.DisableValidators, report.Flags.ServerMode, report.Flags.Label,
					report.Flags.LabelValue, flags.ContainerEngine, report.Flags.IndexImage)
				// an extra inner loop is needed because of the way the model is set up vs. how the report is generated
				for _, Channel := range Package.Channels {
					auditBundle.Channels = append(auditBundle.Channels, Channel.Name)
				}
				auditBundle.PackageName = packageName
				auditBundle.DefaultChannel = Package.DefaultChannel.Name
				// this collects olm.bundle.objects not found in the index version and this seems correct
				for _, property := range Bundle.Properties {
					auditBundle.PropertiesDB = append(auditBundle.PropertiesDB,
						pkg.PropertiesAnnotation{Type: property.Type, Value: string(property.Value)})
				}
				headBundle, err := Channel.Head()
				if err == nil {
					if headBundle == Bundle {
						auditBundle.IsHeadOfChannel = true
					}
				}
				if flags.StaticCheckFIPSCompliance && auditBundle.IsHeadOfChannel {
					err = handleFIPS(auditBundle.OperatorBundleImagePath, csv, report)
					if err != nil {
						// Check for specific error types and provide more informative messages
						if exitError, ok := err.(*exec.ExitError); ok {
							if exitError.ExitCode() == 127 {
								auditBundle.Errors = append(auditBundle.Errors,
									"Failed to run FIPS external validator: Command not found.")
							} else {
								auditBundle.Errors = append(auditBundle.Errors,
									fmt.Sprintf("FIPS external validator returned with exit code %d.", exitError.ExitCode()))
							}
						} else {
							auditBundle.Errors = append(auditBundle.Errors,
								fmt.Sprintf("Difficulty running FIPS external validator: %s", err.Error()))
						}
					}
				}
				report.AuditBundle = append(report.AuditBundle, *auditBundle)
			}
		}
	}
	return report, nil
}

func GetDataFromIndexDB(report index.Data) (index.Data, error) {
	// Connect to the database
	db, err := sql.Open("sqlite3", "./output/"+
		actions.GetVersionTagFromImage(report.Flags.IndexImage)+"/index.db")
	if err != nil {
		return report, fmt.Errorf("unable to connect in to the database : %s", err)
	}

	sql, err := report.BuildBundlesQuery()
	if err != nil {
		return report, err
	}

	row, err := db.Query(sql)
	if err != nil {
		return report, fmt.Errorf("unable to query the index db : %s", err)
	}

	defer row.Close()
	for row.Next() {
		var bundleName string
		var csv *string
		var bundlePath string
		var csvStruct *v1alpha1.ClusterServiceVersion

		err = row.Scan(&bundleName, &csv, &bundlePath)
		if err != nil {
			log.Errorf("unable to scan data from index %s\n", err.Error())
		}
		log.Infof("Generating data from the bundle (%s)", bundleName)
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

		auditBundle = actions.GetDataFromBundleImage(auditBundle, report.Flags.DisableScorecard,
			report.Flags.DisableValidators, report.Flags.ServerMode, report.Flags.Label,
			report.Flags.LabelValue, flags.ContainerEngine, report.Flags.IndexImage)

		sqlString := fmt.Sprintf("SELECT c.channel_name, c.package_name FROM channel_entry c "+
			"where c.operatorbundle_name = '%s'", auditBundle.OperatorBundleName)
		row, err := db.Query(sqlString)
		if err != nil {
			return report, fmt.Errorf("unable to query channel entry in the index db : %s", err)
		}

		defer row.Close()
		var channelName string
		var packageName string
		for row.Next() { // Iterate and fetch the records from result cursor
			_ = row.Scan(&channelName, &packageName)
			auditBundle.Channels = append(auditBundle.Channels, channelName)
			auditBundle.PackageName = packageName
		}

		if len(strings.TrimSpace(auditBundle.PackageName)) == 0 && auditBundle.Bundle != nil {
			auditBundle.PackageName = auditBundle.Bundle.Package
		}

		sqlString = fmt.Sprintf("SELECT default_channel FROM package WHERE name = '%s'", auditBundle.PackageName)
		row, err = db.Query(sqlString)
		if err != nil {
			return report, fmt.Errorf("unable to query default channel entry in the index db : %s", err)
		}

		defer row.Close()
		var defaultChannelName string
		for row.Next() { // Iterate and fetch the records from result cursor
			_ = row.Scan(&defaultChannelName)
			auditBundle.DefaultChannel = defaultChannelName
		}

		//TODO Think this should actually be:
		// SELECT DISTINCT type, value FROM properties
		// WHERE operatorbundle_name=?
		// AND (operatorbundle_version=? OR operatorbundle_version is NULL)
		// AND (operatorbundle_path=? OR operatorbundle_path is NULL)
		// but leaving this as-is because this is the baseline for index-based audit reports.
		// The redundant entries caused w/out DISTINCT seem okay?
		sqlString = fmt.Sprintf("SELECT type, value FROM properties WHERE operatorbundle_name = '%s'",
			auditBundle.OperatorBundleName)
		row, err = db.Query(sqlString)
		if err != nil {
			return report, fmt.Errorf("unable to query properties entry in the index db : %s", err)
		}

		defer row.Close()
		var properType string
		var properValue string
		for row.Next() { // Iterate and fetch the records from result cursor
			_ = row.Scan(&properType, &properValue)
			auditBundle.PropertiesDB = append(auditBundle.PropertiesDB,
				pkg.PropertiesAnnotation{Type: properType, Value: properValue})
		}

		sqlString = fmt.Sprintf("select count(*) from channel where head_operatorbundle_name = '%s'",
			auditBundle.OperatorBundleName)
		row, err = db.Query(sqlString)
		if err != nil {
			return report, fmt.Errorf("unable to query properties entry in the index db : %s", err)
		}

		defer row.Close()
		var found int
		for row.Next() { // Iterate and fetch the records from result cursor
			_ = row.Scan(&found)
			auditBundle.IsHeadOfChannel = found > 0
		}

		report.AuditBundle = append(report.AuditBundle, *auditBundle)
	}

	return report, nil
}
