package tcp_routing_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	"testing"

	"github.com/cloudfoundry-incubator/cf-routing-acceptance-tests/helpers"
	routing_helpers "github.com/cloudfoundry-incubator/cf-routing-test-helpers/helpers"
	"github.com/cloudfoundry-incubator/cf-test-helpers/cf"
	"github.com/cloudfoundry-incubator/cf-test-helpers/generator"
	cf_helpers "github.com/cloudfoundry-incubator/cf-test-helpers/helpers"
	routing_api "github.com/cloudfoundry-incubator/routing-api"
	"github.com/cloudfoundry-incubator/routing-api/models"
	uaaclient "github.com/cloudfoundry-incubator/uaa-go-client"
	uaaconfig "github.com/cloudfoundry-incubator/uaa-go-client/config"
)

func TestTcpRouting(t *testing.T) {
	RegisterFailHandler(Fail)

	routingConfig = helpers.LoadConfig()
	if routingConfig.DefaultTimeout > 0 {
		DEFAULT_TIMEOUT = routingConfig.DefaultTimeout * time.Second
	}

	if routingConfig.CfPushTimeout > 0 {
		CF_PUSH_TIMEOUT = routingConfig.CfPushTimeout * time.Second
	}

	componentName := "TCP Routing"

	rs := []Reporter{}

	context = cf_helpers.NewContext(routingConfig.Config)
	environment = cf_helpers.NewEnvironment(context)

	if routingConfig.ArtifactsDirectory != "" {
		cf_helpers.EnableCFTrace(routingConfig.Config, componentName)
		rs = append(rs, cf_helpers.NewJUnitReporter(routingConfig.Config, componentName))
	}

	RunSpecsWithDefaultAndCustomReporters(t, componentName, rs)
}

const preallocatedExternalPorts = 100

var (
	DEFAULT_TIMEOUT = 2 * time.Minute
	CF_PUSH_TIMEOUT = 2 * time.Minute

	sampleReceiverPath string
	externalIP         string
	domainName         string

	routingConfig    helpers.RoutingConfig
	routingApiClient routing_api.Client
	context          cf_helpers.SuiteContext
	environment      *cf_helpers.Environment
	logger           lager.Logger

	externalPort  uint32
	bucketSize    int
	containerPort uint32
)

func validateTcpRouteMapping(tcpRouteMapping models.TcpRouteMapping) bool {
	if tcpRouteMapping.TcpRoute.RouterGroupGuid == "" {
		return false
	}

	if tcpRouteMapping.TcpRoute.ExternalPort <= 0 {
		return false
	}

	if tcpRouteMapping.HostIP == "" {
		return false
	}

	if tcpRouteMapping.HostPort <= 0 {
		return false
	}

	return true
}

var _ = SynchronizedBeforeSuite(func() []byte {
	routingApiClient = routing_api.NewClient(routingConfig.RoutingApiUrl)

	logger = lagertest.NewTestLogger("test")

	uaaClient := newUaaClient(routingConfig, logger)
	token, err := uaaClient.FetchToken(true)
	Expect(err).ToNot(HaveOccurred())

	routingApiClient.SetToken(token.AccessToken)
	routerGroupGuid := routing_helpers.GetRouterGroupGuid(routingApiClient)
	domainName = fmt.Sprintf("%s.%s", generator.PrefixedRandomName("TCP-DOMAIN-"), routingConfig.SystemDomain)
	cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
		routing_helpers.CreateSharedDomain(domainName, routerGroupGuid, DEFAULT_TIMEOUT)
	})

	return []byte{}
}, func(payload []byte) {
	environment.Setup()
})

var _ = SynchronizedAfterSuite(func() {
}, func() {
	cf.AsUser(context.AdminUserContext(), context.ShortTimeout(), func() {
		routing_helpers.DeleteSharedDomain(domainName, DEFAULT_TIMEOUT)
	})
	environment.Teardown()
	CleanupBuildArtifacts()
})

func newUaaClient(routerApiConfig helpers.RoutingConfig, logger lager.Logger) uaaclient.Client {

	tokenURL := fmt.Sprintf("%s:%d", routerApiConfig.OAuth.TokenEndpoint, routerApiConfig.OAuth.Port)

	cfg := &uaaconfig.Config{
		UaaEndpoint:           tokenURL,
		SkipVerification:      routerApiConfig.OAuth.SkipOAuthTLSVerification,
		ClientName:            routerApiConfig.OAuth.ClientName,
		ClientSecret:          routerApiConfig.OAuth.ClientSecret,
		MaxNumberOfRetries:    3,
		RetryInterval:         500 * time.Millisecond,
		ExpirationBufferInSec: 30,
	}

	uaaClient, err := uaaclient.NewClient(logger, cfg, clock.NewClock())
	Expect(err).ToNot(HaveOccurred())

	_, err = uaaClient.FetchToken(true)
	Expect(err).ToNot(HaveOccurred())

	return uaaClient
}
