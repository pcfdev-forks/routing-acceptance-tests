package tcp_routing_test

import (
	"errors"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/cloudfoundry-incubator/cf-routing-acceptance-tests/helpers/assets"
	"github.com/cloudfoundry-incubator/cf-routing-test-helpers/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Tcp Routing", func() {
	Context("single app port", func() {
		var (
			appName            string
			tcpDropletReceiver = assets.NewAssets().TcpDropletReceiver
			serverId1          string
			externalPort1      uint16
		)

		BeforeEach(func() {
			appName = helpers.GenerateAppName()
			serverId1 = "server1"
			cmd := fmt.Sprintf("tcp-droplet-receiver --serverId=%s", serverId1)
			spaceName := context.RegularUserContext().Space
			externalPort1 = helpers.CreateTcpRouteWithRandomPort(spaceName, domainName, DEFAULT_TIMEOUT)

			// Uses --no-route flag so there is no HTTP route
			helpers.PushAppNoStart(appName, tcpDropletReceiver, routingConfig.GoBuildpackName, domainName, CF_PUSH_TIMEOUT, "-c", cmd, "--no-route")
			helpers.EnableDiego(appName, DEFAULT_TIMEOUT)
			helpers.UpdatePorts(appName, []uint16{3333}, DEFAULT_TIMEOUT)
			helpers.CreateRouteMapping(appName, "", externalPort1, 3333, DEFAULT_TIMEOUT)
			helpers.StartApp(appName, DEFAULT_TIMEOUT)
		})

		AfterEach(func() {
			helpers.AppReport(appName, DEFAULT_TIMEOUT)
			helpers.DeleteApp(appName, DEFAULT_TIMEOUT)
		})

		It("maps a single external port to an application's container port", func() {
			// connect to TCP router/ELB and assert on something
			for _, routerAddr := range routingConfig.Addresses {
				err := checkConnection(serverId1, routerAddr, externalPort1)
				Expect(err).ToNot(HaveOccurred())
			}
		})
	})
})

const (
	DEFAULT_CONNECT_TIMEOUT = 3 * time.Second
	CONN_TYPE               = "tcp"
)

func checkConnection(serverId, addr string, externalPort uint16) error {
	address := fmt.Sprintf("%s:%d", addr, externalPort)

	conn, err := net.DialTimeout(CONN_TYPE, address, DEFAULT_CONNECT_TIMEOUT)
	if err != nil {
		return err
	}

	message := []byte(fmt.Sprintf("Time is %d", time.Now().Nanosecond()))
	_, err = conn.Write(message)
	if err != nil {
		return err
	}

	expectedMessage := []byte(serverId + ":" + string(message))
	buff := make([]byte, len(expectedMessage))
	_, err = conn.Read(buff)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(buff, expectedMessage) {
		return errors.New(fmt.Sprintf("Message mismatch. Actual=[%s], Expected=[%s]",
			string(buff),
			string(expectedMessage)))
	}

	return conn.Close()
}
