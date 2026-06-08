package netcheck

import (
	"net"
	"time"
)

func checkTCP() error {
	conn, err := net.DialTimeout("tcp", "1.1.1.1:443", 3*time.Second)
	if err != nil {
		return err
	}
	return conn.Close()
}
