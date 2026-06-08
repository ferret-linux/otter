package netcheck

import (
	"net"
	"time"
)

var tcpTargets = []string{
	"1.1.1.1:443",
	"8.8.8.8:443",
	"140.82.112.4:443",
	"185.199.108.133:443",
}

func checkTCP() error {
	for _, addr := range tcpTargets {
		conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
		if err == nil {
			conn.Close()
			return nil
		}
	}
	return errAllFailed
}
