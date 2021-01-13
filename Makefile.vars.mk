IMG_TAG ?= latest

BIN_FILENAME ?= $(PROJECT_ROOT_DIR)/k8up
TESTBIN_DIR ?= $(PROJECT_ROOT_DIR)/testbin/bin

CRD_FILE ?= k8up-crd.yaml
CRD_FILE_LEGACY ?= k8up-crd-legacy.yaml
CRD_ROOT_DIR ?= config/crd/apiextensions.k8s.io
CRD_SPEC_VERSION ?= v1

KIND_VERSION ?= 0.9.0
KIND_NODE_VERSION ?= v1.20.0
KIND ?= $(TESTBIN_DIR)/kind

ENABLE_LEADER_ELECTION ?= false

KIND_KUBECONFIG ?= $(TESTBIN_DIR)/kind-kubeconfig-$(KIND_NODE_VERSION)
KIND_CLUSTER ?= k8up-$(KIND_NODE_VERSION)
KIND_KUBECTL_ARGS ?= --validate=true

E2E_TAG ?= e2e_$(shell sha1sum $(BIN_FILENAME) | cut -b-8)
E2E_REPO ?= local.dev/k8up/e2e
E2E_IMG = $(E2E_REPO):$(E2E_TAG)

KUSTOMIZE ?= go run sigs.k8s.io/kustomize/kustomize/v3

# Image URL to use all building/pushing image targets
DOCKER_IMG ?= docker.io/vshn/k8up:$(IMG_TAG)
QUAY_IMG ?= quay.io/vshn/k8up:$(IMG_TAG)

testbin_created = $(TESTBIN_DIR)/.created
