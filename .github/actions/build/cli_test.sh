#!/bin/bash

set -eux

# setup
kind create cluster
export KUBECONFIG="$(kind get kubeconfig-path --name="kind")"

make cli-test

