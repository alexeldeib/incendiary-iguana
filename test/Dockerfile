# Build the manager binary
FROM golang:1.13

ARG OS=linux
ARG ARCH=amd64

ENV PATH "$PATH:/usr/local/kubebuilder/bin"
ENV KUBEBUILDER_CONTROLPLANE_START_TIMEOUT "60s"
ENV GO111MODULE "on"
ENV AZURE_AUTH_LOCATION "/workspace/sp.json"

WORKDIR /workspace

RUN apt update -y 
RUN go get golang.org/x/tools/cmd/goimports
RUN mkdir -p /usr/local/kubebuilder
RUN curl -sL https://go.kubebuilder.io/dl/2.0.0/${OS}/${ARCH} | tar -xz -C /tmp/ \
  && mv /tmp/kubebuilder_2.0.0_${OS}_${ARCH}/* /usr/local/kubebuilder 

RUN go get golang.org/x/tools/cmd/goimports
RUN go get github.com/onsi/ginkgo/ginkgo
RUN go get sigs.k8s.io/kustomize/v3/cmd/kustomize@master

ADD . .

RUN go mod download
