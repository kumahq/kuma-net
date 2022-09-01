TOP := $(shell pwd)
TOOLS_DIR = $(HOME)/bin
CURL_PATH ?= curl
CURL_DOWNLOAD := $(CURL_PATH) --location --fail --progress-bar

KIND_CLUSTER_NAME ?= kuma
KIND_KUBECONFIG_DIR := $(HOME)/.kube
KIND_KUBECONFIG := $(KIND_KUBECONFIG_DIR)/kind-$(KIND_CLUSTER_NAME)-config
HELM_PATH := $(TOOLS_DIR)/helm
KIND_PATH := $(TOOLS_DIR)/kind
KUBECTL_PATH := $(TOOLS_DIR)/kubectl
HELM_VERSION ?= v3.8.2
KIND_VERSION ?= v0.14.0
KUBECTL_VERSION ?= v1.23.5
KUMA_VERSION ?= 1.8.0

# wrk test configuration
WRK_CONN := 100
WRK_THREAD := 12
WRK_DURATION := 60
export PATH := $(TOOLS_DIR):$(PATH)
CONFIG_PATH := single-node
ifdef USE_MULTI_NODE
	CONFIG_PATH = multi-node
endif

UNAME_OS := $(shell uname -s)
UNAME_ARCH := $(shell uname -m)
ARCH := UNAME_ARCH
ifeq ($(UNAME_ARCH),x86_64)
	ARCH = amd64
endif

OS := UNAME_OS
ifeq ($(UNAME_OS),Linux)
	OS = linux
else ifeq ($(UNAME_OS),Darwin)
	OS = darwin
endif

.PHONY: install/all
install/all: install/kind install/helm install/kubectl

.PHONY: install/kind
install/kind: ## Bootstrap: Install KIND (Kubernetes in Docker)
	# see https://kind.sigs.k8s.io/docs/user/quick-start/#installation
	@if [ -e $(KIND_PATH) ]; then echo "Kind $$( $(KIND_PATH) version ) is already installed at $(KIND_PATH)" ; fi
	@if [ ! -e $(KIND_PATH) ]; then \
		echo "Installing Kind $(CI_KIND_VERSION) ..." \
		&& set -x \
		&& $(CURL_DOWNLOAD) -o kind https://github.com/kubernetes-sigs/kind/releases/download/$(KIND_VERSION)/kind-$(OS)-$(ARCH) \
		&& chmod +x kind \
		&& mkdir -p $(TOOLS_DIR) \
		&& mv kind $(KIND_PATH) \
		&& set +x \
		&& echo "Kind $(KIND_VERSION) has been installed at $(KIND_PATH)" ; fi

install/kubectl: ## Bootstrap: Install kubectl
	# see https://kubernetes.io/docs/tasks/tools/install-kubectl/#install-kubectl-binary-with-curl-on-linux
	@if [ -e $(KUBECTL_PATH) ]; then echo "Kubectl $$( $(KUBECTL_PATH) version ) is already installed at $(KUBECTL_PATH)" ; fi
	@if [ ! -e $(KUBECTL_PATH) ]; then \
		echo "Installing Kubectl $(KUBECTL_VERSION) ..." \
		&& set -x \
		&& $(CURL_DOWNLOAD) -O https://storage.googleapis.com/kubernetes-release/release/$(KUBECTL_VERSION)/bin/$(OS)/$(ARCH)/kubectl \
		&& chmod +x kubectl \
		&& mkdir -p $(TOOLS_DIR) \
		&& mv kubectl $(KUBECTL_PATH) \
		&& set +x \
		&& echo "Kubectl $(KUBECTL_VERSION) has been installed at $(KUBECTL_PATH)" ; fi

.PHONY: install/helm
install/helm: ## Bootstrap: Install Helm
	$(CURL_DOWNLOAD) https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | \
		env HELM_INSTALL_DIR=$(TOOLS_DIR) DESIRED_VERSION=$(HELM_VERSION) USE_SUDO=false bash

