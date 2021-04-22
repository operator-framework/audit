# How Audit works?

Following the description of steps performed by Audit. 

## Extract the database from the image informed

Following an example to extract the sql database from the image manually currently: 

```sh
docker create --name rh-catalog registry.redhat.io/redhat/redhat-operator-index:v4.6 "yes"
docker cp rh-catalog:/database/index.db .
```
After that, the `index.db` file can be used by the tool which can gathering the required information via sql. Audit tool does the same. The image is extracted in the `output/` directory.

**NOTE** The above steps are base on the current index catalog format as a database. It might be changed for JSON format which should be used instead. The audit command just requires to support the latest index catalog format which in this case will be JSON. More info [Package representation and management in an index](https://github.com/operator-framework/enhancements/blob/master/enhancements/declarative-index-config.md).

## Perform SQL queries to obtain the data from the index db

Note that the first query which will be executed by the commands to get the bundles, or channels, or packages are build according to the flags. See the methods `Build<report-type>Query` in the `pkg/reports/<report-type>/data.go`. 

## Download and extract all bundles files by using the operator bundle path which is stored in the index db

The OLM index image does not store the operator bundle manifests. The operator bundle registry address can be found in the `bundlepath` entry from the`operatorbundle` table. 
 
Following an example with the manually steps to download an extract the bundle operator manifests only to let you know how to check and test it locally:

```
$ docker pull <bundlepath>
$ docker save <bundle image> > mybundle.tar
$ tar -xvf mybundle.tar 
$ tree
.
├── 696782e86a62476638704177b71dd37382864a1801866002cd6628d1a3eec4c0
│   ├── VERSION
│   ├── json
│   └── layer.tar
├── f6bbe84c78f5d0725d248207f92a23edfcbe66d3016f841018c03f933341dae3.json
├── manifest.json
└── mybundle.tar
$ 
``` 

Now, see that all manifests are in the `layer.tar`:

```
$ tar -xvf 696782e86a62476638704177b71dd37382864a1801866002cd6628d1a3eec4c0/layer.tar
$ tree
.
├── 696782e86a62476638704177b71dd37382864a1801866002cd6628d1a3eec4c0
│   ├── VERSION
│   ├── json
│   └── layer.tar
├── f6bbe84c78f5d0725d248207f92a23edfcbe66d3016f841018c03f933341dae3.json
├── manifest.json
├── manifests
│   ├── myopertor.v0.5.5.clusterserviceversion.yaml
│   ├── apps_v1alpha1_apimanager_crd.yaml
│   ├── capabilities_v1alpha1_api_crd.yaml
│   ├── capabilities_v1alpha1_binding_crd.yaml
│   ├── capabilities_v1alpha1_limit_crd.yaml
│   ├── capabilities_v1alpha1_mappingrule_crd.yaml
│   ├── capabilities_v1alpha1_metric_crd.yaml
│   ├── capabilities_v1alpha1_plan_crd.yaml
│   └── capabilities_v1alpha1_tenant_crd.yaml
├── metadata
│   └── annotations.yaml
├── mybundle.tar
└── root
    └── buildinfo
        ├── Dockerfile-myoperator-rhel7-operator-metadata-2.8.2-5
        └── content_manifests
            └── myopertor-bundle-container-2.8.2-5.json
```

For further information see `pkg/actions/run_validators.go`.

## Get the required data for the report from the operator bundle manifest files

See the `AddDataFromBundle` in `pkg/reports/bndles/columns.go`.

## Use the [operator-framework/api][of-api] to execute the bundle validator checks

Check its implementation in `pkg/actions/run_validators.go` for further information.

## Use SDK tool to execute the Scorecard bundle checks

For the SDK tool be able to run scorecard tests its configuration requires to be wrote in the bundle directory (e.g `tests/scorecard/config.yaml` see [here](https://github.com/operator-framework/operator-sdk/blob/master/testdata/go/v3/memcached-operator/bundle/tests/scorecard/config.yaml)). Operator bundles which are build with SDK will have the scorecard tests configured by default. (e.g see [here](https://github.com/operator-framework/community-operators/tree/master/community-operators/namespace-configuration-operator/1.0.1)).
 
Note that the bundles might have some specific tests implemented by them using the option to write custom tests. For further information check the [Writing Custom Scorecard Tests](https://sdk.operatorframework.io/docs/advanced-topics/scorecard/custom-tests/) documentation. In this way, if a bundle does not have the default scorecard test configured audit command will add it. Otherwise, the audit command by default will use what was provided instead. 

Now see how to use [SDK](https://github.com/operator-framework/operator-sdk) tool to check the bundles with scorecard:

```
 $ operator-sdk scorecard bundle --wait-time=120 --output=json
{
  "kind": "TestList",
  "apiVersion": "scorecard.operatorframework.io/v1alpha3",
  "items": [
    {
      "kind": "Test",
      "apiVersion": "scorecard.operatorframework.io/v1alpha3",
      "spec": {
        "image": "quay.io/operator-framework/scorecard-test:v1.4.0",
        "entrypoint": [
          "scorecard-test",
          "olm-spec-descriptors"
        ],
        "labels": {
          "suite": "olm",
          "test": "olm-spec-descriptors-test"
        }
      },
      "status": {
        "results": [
          {
            "name": "olm-spec-descriptors",
            "log": "Loaded ClusterServiceVersion: memcached-operator.v0.0.1\nLoaded 1 Custom Resources from alm-examples\n",
            "state": "fail",
            "errors": [
              "size does not have a spec descriptor"
            ],
            "suggestions": [
              "Add a spec descriptor for size"
            ]
          }
        ]
      }
    },
    {
      "kind": "Test",
      "apiVersion": "scorecard.operatorframework.io/v1alpha3",
      "spec": {
        "image": "quay.io/operator-framework/scorecard-test:v1.4.0",
        "entrypoint": [
          "scorecard-test",
          "olm-crds-have-resources"
        ],
        "labels": {
          "suite": "olm",
          "test": "olm-crds-have-resources-test"
        }
      },
      "status": {
        "results": [
          {
            "name": "olm-crds-have-resources",
            "log": "Loaded ClusterServiceVersion: memcached-operator.v0.0.1\n",
            "state": "fail",
            "errors": [
              "Owned CRDs do not have resources specified"
            ]
          }
        ]
      }
    },
    {
      "kind": "Test",
      "apiVersion": "scorecard.operatorframework.io/v1alpha3",
      "spec": {
        "image": "quay.io/operator-framework/scorecard-test:v1.4.0",
        "entrypoint": [
          "scorecard-test",
          "olm-crds-have-validation"
        ],
        "labels": {
          "suite": "olm",
          "test": "olm-crds-have-validation-test"
        }
      },
      "status": {
        "results": [
          {
            "name": "olm-crds-have-validation",
            "log": "Loaded 1 Custom ...,},}]\n",
            "state": "pass"
          }
        ]
      }
    },
    {
      "kind": "Test",
      "apiVersion": "scorecard.operatorframework.io/v1alpha3",
      "spec": {
        "image": "quay.io/operator-framework/scorecard-test:v1.4.0",
        "entrypoint": [
          "scorecard-test",
          "olm-bundle-validation"
        ],
        "labels": {
          "suite": "olm",
          "test": "olm-bundle-validation-test"
        }
      },
      "status": {
        "results": [
          {
            "name": "olm-bundle-validation",
            "log": "time=\"2021-02-23T19:32:41Z\" level=debug msg=\"Found manifests directory\" name=bundle-test\ntime=\"2021-02-23T19:32:41Z\" level=debug msg=\"Found metadata directory\" name=bundle-test\ntime=\"2021-02-23T19:32:41Z\" level=debug msg=\"Getting mediaType info from manifests directory\" name=bundle-test\ntime=\"2021-02-23T19:32:41Z\" level=info msg=\"Found annotations file\" name=bundle-test\ntime=\"2021-02-23T19:32:41Z\" level=info msg=\"Could not find optional dependencies file\" name=bundle-test\n",
            "state": "pass"
          }
        ]
      }
    },
    {
      "kind": "Test",
      "apiVersion": "scorecard.operatorframework.io/v1alpha3",
      "spec": {
        "image": "quay.io/operator-framework/scorecard-test:v1.4.0",
        "entrypoint": [
          "scorecard-test",
          "olm-status-descriptors"
        ],
        "labels": {
          "suite": "olm",
          "test": "olm-status-descriptors-test"
        }
      },
      "status": {
        "results": [
          {
            "name": "olm-status-descriptors",
            "log": "Loaded ClusterServiceVersion: memcached-operator.v0.0.1\nLoaded 1 Custom Resources from alm-examples\n",
            "state": "fail",
            "errors": [
              "memcacheds.cache.example.com does not have a status descriptor"
            ]
          }
        ]
      }
    },
    {
      "kind": "Test",
      "apiVersion": "scorecard.operatorframework.io/v1alpha3",
      "spec": {
        "image": "quay.io/operator-framework/scorecard-test:v1.4.0",
        "entrypoint": [
          "scorecard-test",
          "basic-check-spec"
        ],
        "labels": {
          "suite": "basic",
          "test": "basic-check-spec-test"
        }
      },
      "status": {
        "results": [
          {
            "name": "basic-check-spec",
            "state": "pass"
          }
        ]
      }
    }
  ]
}

```

To check its implementation see `pkg/actions/run_scorecard.go`

## Output a report providing the information obtained and processed. 

All reports have a method `writeXls()` and `writeJSON()`. Then, to write the `xls` format audit tool uses [excelize](https://github.com/360EntSecGroup-Skylar/excelize) lib. You cna check its [docs](https://pkg.go.dev/github.com/360EntSecGroup-Skylar/excelize/v2#readme-excelize) which has a lot of code examples. You can check then in `pkg/reports/<report-type>/report.go`.
