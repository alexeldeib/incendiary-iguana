#!/bin/bash

set -eux

# setup
kind create cluster
export KUBECONFIG=$(kind get kubeconfig --name=kind)
export USE_EXISTING_CLUSTER="true"

make manager-test

