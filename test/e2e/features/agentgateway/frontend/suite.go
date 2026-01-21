//go:build e2e

package frontend

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	"github.com/onsi/gomega"
	"github.com/stretchr/testify/suite"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kgateway-dev/kgateway/v2/pkg/utils/fsutils"
	"github.com/kgateway-dev/kgateway/v2/pkg/utils/kubeutils"
	"github.com/kgateway-dev/kgateway/v2/pkg/utils/requestutils/curl"
	"github.com/kgateway-dev/kgateway/v2/test/e2e"
	"github.com/kgateway-dev/kgateway/v2/test/e2e/defaults"
	"github.com/kgateway-dev/kgateway/v2/test/e2e/tests/base"
	"github.com/kgateway-dev/kgateway/v2/test/gomega/matchers"
)

var _ e2e.NewSuiteFunc = NewTestingSuite

var (
	setupManifest        = filepath.Join(fsutils.MustGetThisDir(), "testdata", "setup.yaml")
	accessLogManifest    = filepath.Join(fsutils.MustGetThisDir(), "testdata", "accesslog.yaml")

	proxyServiceObjectMeta = metav1.ObjectMeta{
		Name:      "gw",
		Namespace: "default",
	}

	// setup manifests applied before the test
	setup = base.TestCase{
		Manifests: []string{
			setupManifest,
			defaults.CurlPodManifest,
			defaults.HttpbinManifest,
		},
	}

	testCases = map[string]*base.TestCase{
		"TestAccessLogAttributes": {
			Manifests: []string{
				accessLogManifest,
			},
		},
	}
)

// testingSuite is a suite of agentgateway frontend tests
type testingSuite struct {
	*base.BaseTestingSuite
}

func NewTestingSuite(ctx context.Context, testInst *e2e.TestInstallation) suite.TestingSuite {
	return &testingSuite{
		base.NewBaseTestingSuite(ctx, testInst, setup, testCases),
	}
}

func (s *testingSuite) TestAccessLogAttributes() {
	s.testAccessLogAttributes()
}

// testAccessLogAttributes makes a request to the httpbin service with a custom header
// and checks if the agentgateway proxy logs contain the expected attributed.
func (s *testingSuite) testAccessLogAttributes() {
	s.TestInstallation.Assertions.EventuallyAgwPolicyCondition(s.Ctx, "accesslog-header", "default", "Accepted", metav1.ConditionTrue)

	userId := fmt.Sprintf("user-%v", rand.Intn(10000))
	s.TestInstallation.Assertions.Gomega.Eventually(func(g gomega.Gomega) {
		// make curl request to httpbin service with the custom header
		s.TestInstallation.Assertions.AssertEventualCurlResponse(
			s.Ctx,
			defaults.CurlPodExecOpt,
			[]curl.Option{
				curl.WithHostHeader("www.example.com"),
				curl.WithHeader("x-user-id", userId),
				curl.WithPath("/get"),
				curl.WithHost(kubeutils.ServiceFQDN(proxyServiceObjectMeta)),
				curl.WithPort(8080),
			},
			&matchers.HttpResponse{
				StatusCode: 200,
			},
			20*time.Second,
			2*time.Second,
		)

		// fetch the agentgateway proxy logs
		// In e2e tests, the proxy is usually a deployment named 'agentgateway'
		pods, err := s.TestInstallation.Actions.Kubectl().GetPodsInNsWithLabel(
			s.Ctx,
			"kgateway-system",
			"app.kubernetes.io/name=agentgateway",
		)
		g.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to get agentgateway pods")
		g.Expect(pods).NotTo(gomega.BeEmpty(), "No agentgateway pods found")

		logs, err := s.TestInstallation.Actions.Kubectl().GetContainerLogs(s.Ctx, "kgateway-system", pods[0])
		g.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to get pod logs")

		// Check if the logs contain the expected x-user-id attribute
		g.Expect(logs).To(gomega.ContainSubstring(fmt.Sprintf("x-user-id: %s", userId)), "missing expected x-user-id in gateway logs")
	}, time.Second*60, time.Second*5, "should find x-user-id in agentgateway logs").Should(gomega.Succeed())
}
