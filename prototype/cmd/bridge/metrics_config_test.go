package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestM0MetricsListenValidation(t *testing.T) {
	cases := []struct {
		address string
		valid   bool
	}{
		{"127.0.0.1:19091", true},
		{"[::1]:19091", true},
		{"", true},
		{"0.0.0.0:19091", false},
		{"[::]:19091", false},
		{"localhost:19091", false},
		{"192.0.2.10:19091", false},
		{"127.0.0.1", false},
		{"127.0.0.1:12345", false}, // must differ from bridge listener
	}
	for _, tc := range cases {
		t.Run(tc.address, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "config.json")
			raw := fmt.Sprintf(`{"mode":"client","listen":"127.0.0.1:12345","cert_file":"c","key_file":"k","ca_file":"a","max_local_connections":2,"remote_url":"https://example.com/v1/stream","service":"vpn","server_name":"example.com","metrics_listen":%q}`, tc.address)
			if err := os.WriteFile(p, []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := load(p)
			if (err == nil) != tc.valid {
				t.Fatalf("address %q: valid=%v, error=%v", tc.address, tc.valid, err)
			}
		})
	}
}
