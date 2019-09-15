
# Image URL to use all building/pushing image targets
IMG ?= alexeldeib/controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"
BAZEL_OPTIONS ?= --local_cpu_resources HOST_CPUS-2  --local_ram_resources HOST_RAM*.50
BAZEL_TEST_OPTIONS ?= $(BAZEL_OPTIONS) --test_output all --test_summary detailed 
DEBUG_TEST_OPTIONS = $(BAZEL_TEST_OPTIONS) --sandbox_debug
GO_TEST_OPTIONS ?= ./api/... ./controllers/... -coverprofile cover.out

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manifests manager

# Run tests
# alternatively ginkgo -v ./...
cli-test: # fmt vet
	# n.b., should set $env:AZURE_AUTH_LOCATION first.
	# $$env:AZURE_AUTH_LOCATION="$(pwd)/sp.json" => can't get this to work on windows
	ginkgo -randomizeSuites -stream --slowSpecThreshold=180 -v -r ./cmd  || exit 1

manager-test:
	ginkgo -randomizeSuites -stream --slowSpecThreshold=180 -v -r ./controllers || exit 1

ci-manager: manifests ci-fmt ci-vet # lint 
	go1.13 build -gcflags '-N -l' -o manager.exe main.go || exit 1

# Build manager binary
manager: manifests fmt vet # lint 
	go build -gcflags '-N -l' -o manager.exe main.go

cli: manifests fmt vet # lint 
	go build -gcflags '-N -l' -o tinker.exe ./cmd

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kubectl apply -f config/crd/bases

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	kubectl apply -f config/crd/bases
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen generate
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./api/...;./controllers/..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...
	goimports -w .

# Run go vet against code
vet:
	go vet ./...

# Run go fmt against code
ci-fmt:
	go1.13 fmt ./... || exit 1
	goimports -w . || exit 1

# Run go vet against code
ci-vet:
	go1.13 vet ./... || exit 1

# -j flag should be set to NUM_CPU_CORES - 1 or less, and be an integer. It defaults to 8 if removed.
lint:
	golangci-lint run --fix -j=2

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile=./hack/boilerplate.go.txt paths=./api/...

# Build the docker image
docker-build: #test
	docker build . -t ${IMG}
	@echo "updating kustomize image patch file for manager resource"
	sed -i'' -e 's@image: .*@image: '"${IMG}"'@' ./config/default/manager_image_patch.yaml

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.1
CONTROLLER_GEN=controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

deps:
	bazel run gazelle -- update-repos -from_file go.mod

gazelle:
	bazel run gazelle -- update

bazel-bin: fmt gazelle 
	rm -f ./bin/manager
	bazel build incendiary-iguana --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64
	cp ./bazel-bin/linux_amd64_static_pure_stripped/incendiary-iguana ./bin/manager

bazel-exe: fmt gazelle
	rm -f ./bin/incendiary-iguana.exe
	bazel build incendiary-iguana --platforms=@io_bazel_rules_go//go/toolchain:windows_amd64
	cp ./bazel-bin/windows_amd64_static_pure_stripped/incendiary-iguana.exe ./bin/manager.exe

bazel-image: fmt gazelle
	bazel run image -- --norun

bazel-test: fmt gazelle
ifeq (,$(DEBUG))
	bazel test ... $(BAZEL_TEST_OPTIONS)
else
	bazel test ... $(DEBUG_TEST_OPTIONS)
endif

publish: bazel-image
	bazel run publish --host_force_python=PY2