kind/start:
	@$(KIND_PATH) get clusters | grep $(KIND_CLUSTER_NAME) >/dev/null 2>&1 && echo "Kind cluster already running." && exit 0 || \
		($(KIND_PATH) create cluster \
			--name "$(KIND_CLUSTER_NAME)" \
			--config "$(TOP)/$(CONFIG_PATH)/kind-config.yaml" \
			--kubeconfig $(KIND_KUBECONFIG) \
			--quiet --wait 120s && \
		KUBECONFIG=$(KIND_KUBECONFIG) $(KUBECTL_PATH) apply -f $(TOP)/$(CONFIG_PATH)/calico.yaml && \
		KUBECONFIG=$(KIND_KUBECONFIG) $(KUBECTL_PATH) scale deployment --replicas 1 coredns --namespace kube-system && \
		make kind/wait)
	@echo
	@echo '>>> You need to manually run the following command in your shell: >>>'
	@echo
	@echo export KUBECONFIG="$(KIND_KUBECONFIG)"
	@echo
	@echo '<<< ------------------------------------------------------------- <<<'
	@echo

kind/stop:
	@$(KIND_PATH) delete cluster --name $(KIND_CLUSTER_NAME)
	@rm -f $(KIND_KUBECONFIG_DIR)/$(KIND_KUBECONFIG)

kind/clean:
	KUBECONFIG=$(KIND_KUBECONFIG) $(KUBECTL_PATH) delete -f $(TOP)/$(CONFIG_PATH)/wrk.yaml && \
	KUBECONFIG=$(KIND_KUBECONFIG) $(KUBECTL_PATH) delete -f $(TOP)/$(CONFIG_PATH)/http-echo.yaml && \
	$(HELM_PATH) uninstall kuma --namespace kuma-system

kind/wait:
	until \
		KUBECONFIG=$(KIND_KUBECONFIG) $(KUBECTL_PATH) wait -n kube-system --timeout=5s --for condition=Ready --all pods ; \
	do echo "Waiting for the cluster to come up" && sleep 1; done

kind/deploy/kuma/wait:
	until \
		KUBECONFIG=$(KIND_KUBECONFIG) $(KUBECTL_PATH) wait -n kuma-perf --timeout=60s --for condition=Ready --all pods ; \
	do echo "Waiting for the cluster to come up" && sleep 1; done

kind/deploy/ebpf: helm/add/kuma
	$(HELM_PATH) upgrade --install --create-namespace --namespace kuma-system --set "experimental.ebpf.enabled=true" --set "cni.experimental.imageEbpf.registry=docker.io/merbridge" --set "cni.enabled=true" --set "cni.chained=true" --set "cni.netDir=/etc/cni/net.d" --set "cni.binDir=/opt/cni/bin" --set "cni.confName=10-calico.conflist" kuma kuma/kuma --version $(KUMA_VERSION) && \
	KUBECONFIG=$(KIND_KUBECONFIG) $(KUBECTL_PATH) apply -f $(TOP)/$(CONFIG_PATH)/http-echo.yaml && \
	KUBECONFIG=$(KIND_KUBECONFIG) $(KUBECTL_PATH) apply -f $(TOP)/$(CONFIG_PATH)/wrk.yaml && \
	make kind/deploy/kuma/wait

kind/deploy/iptables: helm/add/kuma
	$(HELM_PATH) upgrade --install --create-namespace --namespace kuma-system --set "cni.enabled=true" --set "cni.chained=true" --set "cni.netDir=/etc/cni/net.d" --set "cni.binDir=/opt/cni/bin" --set "cni.confName=10-calico.conflist" kuma kuma/kuma --version $(KUMA_VERSION) && \
	KUBECONFIG=$(KIND_KUBECONFIG) $(KUBECTL_PATH) apply -f $(TOP)/$(CONFIG_PATH)/http-echo.yaml && \
	KUBECONFIG=$(KIND_KUBECONFIG) $(KUBECTL_PATH) apply -f $(TOP)/$(CONFIG_PATH)/wrk.yaml && \
	make kind/deploy/kuma/wait

run/perf:
	CLUSTER_IP=$(KUBECONFIG=$(KIND_KUBECONFIG) $(KUBECTL_PATH) get services/http-echo -n kuma-demo -o go-template='{{(.spec.clusterIP)}}') && \
	KUBECONFIG=$(KIND_KUBECONFIG) $(KUBECTL_PATH) exec -it deployment/wrk -n kuma-perf -c wrk -- wrk -c$(WRK_CONN) -t$(WRK_THREAD) -d$(WRK_DURATION)s --latency http://$(CLUSTER_IP):5000/

helm/add/kuma:
	$(HELM_PATH) repo add kuma https://kumahq.github.io/charts && \
	$(HELM_PATH) repo update