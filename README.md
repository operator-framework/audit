[![Go Report Card](https://goreportcard.com/badge/github.com/camilamacedo86/audit)](https://goreportcard.com/report/github.com/camilamacedo86/audit)
[![Coverage Status](https://coveralls.io/repos/github/github.com/operator-framework/audit/badge.svg?branch=main)](https://coveralls.io/github/camilamacedo86/audit?branch=main)

---
# Audit

**IMPORTANT** This project still at the POC level since its first release did not get done so far. Before running the reports, ensure that you have its latest version by running `git pull` and `git status`.

## Overview

The audit is an analytic tool which uses the Operator Framework solutions. Its purpose is to obtain and report and aggregate data provided by checks and analyses done in the operator bundles, packages and channels from an index catalog image.

Note that the latest version of the reports generated for all images can be checked in [testdata/report](testdata/reports). The file names are create by using the kind/type of the report, image name and date. (E.g. `testdata/report/bundles_quay.io_operatorhubio_catalog_latest_2021-04-22.xlsx`)

### Goals

- Be able to audit and gathering aspects all bundles, packages and channel of an OLM index catalog and output a report
- Be able to extract a report with the audit results and in some formats such as json. 
- Be able to perform validations and analyses in the index catalog for the bundle and catalog level.

For further information about its motivation see the [EP Audit command operation][audit-ep]. 

## Pre-requirements

- go 1.15+ < 1.16
- docker 
- access to the registry where the index catalog and operator bundle images are distribute
- access to a Kubernetes cluster
- [operator-sdk][operator-sdk] installed >= `1.5.0

NOTE that you can run the reports without SDK and the cluster running with by using the flag `--disable-scorecard`. That is only required for the scorecard results.  

## Install

To get the project and install the binary:

```sh
$ git clone git@github.com:operator-framework/audit.git
$ cd audit
$ make install
```

## Usage

### Ensure that you have access to pull the images

You need first use docker login to have access to the images. Following the example for the OCP downstream catalog to audit the image `registry.redhat.io/redhat/certified-operator-index:v4.7`.

```sh
docker login https://registry.connect.redhat.com
docker login https://registry.redhat.io
```

### Generating the reports

Now, you can audit all operator bundles of an image catalog with: 

```sh 
audit-tool bundles --index-image=registry.redhat.io/redhat/redhat--operator-index:v4.7 --head-only --output-path=testdata/xls
```

Now, you can audit all packages of an image catalog with: 

```sh 
audit-tool packages --index-image=registry.redhat.io/redhat/redhat--operator-index:v4.7 --output-path=testdata/xls
```

Note that you can also output the results in JSON format:

```sh 
audit-tool bundles index \
    --index-image=registry.redhat.io/redhat/redhat-operator-index:v4.7 \
    --limit=3 \
    --head-only \
    --output=json \  
    --output-path=testdata/json
``` 

### Options

Use the `--help` flag to check the options and the further information about its commands. Following an example:

```sh
$ audit-tool bundles --help
Provides reports with the details of all bundles operators ship in the index image informed according to the criteria defined via the flags.

 **When this report is useful?** 

This report is useful when is required to check the operator bundles details.

Usage:
  audit-tool bundles [flags]

Flags:
      --disable-scorecard    if set, will disable the scorecard tests
      --disable-validators   if set, will disable the validators tests
      --filter string        filter by the packages names which are like *filter*
      --head-only            if set, will just check the operator bundle which are head of the channels
  -h, --help                 help for bundles
      --index-image string   index image and tag which will be audit
      --label string         filter by bundles which has index images where contains *label*
      --label-value string   filter by bundles which has index images where contains *label=label-value*. This option can only be used with the --label flag.
      --limit int32          limit the num of operator bundles to be audit
      --output string        inform the output format. [Flags: xls, json]. (Default: xls) (default "xls")
      --output-path string   inform the path of the directory to output the report. (Default: current directory) (default "/Users/camilamacedo/go/src/github.com/operator-framework/audit-1")
```

### Filtering results by names

See that you can use the `--filter` --flag to filter the results by the package name:

```sh
./bin/audit audit [bundles|packages|channels] --index-image=registry.redhat.io/redhat/redhat-operator-index:v4.5 --filter="mypackagename"
```

## Reports

| Report Type | Command | Description |
| ------ | ----- |  ------ |
| bundles | `audit bundle --index-image [OPTIONS]` | Audit all Bundles |
| packages | `audit packages --index-image [OPTIONS]` | Audit all Packages |
| channels | `audit channels --index-image [OPTIONS]` | Audit all Channels |

## Testdata

The samples in `testdata/samples` which are generated by running `make generate-samples`. Also, to run `make generate-testdata` to re-generate all reports in the testdata.

## FAQ

### How Audit works?

Following the steps performed by Audit. 

- Extract the database from the image informed
- Perform SQL queries to obtain the data from the index db
- Download and extract all bundles files by using the operator bundle path which is stored in the index db  
- Get the required data for the report from the operator bundle manifest files 
- Use the [operator-framework/api][of-api] to execute the bundle validator checks
- Use SDK tool to execute the Scorecard bundle checks
- Output a report providing the information obtained and processed. 

For some detailed information about its implementation check [here](docs/steps.md).

### What means UNKNOWN ?

If is not possible gathering the information, for example, when the Operator Bundle Path info is not in the index db then, audit will set the data as `UNKNOWN`. 

### What means NOT USED ?

If you see a column with this information than that means that the specific criteria is not useful or applied to none operator bundle of a package or the specific bundle itself.

### What are the images used to generate the full reports?

- OCP images: See [Understanding Operator catalogs](https://github.com/openshift/openshift-docs/blob/master/modules/olm-understanding-operator-catalog-images.adoc#understanding-operator-catalogs)
- Community operator image (`quay.io/operatorhubio/catalog:latest`): Its source is from [upstream-community-operators](https://github.com/operator-framework/community-operators/tree/master/upstream-community-operators)

### What are the reports in the testdata/backport? 

These reports were generated for we are able to identify the projects which are using the index image label `com.redhat.delivery.backport=true` and are distributed on 4.5. 

[of-api]: https://github.com/operator-framework/api
[scorecard-config]: https://github.com/operator-framework/operator-sdk/blob/v1.5.0/testdata/go/v3/memcached-operator/bundle/tests/scorecard/config.yaml
[operator-sdk]: https://github.com/operator-framework/operator-sdk
[audit-ep]: https://github.com/operator-framework/enhancements/blob/master/enhancements/audit-command.md