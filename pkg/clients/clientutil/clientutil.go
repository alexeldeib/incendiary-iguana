/*
Copyright 2019 Alexander Eldeib.
*/

package clientutil

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// Any executes an array of functions in order and returns true if any of them returns true. Returns false otherwise.
func Any(funcs []func() bool) bool {
	for _, f := range funcs {
		if f() {
			return true
		}
	}
	return false
}

// Initialize takes equal-length arrays of detector and remediator functions. If a given detector returns true, Initialize calls the corresponding remediator.
func Initialize(detectors []func() bool, remediators []func()) {
	for idx, f := range detectors {
		if f() {
			remediators[idx]()
		}
	}
}

// https://github.com/kubernetes-sigs/cluster-api-provider-azure/blob/60b7c6058550ae694935fb03103460a2efa4e332/pkg/cloud/azure/services/virtualmachines/virtualmachines.go#L215
// GenerateRandomString generates a random string of lenth n, panicking on failure.
func GenerateRandomString(n int) string {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		fmt.Printf("error in generate random: %+#v", err.Error())
	}
	return base64.URLEncoding.EncodeToString(b) //, err
}

// StringPtrChanged returns true if the new user-defined value for a string matches the existing value.
// If the resource does not exist, old will be nil, which should be treated as requiring update.
func StringPtrChanged(old *string, new *string) func() bool {
	return func() bool {
		return new != nil && (old == nil || *old != *new)
	}
}
