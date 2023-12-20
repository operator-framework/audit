~~[![Go Report Card](https://goreportcard.com/badge/github.com/camilamacedo86/audit)](https://goreportcard.com/report/github.com/camilamacedo86/audit)
[![Coverage Status](https://coveralls.io/repos/github/github.com/operator-framework/audit/badge.svg?branch=main)](https://coveralls.io/github/camilamacedo86/audit?branch=main)

---
# Audit

## Overview

The audit is an **experimental** analytic tool which uses the Operator Framework solutions. Its purpose is to obtain and report and aggregate data provided by checks and analyses done in the operator bundles, packages and channels from an index catalog image.

Note that the latest version of the reports generated for all images can be checked in [testdata/report](testdata/reports). The file names are create by using the kind/type of the report, image name and date. (E.g. `testdata/report/bundles_quay.io_operatorhubio_catalog_latest_2021-04-22.xlsx`).

For further information about its motivation see the [EP Audit command operation][audit-ep]. 

## Pre-requirements

- go 1.19 
- docker or podman
- tar and skopeo (to scrape Dockerfile from image layer oci dirs -- common to all Red Hat built operators)
- access to the registry where the index catalog and operator bundle images are distributed
- access to a Kubernetes cluster
- [operator-sdk][operator-sdk] installed >= `1.5.0

**NOTE** that you can run the reports without SDK and the cluster running with by using the flag `--disable-scorecard`. That is only required for the scorecard results.  

## Install binary:

Check the release binaries provided in the [release page](https://github.com/operator-framework/audit/releases).

## Install from the source

To get the project and install the binary:

```sh
$ git clone git@github.com:operator-framework/audit.git
$ cd audit
$ make install
```

Now, you can run `$ audit-tool --help` to check it out.

## Usage

### Ensure that you have access to pull the images

You may first need to run `docker login` or `podman login` to have access to the images.

#### With Podman

Per default the audit commands use docker for dealing with container images. If you wish to use podman instead

- either set the environment variable `export CONTAINER_ENGINE=podman` beforehand.
- or add `--container-engine=podman` to each command

```sh
export CONTAINER_ENGINE=podman
```

### Generating the reports

Now, you can audit all operator bundles of an image catalog with: 

```sh 
audit-tool index bundles --index-image=registry.redhat.io/redhat/redhat-operator-index:v4.7 
```

Then, this report will result in a JSON file with all data exctract from the index and the bundles. Note that audit
will download each bundle and extracted the info from them. Therefore, the reports available in the page [https://operator-framework.github.io/audit/](https://operator-framework.github.io/audit/)
are done using the sub-command `dashboard`. All custom reports requires the bundles report in jSON format
so that they can are able to gathering the data and process it accordingly.

### Options

Use the `--help` flag to check the options and the further information about its commands. Following an example:

```sh
$ audit-tool --help
The audit is an analytic tool which uses the Operator Framework solutions. Its purpose is to obtain and report and aggregate data provided by checks and analyses done in the operator bundles, packages and channels from an index catalog image.

Usage:
  audit-tool [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  dashboard   generate specific custom reports based on the audit JSONs output
  help        Help about any command
  index       audit index catalog image

Flags:
  -h, --help   help for audit-tool

Use "audit-tool [command] --help" for more information about a command.
...
```

### To have a faster result, you can filter using the package name

See that you can use the `--filter` --flag to filter the results by the package name:

```sh
audit-tool index [bundles] --index-image=registry.redhat.io/redhat/redhat-operator-index:v4.5 --filter="mypackagename"
```

### To run in dedicated environments

Use the flag `--server-mode` to generate the reports in dedicated environments. By using this flag option the images
which are downloaded will not be removed, allowing the reports to be generated faster after the first execution.

Also, ensure that you have enough space to store all images. Note that the default behavior is to remove them, when this option is not used.  

## Reports

### Base for all reports (audit index bundle)

The command `audit index bundle --index-image [OPTIONS]` will audit the image and bundles shipped on the index to extract all data.

### HTML reports 

To generate the reports such as you can find in [https://operator-framework.github.io/audit/](https://operator-framework.github.io/audit/) you
will need to have the bundles report (JSON one build with `audit index bundle --index-image` ) and you will use the `dashboard` commands, see:

```
$ audit-tool dashboard --help
generate specific custom reports based on the audit JSONs output

Usage:
audit-tool dashboard [command]

Available Commands:
deprecate-apis generates a custom report to check packages impact by k8s apis removal.
multiarch      generates a custom report based on defined criteria over Multiple Architectures
qa             it is an custom dashboard which generates a custom report based on defined criteria over some specific defined criteria over the quality of the packages
validator      generates a custom report based on the results filter by this validation informed

Flags:
-h, --help   help for dashboard
```

Example:

```sh
audit-tool dashboard deprecate-apis --file=testdata/report/bundles_quay.io_operatorhubio_catalog_latest_2021-04-22.json 
```

#### deprecate-apis:  

* By default, it only checks the bundles which are using APIs that were removed on OCP 4.9, and K8s 1.22
* You can use to check the potential impact on the catalog for APIs that were removed in 1.25 and 1.26 (in this case, we can only 
verify the Operator bundles which are asking permissions for those APIs. However, RBAC configurations does 
not require the versions of the APIs so that, we cannot know if the project is using the removed version or not)

#### multiarch:

This one will check the Operator bundles against multiple architecture configurations.
To know more see the [Operator Framework/API validator](https://github.com/operator-framework/api/blob/v0.17.1/pkg/validation/internal/multiarch.go)

**Note**: Check [here](https://operator-framework.github.io/audit/testdata/reports/redhat_redhat_operator_index/dashboards/multiarch_registry.redhat.io_redhat_redhat_operator_index_v4.11.html) example.

#### qa:

This option will create a report to check the projects against some quality aspects. The results of the 
checks done checked against the [validators][validator] in and [SDK scorcard][scorecard]  and are used to build this reports.

**Note**: Check [here](https://operator-framework.github.io/audit/testdata/reports/redhat_redhat_operator_index/dashboards/qa_registry.redhat.io_redhat_redhat_operator_index_v4.11.html) example.

#### validator

This option is useful if you are looking for to generate a report with all Operator bundles that fails
under some [validator][validator] or [SDK scorcard][scorecard] check.

## How the reports in the page are generated

See that you will find a directory `testdata`. Therefore, you can: 

- run `make generate-samples` just for test purpose and to generate `testdata/samples` 
- run `make generate-testdata` to re-generate all reports in the testdata
- run `make generate-all` which will run all reports and dashboards(html ones) as the index.html

### Index page

The `index.html` page is generated via `make generate-index`. 
It will aggregate in its results all dashboards found per image which are available in the testdata. 
To check it, see https://operator-framework.github.io/audit/ . 

## FAQ

### How Audit works?

Following the steps performed by Audit. 

- Extract the database from the image informed
- Perform SQL queries to obtain the data from the index db
- Download and extract all bundles files by using the operator bundle path which is stored in the index db  
- Get the required data for the report from the operator bundle manifest files 
- Use the [operator-framework/api][of-api] to execute the bundle validator checks
- Use SDK tool to execute the Scorecard bundle checks
- Output a JSON report providing the information obtained and processed. 

For some detailed information about its implementation check [here](docs/steps.md).

Then, the based JSON can be used to generated the other custom dashbaords.

**Example: (Multi-arch reports)**

They are generated by the command:

`audit-tool dashboard multiarch --file="bundle report - json file with all datat"`

The command will do:
- get the data from the JSON, which has all bundle info extracted from the index
- get all packages and head of channels
- run docker manifest inspect for each image used/defined in the CSV
- grab the info and the logic criteria as we do in the validator 
- Then, with all results, aduit build the report in HTML

See that we have the makefile targets that generate all reports: https://github.com/operator-framework/audit/blob/v0.2.0/Makefile#L105-L107

### What are the images used to generate the full reports?

- OCP images: See [Understanding Operator catalogs](https://github.com/openshift/openshift-docs/blob/master/modules/olm-understanding-operator-catalog-images.adoc#understanding-operator-catalogs)
- Community operator image (`quay.io/operatorhubio/catalog:latest`): Its source is from [upstream-community-operators](https://github.com/operator-framework/community-operators/tree/master/upstream-community-operators)

### What is in the hack/special-needs 

In this directory we have been storing some scripts that help us to generate specific 
special needs reports that would not fit under the sub-command or that could one day be
improved to become a sub-command. 

### What is in the hack directory?

All scripts to automate the reports generated in the page for example are in the hack
directory. 

### What are the steps to generate the pages?

- Check pre-requirements

```
$ operator-sdk version (see if you have SDK locally it will be required for the scorecard checks)
$ kind create cluster (ensure that you have a cluster up and running it will also required for scarecard checks)
```

- Login in the registry and run `make generate-all`

```shell
docker login https://registry.redhat.io
make generate-all
```

**NOTE** If something fails you can check what failed and just call directory
the scripts for what is missing to acomplished the goal. You can look at
the Makefile to know how to do manually the calls. 

### Release Process

Only creates and push a new tag then, the github actions will build and 
add the artefacts in the release page. 

[of-api]: https://github.com/operator-framework/api
[scorecard-config]: https://github.com/operator-framework/operator-sdk/blob/v1.5.0/testdata/go/v3/memcached-operator/bundle/tests/scorecard/config.yaml
[operator-sdk]: https://github.com/operator-framework/operator-sdk
[audit-ep]: https://github.com/operator-framework/enhancements/blob/master/enhancements/audit-command.md
[validator]: https://github.com/operator-framework/api/blob/v0.17.1/pkg/validation/validation.go#L66-L85
[scorecard]: https://sdk.operatorframework.io/docs/testing-operators/scorecard/