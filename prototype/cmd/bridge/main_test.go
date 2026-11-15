package main

import (
 "os"
 "path/filepath"
 "strings"
 "testing"
)

func TestLoadRejectsUnsafeClientConfigurations(t *testing.T) {
 cases := []string{
  `{"mode":"client","listen":"127.0.0.1:0","cert_file":"c","key_file":"k","ca_file":"a","max_local_connections":1,"remote_url":"http://example.com/v1/stream","service":"vpn","server_name":"example.com"}`,
  `{"mode":"client","listen":"127.0.0.1:0","cert_file":"c","key_file":"k","ca_file":"a","max_local_connections":0,"remote_url":"https://example.com/v1/stream","service":"vpn","server_name":"example.com"}`,
  `{"mode":"server","listen":":443","cert_file":"c","key_file":"k","ca_file":"a","max_streams":0,"routes":{"vpn":[{"Name":"x","Address":"127.0.0.1:1"}]},"authorized":{"aa":["vpn"]}}`,
  `{"mode":"client","listen":"127.0.0.1:0","cert_file":"c","key_file":"k","ca_file":"a","max_local_connections":1,"remote_url":"https://example.com/v1/stream","service":"vpn","server_name":"example.com","unexpected":true}`,
 }
 for i, input := range cases {
  p := filepath.Join(t.TempDir(), "config.json")
  if err := os.WriteFile(p, []byte(input), 0600); err != nil { t.Fatal(err) }
  if _, err := load(p); err == nil { t.Errorf("case %d accepted unsafe configuration", i) }
 }
}

func TestLoadValidClient(t *testing.T) {
 p := filepath.Join(t.TempDir(), "config.json")
 input := `{"mode":"client","listen":"127.0.0.1:12345","cert_file":"c","key_file":"k","ca_file":"a","max_local_connections":2,"remote_url":"https://example.com/v1/stream","service":"vpn","server_name":"example.com"}`
 if err := os.WriteFile(p, []byte(input), 0600); err != nil { t.Fatal(err) }
 c, err := load(p)
 if err != nil { t.Fatal(err) }
 if c.MaxLocalConnections != 2 || !strings.HasPrefix(c.RemoteURL, "https://") { t.Fatalf("unexpected config: %+v", c) }
}
