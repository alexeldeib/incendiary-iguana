# Build the manager binary
FROM golang:1.13

ARG OS=linux
ARG ARCH=amd64

ENV PATH "$PATH:/usr/local/kubebuilder/bin"
ENV KUBEBUILDER_CONTROLPLANE_START_TIMEOUT "60s"
ENV GO111MODULE "on"

WORKDIR /workspace

RUN apt update -y 
RUN go get golang.org/x/tools/cmd/goimports
RUN mkdir -p /usr/local/kubebuilder
RUN export OS=${OS:-linux} && export ARCH=${ARCH:-amd64} \
  && curl -sL https://go.kubebuilder.io/dl/2.0.0/${OS}/${ARCH} | tar -xz -C /tmp/ \
  && mv /tmp/kubebuilder_2.0.0_${OS}_${ARCH}/* /usr/local/kubebuilder/bin

RUN go get golang.org/x/tools/cmd/goimports
RUN go get github.com/onsi/ginkgo/ginkgo
RUN go get sigs.k8s.io/kustomize/v3/cmd/kustomize@master
RUN go get sigs.k8s.io/kind@v0.5.1
RUN go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.1

RUN git clone https://github.com/alexeldeib/incendiary-iguana.git
RUN cd incendiary-iguana && go mod download && cd .. && rm -rf ./incendiary-iguana

RUN mkdir /scripts

ADD cli_build.sh /scripts
ADD cli_test.sh /scripts
ADD mgr_build.sh /scripts
ADD mgr_test.sh /scripts

RUN chmod +x /scripts/*

RUN apt update -y && apt -y install apt-transport-https ca-certificates curl software-properties-common
RUN curl -fsSL https://download.docker.com/linux/debian/gpg | apt-key add -
RUN add-apt-repository \
  "deb [arch=amd64] https://download.docker.com/linux/debian \
  buster \
  stable"
RUN apt update -y && apt -y install docker-ce=18.06.2~ce~3-0~debian

# RUN go get github.com/go-delve/delve/cmd/dlv@v1.3.0
