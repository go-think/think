package helper

import (
	"net"
	"os"
	"strings"
)

// ParseAddr resolves the host and port for the application HTTP server.
// Defaults to 127.0.0.1:8080 matching standard local development behavior (e.g. Laravel serve).
func ParseAddr(addrs ...string) string {
	addr := "127.0.0.1"
	port := "8080"

	// Environment variable fallbacks matching SERVER_HOST, HOST, SERVER_PORT, PORT
	if h := os.Getenv("SERVER_HOST"); h != "" {
		addr = h
	} else if h := os.Getenv("HOST"); h != "" {
		addr = h
	} else if a := os.Getenv("THINKGO_ADDR"); a != "" {
		addr = a
	}

	if p := os.Getenv("SERVER_PORT"); p != "" {
		port = p
	} else if p := os.Getenv("PORT"); p != "" {
		port = p
	} else if p := os.Getenv("THINKGO_PORT"); p != "" {
		port = p
	}

	if len(addrs) > 0 && addrs[0] != "" {
		raw := addrs[0]
		if strings.Contains(raw, ":") {
			host, p, err := net.SplitHostPort(raw)
			if err == nil {
				if host != "" {
					addr = host
				}
				if p != "" {
					port = p
				}
			} else {
				parts := strings.Split(raw, ":")
				if parts[0] != "" {
					addr = parts[0]
				}
				if len(parts) > 1 && parts[1] != "" {
					port = parts[1]
				}
			}
		} else {
			addr = raw
		}
	}

	return net.JoinHostPort(addr, port)
}
