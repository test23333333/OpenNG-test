package tcp

import (
	"net"

	"github.com/haoxingxing/OpenNG/logging"

	"github.com/pires/go-proxyproto"
)

func NewTCPProxyProtocolHandler() ServiceHandler {
	return NewServiceFunction(func(conn *Connection) SerRet {
		sorce := conn.Addr().String()
		sourceip, _, err := net.SplitHostPort(sorce)
		if err != nil || sourceip != "127.0.0.1" {
			logging.Println("sys", "[PROXYPROTOCOL]", "Unallowed Source IP Addr", sourceip)
			return Close
		}
		conn.Upgrade(proxyproto.NewConn(conn.TopConn()), "")
		return Continue
	})
}