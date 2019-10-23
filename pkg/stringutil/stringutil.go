/*
Copyright 2019 Alexander Eldeib.
*/

package stringutil

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	mrand "math/rand"
	"strings"
	"time"
)

const safeBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const lowerBytes = "abcdefghijklmnopqrstuvwxyz0123456789"

const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// // TODO(ace): probably move this to a more descriptive package
// // Any executes an array of functions in order and returns true if any of them returns true. Returns false otherwise.
// func Any(funcs []func() bool) bool {
// 	for _, f := range funcs {
// 		if f() {
// 			return true
// 		}
// 	}
// 	return false
// }

// // Initialize takes equal-length arrays of detector and remediator functions. If a given detector returns true, Initialize calls the corresponding remediator.
// func Initialize(detectors []func() bool, remediators []func()) {
// 	for idx, f := range detectors {
// 		if f() {
// 			remediators[idx]()
// 		}
// 	}
// }

// https://github.com/kubernetes-sigs/cluster-api-provider-azure/blob/60b7c6058550ae694935fb03103460a2efa4e332/pkg/cloud/azure/services/virtualmachines/virtualmachines.go#L215
// GenerateRandomBytes generates a random string of n bytes, panicking on failure.
func GenerateRandomBytes(n int) string {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		fmt.Printf("error in generate random: %+#v", err.Error())
	}
	return base64.StdEncoding.EncodeToString(b) //, err
}

// GenerateSafeRandomString generates a string of n bytes from the known set of random safe characters.
func GenerateSafeRandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = safeBytes[mrand.Intn(len(safeBytes))]
	}
	return string(b)
}

// GenerateRandomString generates a string of n characters from the known set of random characters.
func GenerateRandomString(n int) string {
	return GenerateRandomStringFromAlphabet(n, safeBytes)
}

// GenerateRandomString generates a string of n characters from the known set of random characters.
// https://stackoverflow.com/a/31832326 because why not
func GenerateRandomStringFromAlphabet(n int, alphabet string) string {
	src := mrand.NewSource(time.Now().UnixNano())
	sb := strings.Builder{}
	sb.Grow(n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(alphabet) {
			sb.WriteByte(alphabet[idx])
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return sb.String()
}

// GenerateLowerCaseAlphaNumeric generates a random string with lowercase alphanumeric characters.
func GenerateLowerCaseAlphaNumeric(n int) string {
	return GenerateRandomStringFromAlphabet(n, lowerBytes)
}
