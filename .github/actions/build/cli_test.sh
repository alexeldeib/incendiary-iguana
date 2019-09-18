#!/bin/bash

set -eux

# setup
kind create cluster
export KUBECONFIG=$(kind get kubeconfig --cluster=kind)
export USE_EXISTING_CLUSTER="true"

make cli-test

