package pkg

import (
	"log"
	"os/exec"
	"reflect"
	"testing"
)

func TestRunSkopeoLayerExtractSuite(t *testing.T) {
	tests := []struct {
		name               string
		imageRef           string
		expectedDockerfile Dockerfile
	}{
		{
			name:     "TestQuayOperatorBundle",
			imageRef: "docker://registry.redhat.io/quay/quay-operator-bundle@sha256:a97a63899d23e23d039ea36bd575c018d7b6295b7942b15a8bded52f09736bda",
			expectedDockerfile: Dockerfile{
				Commands: []DockerfileCommand{
					{CommandType: "FROM", Value: "scratch"},
					{CommandType: "LABEL", Value: `com.redhat.delivery.operator.bundle=true`},
					{CommandType: "LABEL", Value: `com.redhat.delivery.openshift.ocp.versions="v4.8"`},
					{CommandType: "LABEL", Value: `com.redhat.openshift.versions="v4.8"`},
					{CommandType: "LABEL", Value: `com.redhat.delivery.backport=false`},
					{CommandType: "LABEL", Value: `operators.operatorframework.io.bundle.mediatype.v1=registry+v1`},
					{CommandType: "LABEL", Value: `operators.operatorframework.io.bundle.manifests.v1=manifests/`},
					{CommandType: "LABEL", Value: `operators.operatorframework.io.bundle.metadata.v1=metadata/`},
					{CommandType: "LABEL", Value: `operators.operatorframework.io.bundle.package.v1=quay-operator`},
					{CommandType: "LABEL", Value: `operators.operatorframework.io.bundle.channels.v1=stable-3.8`},
					{CommandType: "LABEL", Value: `operators.operatorframework.io.bundle.channel.default.v1=stable-3.8`},
					{CommandType: "LABEL", Value: `com.redhat.component="quay-operator-bundle-container"`},
					{CommandType: "LABEL", Value: `name="quay/quay-operator-bundle"`},
					{CommandType: "LABEL", Value: `summary="Quay Operator bundle container image"`},
					{CommandType: "LABEL", Value: `description="Operator bundle for Quay Operator"`},
					{CommandType: "LABEL", Value: `maintainer="Red Hat <support@redhat.com>"`},
					{CommandType: "LABEL", Value: `version=v3.8.11`},
					{CommandType: "LABEL", Value: `io.k8s.display-name="Red Hat Quay Operator Bundle"`},
					{CommandType: "LABEL", Value: `io.openshift.tags="quay"`},
					{CommandType: "COPY", Value: `bundle/manifests/*.yaml /manifests/`},
					{CommandType: "COPY", Value: `bundle/manifests/metadata/annotations.yaml /metadata/annotations.yaml`},
					{CommandType: "LABEL", Value: `release=20`},
					{CommandType: "ADD", Value: `quay-operator-bundle-container-v3.8.11-20.json /root/buildinfo/content_manifests/quay-operator-bundle-container-v3.8.11-20.json`},
					{CommandType: "LABEL", Value: `"com.redhat.license_terms"="https://www.redhat.com/agreements" "distribution-scope"="public" "vendor"="Red Hat, Inc." "build-date"="2023-08-07T23:21:46" "architecture"="x86_64" "vcs-type"="git" "vcs-ref"="f6eb857b8bd8768d51a311bc274f53ce7856ae04" "io.k8s.description"="Operator bundle for Quay Operator" "url"="https://access.redhat.com/containers/#/registry.access.redhat.com/quay/quay-operator-bundle/images/v3.8.11-20"`},
				},
			},
		},
		{
			name:     "Test3ScaleOperatorBundle",
			imageRef: "docker://registry.redhat.io/3scale-mas/3scale-rhel7-operator@sha256:0a6673eae2f0e8d95b919b0243e44d2c0383d13e2e616ac8d3f80742d496d292",
			expectedDockerfile: Dockerfile{
				Commands: []DockerfileCommand{
					{CommandType: "FROM", Value: "registry.redhat.io/devtools/go-toolset-rhel7:1.19.13-1.1697640714 AS builder"},
					{CommandType: "ENV", Value: `PROJECT_NAME="3scale-operator"`},
					{CommandType: "ENV", Value: `OUTPUT_DIR="/tmp/_output"`},
					{CommandType: "ENV", Value: `BINARY_NAME="manager"`},
					{CommandType: "ENV", Value: `BUILD_PATH="${REMOTE_SOURCE_DIR}/app"`},
					{CommandType: "WORKDIR", Value: `${BUILD_PATH}`},
					{CommandType: "COPY", Value: `$REMOTE_SOURCE $REMOTE_SOURCE_DIR`},
					{CommandType: "ADD", Value: `patches /tmp/patches`},
					{CommandType: "RUN", Value: `find /tmp/patches -type f -name '*.patch' -print0 | sort --zero-terminated | xargs -t -0 -n 1 patch --force -p1`},
					{CommandType: "USER", Value: `root`},
					{CommandType: "RUN", Value: `mkdir -p ${OUTPUT_DIR}`},
					{CommandType: "RUN", Value: `echo "build path: ${BUILD_PATH}"`},
					{CommandType: "RUN", Value: `echo "output path: ${OUTPUT_DIR}"`},
					{CommandType: "RUN", Value: `scl enable go-toolset-1.19 "GOOS=linux GOARCH=$(scl enable go-toolset-1.19 'go env GOARCH') CGO_ENABLED=0 GO111MODULE=on go build -o ${OUTPUT_DIR}/${BINARY_NAME} main.go"`},
					{CommandType: "RUN", Value: `mkdir ${OUTPUT_DIR}/licenses/`},
					{CommandType: "RUN", Value: `cp "./licenses.xml" "${OUTPUT_DIR}/licenses/"`},
					{CommandType: "FROM", Value: `registry.redhat.io/ubi7/ubi-minimal:7.9-1196`},
					{CommandType: "LABEL", Value: `com.redhat.component="3scale-mas-operator-container" name="3scale-mas/3scale-rhel7-operator" version="1.17.0" summary="3scale Operator container image" description="Operator provides a way to install a 3scale API Management and ability to define 3scale API definitions." io.k8s.display-name="3scale Operator" io.openshift.expose-services="" io.openshift.tags="3scale, 3scale-amp, api" upstream_repo="https://github.com/3scale/3scale-operator" upstream_ref="a5d72cc78a29ce38f3c60761cd7d2afff0727feb" maintainer="eastizle@redhat.com"`},
					{CommandType: "ENV", Value: `OPERATOR_BINARY_NAME="manager" USER_UID=1001 USER_NAME=3scale-operator`},
					{CommandType: "USER", Value: `root`},
					{CommandType: "COPY", Value: `--from=builder /tmp/_output/${OPERATOR_BINARY_NAME} /`},
					{CommandType: "RUN", Value: `chown ${USER_UID} /${OPERATOR_BINARY_NAME}`},
					{CommandType: "ENV", Value: `LICENSES_DIR="/root/licenses/3scale-operator/"`},
					{CommandType: "RUN", Value: `mkdir -p ${LICENSES_DIR}`},
					{CommandType: "COPY", Value: `--from=builder /tmp/_output/licenses/licenses.xml ${LICENSES_DIR}`},
					{CommandType: "RUN", Value: `chown ${USER_UID} ${LICENSES_DIR}/licenses.xml`},
					{CommandType: "ENTRYPOINT", Value: `["/manager"]`},
					{CommandType: "USER", Value: `${USER_UID}`},
				},
			},
		},
		// Additional tests can be added here
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//t.Parallel() // Do not enable parallel execution, these tests share skopeo and filesystem resources

			result, err := RunSkopeoLayerExtract(tt.imageRef)
			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					log.Printf("Skopeo command failed with exit code: %d", exitError.ExitCode())
					log.Printf("Stderr: %s", exitError.Stderr)
				}
				t.Fatalf("RunSkopeoLayerExtract returned an error: %v", err)
			}

			if !reflect.DeepEqual(result, tt.expectedDockerfile) {
				t.Errorf("RunSkopeoLayerExtract() = %v, want %v", result, tt.expectedDockerfile)
			}
		})
	}
}
