## PKG/ACTIONS Directory Run Down

The `pkg/actions` directory focuses on core actions and functionalities of the audit tool.

### run_validators.go

- **RunValidators**: Main function to run various validators.
- **checkBundleAgainstCommonCriteria**: Checks bundles against common criteria.
- **fromOCPValidator**: Related to OCP validation.
- **fromAuditValidatorsBundleSize**: Checks bundle sizes as part of the validation.

### run_scorecard.go

- **RunScorecard**: Main function to run the scorecard functionality.
- **writeScorecardConfig**: Writes or updates the configuration for the scorecard.

### get_bundle.go

- **GetDataFromBundleImage**: Fetches data from a bundle image.
- **createBundleDir**: Creates a directory for the bundle.
- **extractBundleFromImage**: Extracts bundle data from an image.
- **cleanupBundleDir**: Cleans up the bundle directory after processing.
- **DownloadImage**: Downloads an image for further processing or analysis.

### extract_index.go

- **ExtractIndexDBorCatalogs**: Extracts database or catalogs from an index.
- **GetVersionTagFromImage**: Retrieves the version tag from an image.
