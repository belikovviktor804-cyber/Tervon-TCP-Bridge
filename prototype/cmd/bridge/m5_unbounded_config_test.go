package main

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func TestM5ZeroSessionQuotaAllowedAndNegativeRejected(t *testing.T) {
    cases := []struct{ role string; limit int; valid bool }{
        {"client", 0, true}, {"server", 0, true},
        {"client", 100, true}, {"server", 100, true},
        {"client", -1, false}, {"server", -1, false},
    }
    for _, tc := range cases {
        t.Run(fmt.Sprintf("%s_%d", tc.role, tc.limit), func(t *testing.T) {
            var raw string
            if tc.role == "client" {
                raw = fmt.Sprintf(`{"mode":"client","listen":"127.0.0.1:19001","cert_file":"c","key_file":"k","ca_file":"a","remote_url":"https://example.com/v2/stream","server_name":"example.com","service":"vpn","max_local_connections":%d,"padding_enabled":true,"padding_rate_enabled":true,"jitter_enabled":true}`, tc.limit)
            } else {
                raw = fmt.Sprintf(`{"mode":"server","listen":"127.0.0.1:18443","cert_file":"c","key_file":"k","ca_file":"a","max_streams":%d,"routes":{"vpn":[{"Name":"trusted","Address":"127.0.0.1:19000"}]},"authorized":{"%s":["vpn"]},"padding_enabled":true,"padding_rate_enabled":true,"jitter_enabled":true}`, tc.limit, strings.Repeat("a", 64))
            }
            path := filepath.Join(t.TempDir(), "config.json")
            if err := os.WriteFile(path, []byte(raw), 0600); err != nil { t.Fatal(err) }
            cfg, err := load(path)
            if (err == nil) != tc.valid { t.Fatalf("limit=%d role=%s want valid=%v got %v", tc.limit, tc.role, tc.valid, err) }
            if err == nil && tc.role == "client" && cfg.MaxLocalConnections != tc.limit {
                t.Fatalf("client limit changed to %d", cfg.MaxLocalConnections)
            }
            if err == nil && tc.role == "server" && cfg.MaxStreams != tc.limit {
                t.Fatalf("server limit changed to %d", cfg.MaxStreams)
            }
        })
    }
}
