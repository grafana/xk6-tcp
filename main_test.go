package tcp

import (
	"context"
	"net"
	"os"
	"strconv"
	"testing"

	"github.com/grafana/xk6-tcp/internal/echo"
)

func TestMain(m *testing.M) {
	server := echo.Setup()
	stopBannerServer := startBannerServer()
	code := m.Run()

	stopBannerServer()
	server.Stop()
	os.Exit(code)
}

func startBannerServer() func() {
	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "localhost:0")
	if err != nil {
		panic(err)
	}

	addr := listener.Addr()

	tcpAddr, ok := addr.(*net.TCPAddr)
	if !ok {
		_ = listener.Close()

		panic("banner listener is not TCP")
	}

	if err := os.Setenv("TCP_BANNER_PORT", strconv.Itoa(tcpAddr.Port)); err != nil {
		_ = listener.Close()

		panic(err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			go func() {
				defer func() { _ = conn.Close() }()

				_, _ = conn.Write([]byte("banner"))
			}()
		}
	}()

	return func() {
		_ = listener.Close()
	}
}
