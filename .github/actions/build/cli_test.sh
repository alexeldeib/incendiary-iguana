#!/bin/bash

set -eux

# setup
kind create cluster
export KUBECONFIG=$(kind get kubeconfig)
export USE_EXISTING_CLUSTER="true"

make cli-test

