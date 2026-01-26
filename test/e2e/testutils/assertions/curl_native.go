//go:build e2e

package assertions

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	. "github.com/onsi/gomega"

	"github.com/kgateway-dev/kgateway/v2/pkg/utils/requestutils/curl"
	"github.com/kgateway-dev/kgateway/v2/test/gomega/matchers"
	"github.com/kgateway-dev/kgateway/v2/test/helpers"
)

// Native HTTP assertion methods that use Go's net/http directly instead of executing
// curl commands inside a pod. These methods provide significant performance improvements
// by eliminating pod scheduling, kubectl exec, and curl output parsing overhead.
//
// Gateway Accessibility Note:
// These methods execute HTTP requests from the test runner machine, not from within the cluster.
// The gateway must be accessible from outside the cluster via one of:
// - LoadBalancer service with external IP
// - NodePort service
// - Port-forwarding (use StartPortForward helper)
//
// Example usage:
//
//	gatewayAddr := installation.Assertions.EventuallyGatewayAddress(ctx, "gw", "default")
//	installation.Assertions.AssertEventualCurlResponseNative(ctx, []curl.Option{
//	    curl.WithHost(gatewayAddr),
//	    curl.WithPort(80),
//	}, &matchers.HttpResponse{StatusCode: 200})

// AssertEventualCurlResponseNative asserts that a native Go HTTP request eventually returns
// the expected response. Uses curl.ExecuteRequest() instead of kubectl exec into a curl pod.
func (p *Provider) AssertEventualCurlResponseNative(
	ctx context.Context,
	curlOptions []curl.Option,
	expectedResponse *matchers.HttpResponse,
	timeout ...time.Duration,
) {
	resp := p.AssertEventualCurlReturnResponseNative(ctx, curlOptions, expectedResponse, timeout...)
	resp.Body.Close()
}

// AssertEventualCurlReturnResponseNative is like AssertEventualCurlResponseNative but returns
// the response for further inspection. Caller is responsible for closing the response body.
func (p *Provider) AssertEventualCurlReturnResponseNative(
	ctx context.Context,
	curlOptions []curl.Option,
	expectedResponse *matchers.HttpResponse,
	timeout ...time.Duration,
) *http.Response {
	currentTimeout, pollingInterval := helpers.GetTimeouts(timeout...)

	var httpResponse *http.Response
	var cachedBodyBytes []byte
	p.Gomega.Eventually(func(g Gomega) {
		resp, err := curl.ExecuteRequest(curlOptions...)
		if err != nil {
			fmt.Printf("Native HTTP request failed: %v\n", err)
			g.Expect(err).NotTo(HaveOccurred())
			return
		}

		// Read the body, cache it, and restore for matcher
		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		g.Expect(err).NotTo(HaveOccurred())
		cachedBodyBytes = make([]byte, len(bodyBytes))
		copy(cachedBodyBytes, bodyBytes)
		resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		httpResponse = resp
		fmt.Printf("Native HTTP response: status=%d, body=%s\n", resp.StatusCode, string(bodyBytes))

		g.Expect(resp).To(matchers.HaveHttpResponse(expectedResponse))
	}).
		WithTimeout(currentTimeout).
		WithPolling(pollingInterval).
		WithContext(ctx).
		Should(Succeed(), "failed to get expected response via native HTTP")

	if len(cachedBodyBytes) > 0 && httpResponse != nil {
		httpResponse.Body.Close()
		httpResponse.Body = io.NopCloser(bytes.NewBuffer(cachedBodyBytes))
	}
	return httpResponse
}

// AssertEventuallyConsistentCurlResponseNative asserts that a native Go HTTP request
// eventually and then consistently matches the expected response.
func (p *Provider) AssertEventuallyConsistentCurlResponseNative(
	ctx context.Context,
	curlOptions []curl.Option,
	expectedResponse *matchers.HttpResponse,
	timeout ...time.Duration,
) {
	// First, wait for eventual success
	p.AssertEventualCurlResponseNative(ctx, curlOptions, expectedResponse, timeout...)

	// Then, verify consistent success
	pollTimeout := 3 * time.Second
	pollInterval := 1 * time.Second
	if len(timeout) > 0 {
		pollTimeout, pollInterval = helpers.GetTimeouts(timeout...)
	}

	p.Gomega.Consistently(func(g Gomega) {
		resp, err := curl.ExecuteRequest(curlOptions...)
		if err != nil {
			g.Expect(err).NotTo(HaveOccurred())
			return
		}
		defer resp.Body.Close()
		g.Expect(resp).To(matchers.HaveHttpResponse(expectedResponse))
	}).
		WithTimeout(pollTimeout).
		WithPolling(pollInterval).
		WithContext(ctx).
		Should(Succeed())
}

// AssertEventualCurlErrorNative asserts that a native Go HTTP request eventually fails
// with an error (e.g., connection refused). This is useful for testing that a service
// is not reachable.
func (p *Provider) AssertEventualCurlErrorNative(
	ctx context.Context,
	curlOptions []curl.Option,
	timeout ...time.Duration,
) {
	currentTimeout, pollingInterval := helpers.GetTimeouts(timeout...)

	p.Gomega.Eventually(func(g Gomega) {
		resp, err := curl.ExecuteRequest(curlOptions...)
		if resp != nil {
			resp.Body.Close()
			fmt.Printf("Expected error but got response: status=%d\n", resp.StatusCode)
		}
		g.Expect(err).To(HaveOccurred(), "expected HTTP request to fail")
	}).
		WithTimeout(currentTimeout).
		WithPolling(pollingInterval).
		WithContext(ctx).
		Should(Succeed(), "expected HTTP request to fail")
}
