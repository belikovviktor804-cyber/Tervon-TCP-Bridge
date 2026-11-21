package main

import (
    "os"
    "path/filepath"
    "testing"
)

func TestConfiguredTransportServicesOnlyAdvertisesExistingRoutes(t *testing.T) {
    root := t.TempDir()
    if err := os.MkdirAll(filepath.Join(root, "config"), 0700); err != nil { t.Fatal(err) }
    path := filepath.Join(root, "config", "server.json")
    write := func(value string) {
        t.Helper()
        if err := os.WriteFile(path, []byte(value), 0600); err != nil { t.Fatal(err) }
    }
    write(`{"mode":"server","listen":"0.0.0.0:8443","routes":{"vpn":[{"name":"tcp","address":"host.docker.internal:30000"}]}}`)
    routes, services, err := configuredTransportServices(root)
    if err != nil {t.Fatal(err)}
    if len(routes)!=1 || routes["tcp"]!="vpn" || len(services)!=1 || services[0]!="vpn" {
        t.Fatalf("legacy route incorrectly changed: %v %v", routes, services)
    }
    write(`{"mode":"server","listen":"0.0.0.0:8443","routes":{"vpn":[{"name":"tcp","address":"host.docker.internal:30000"}],"vpn-xhttp":[{"name":"xhttp","address":"host.docker.internal:30443"}]}}`)
    routes, services, err = configuredTransportServices(root)
    if err!=nil {t.Fatal(err)}
    if len(routes)!=2 || routes["xhttp"]!="vpn-xhttp" || len(services)!=2 || services[1]!="vpn-xhttp" {
        t.Fatalf("XHTTP capability missing: %v %v",routes,services)
    }
    write(`{"mode":"server","routes":{"vpn-xhttp":[{"address":"host.docker.internal:30443"}]}}`)
    if _,_,err := configuredTransportServices(root); err==nil {t.Fatal("accepted absent TCP route")}
    write(`{"mode":"server","routes":{"vpn":[{"address":"host.docker.internal:30000"}],"vpn-xhttp":[]}}`)
    routes, services, err = configuredTransportServices(root)
    if err != nil {t.Fatal(err)}
    if len(routes)!=1 || len(services)!=1 {t.Fatal("empty XHTTP route advertised")}
}
