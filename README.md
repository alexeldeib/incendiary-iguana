Incendiary Iguana
---

![Github Actions Status](https://github.com/alexeldeib/incendiary-iguana/workflows/build%20and%20test/badge.svg)
[![Azure DevOps Status](https://dev.azure.com/alexeldeib/incendiary-iguana/_apis/build/status/alexeldeib.incendiary-iguana?branchName=master)](https://dev.azure.com/alexeldeib/incendiary-iguana/_build/latest?definitionId=2&branchName=master)


[![GitHub release](https://img.shields.io/github/release/alexeldeib/incendiary-iguana.svg)](https://GitHub.com/alexeldeib/incendiary-iguana/releases/)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexeldeib/incendiary-iguana)](https://goreportcard.com/report/github.com/alexeldeib/incendiary-iguana)

# Motivation

This project aims to offer Custom Resource Definitions and a declarative layer for resource management on Azure. It offers three tools to achieve this goal: 
- a set of clients which properly handle idempotency for Azure resources.
- a set of Kubernetes custom controllers to reconcile Azure types.
- a cli which takes the same CRDs as input and parallelizes asynchronous reconciliation much in the same way as the custom controllers.

It should be possible to build higher level orchestration tooling or arrangements of infrastructure simply by defining the appropriate custom resources and calling `Ensure()` on the resource-specific clients. Kubernetes controllers only require a simple client interface: essentially reconcile (idempotent create or update) and delete. `Ensure()` in these client packages wraps the methods exposed by Azure REST APIs to produce proper idempotency which is expected and required from Kubernetes controllers.

The Azure SDKs often put on the facade of idempotency -- they expose a CreateOrUpdate method and accept a full specification of the object on every call. In reality, many of the backing APIs will return validation errors (e.g. VM change to custom data, ssh keys), or in some cases will perform delete/recreate operations with downtime (e.g. Redis). Changing resource locations for almost any resource is an immediate validation failure. All of this behavior is undesirable for reconciliation in the context of a Kubernetes controller, or should at least be handled.

On top of this, when we map Kubernetes resources to Azure resources we must be certain that we properly track all resources we are responsible for managing in Azure: if a user creates a CRD for a resource group with for subscription A and location X, and then changes the same Kubernetes object to be in subscription B, we should manage the cleanup of the original resource group or provide validation warnings to force the user to make a deliberate decision.
