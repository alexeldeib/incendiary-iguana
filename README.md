Incendiary Iguana
---

![Github Actions Status](https://github.com/alexeldeib/incendiary-iguana/workflows/build%20and%20test/badge.svg)


[![Azure DevOps Status](https://dev.azure.com/alexeldeib/incendiary-iguana/_apis/build/status/alexeldeib.incendiary-iguana?branchName=master)](https://dev.azure.com/alexeldeib/incendiary-iguana/_build/latest?definitionId=2&branchName=master)

# Motivation

This project aims to offer Custom Resource Definitions and a declarative layer for resource management on Azure. It offers three tools to achieve this goal: 
- a set of clients which properly handle idempotency for Azure resources.
- a set of Kubernetes custom controllers to reconcile Azure types.
- a cli which takes the same CRDs as input and parallelizes asynchronous reconciliation much in the same way as the custom controllers.

It should be possible to build higher level orchestration tooling or arrangements of infrastructure simply by defining the appropriate custom resources and calling `Ensure()` on the resource-specific clients.
