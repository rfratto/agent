include ../../Makefile.common

DEEPCOPY_TARGETS := apis/monitoring/v1/zz_generated.deepcopy.go
CLIENT_TARGET    := client/versioned/clientset.go
LISTER_TARGETS   := client/listers/monitoring/v1/grafanaagent.go
INFORMER_TARGETS := client/informers/externalversions/monitoring/v1/grafanaagent.go

generate: $(DEEPCOPY_TARGETS) $(CLIENT_TARGET) $(LISTER_TARGETS) $(INFORMER_TARGETS) generate-crds

# Each job that can run in a container is defined twice:
# once if BUILD_IN_CONTAINER is set, which will cause it
# to run in the container, and once if BUILD_IN_CONTAINER is
# not set.
ifeq ($(BUILD_IN_CONTAINER),true)

CONTAINER_JOBS := $(DEEPCOPY_TARGETS) $(CLIENT_TARGET) $(LISTER_TARGETS) $(INFORMER_TARGETS) generate-crds
$(CONTAINER_JOBS): build-image
	$(call run_in_container,$@)

else # End run_in_container wrappers

$(DEEPCOPY_TARGETS):
	cd ./apis/monitoring/v1 && controller-gen object paths=.

$(CLIENT_TARGET):
	client-gen \
		--go-header-file .header --v=1 --logtostderr \
		--input-base "$(GO_PKG)/pkg/operator/apis" \
		--clientset-name "versioned" \
		--input "monitoring/v1" \
		--output-package "$(GO_PKG)/pkg/operator/client"

$(LISTER_TARGETS):
	lister-gen \
		--go-header-file .header --v=1 --logtostderr \
		--input-dirs "$(GO_PKG)/pkg/operator/apis/monitoring/v1" \
		--output-package "$(GO_PKG)/pkg/operator/client/listers"

$(INFORMER_TARGETS): $(LISTER_TARGETS) $(CLIENT_TARGET)
	informer-gen \
		--go-header-file .header --v=1 --logtostderr \
		--versioned-clientset-package "$(GO_PKG)/pkg/operator/client/versioned" \
		--listers-package "$(GO_PKG)/pkg/operator/client/listers" \
		--input-dirs "$(GO_PKG)/pkg/operator/apis/monitoring/v1" \
		--output-package "$(GO_PKG)/pkg/operator/client/informers"

generate-crds:
	cd ./apis/monitoring/v1 && controller-gen crd:crdVersions=v1 paths=. output:crd:dir=/src/agent/production/operator/crds/

endif # End job definitions

.PHONY: generate generate-crds
