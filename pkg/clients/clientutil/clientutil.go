/*
Copyright 2019 Alexander Eldeib.
*/

package clientutil

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
