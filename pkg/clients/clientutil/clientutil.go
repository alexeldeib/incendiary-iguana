/*
Copyright 2019 Alexander Eldeib.
*/

package clientutil

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	mrand "math/rand"
	"net/http"
	"net/http/httputil"

	"github.com/Azure/go-autorest/autorest"
)

const safeBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// TODO(ace): probably move this to a more descriptive package
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
	return base64.StdEncoding.EncodeToString(b) //, err
}

func GenerateSafeRandomString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = safeBytes[mrand.Intn(len(safeBytes))]
	}
	return string(b)
}

func LogRequest() autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			r, err := p.Prepare(r)
			if err != nil {
				fmt.Println(err)
			}
			dump, _ := httputil.DumpRequestOut(r, true)
			fmt.Println(string(dump))
			return r, err
		})
	}
}

func LogResponse() autorest.RespondDecorator {
	return func(p autorest.Responder) autorest.Responder {
		return autorest.ResponderFunc(func(r *http.Response) error {
			err := p.Respond(r)
			if err != nil {
				fmt.Println(err)
			}
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Println(string(dump))
			return err
		})
	}
}
