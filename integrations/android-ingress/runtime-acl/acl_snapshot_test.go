package bridge

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
	"testing"
)

func aclFixture() (map[string][]string, map[string][]Backend, [sha256.Size]byte) {
	id := sha256.Sum256([]byte("m6 test device"))
	routes := map[string][]Backend{"vpn": {{Name: "existing", Address: "127.0.0.1:30000"}}}
	return map[string][]string{hex.EncodeToString(id[:]): {"vpn"}}, routes, id
}

func TestM6RuntimeACLAtomicReplaceAndReject(t *testing.T) {
	initial, routes, id := aclFixture()
	acl, err := NewRuntimeACL(initial, routes)
	if err != nil { t.Fatal(err) }
	if !acl.Allowed(id,"vpn") || acl.Allowed(id,"not-configured") {
		t.Fatal("initial ACL failed")
	}
	// A caller may mutate the input after creating a snapshot. It MUST
	// not change the authorization that concurrent streams observe.
	delete(initial, hex.EncodeToString(id[:]))
	if !acl.Allowed(id,"vpn") { t.Fatal("snapshot aliased caller data") }
	other := sha256.Sum256([]byte("new device"))
	renew := map[string][]string{hex.EncodeToString(other[:]): {"vpn"}}
	if err := acl.Replace(renew,routes); err != nil { t.Fatal(err) }
	if acl.Allowed(id,"vpn") || !acl.Allowed(other,"vpn") {
		t.Fatal("updated ACL not active")
	}
	if err := acl.Replace(map[string][]string{"garbage": {"vpn"}},routes); err == nil {
		t.Fatal("malformed ACL accepted")
	}
	if !acl.Allowed(other,"vpn") {
		t.Fatal("invalid ACL destroyed previous snapshot")
	}
	if err := acl.Replace(map[string][]string{hex.EncodeToString(id[:]): {"unknown"}},routes); err == nil {
		t.Fatal("unknown backend service accepted")
	}
}

func TestM6RuntimeACLConcurrentReadAndRefresh(t *testing.T) {
	initial,routes,id := aclFixture()
	acl,err := NewRuntimeACL(initial,routes)
	if err!=nil { t.Fatal(err) }
	other := sha256.Sum256([]byte("second device"))
	expanded := map[string][]string{
		hex.EncodeToString(id[:]): {"vpn"},
		hex.EncodeToString(other[:]): {"vpn"},
	}
	var wg sync.WaitGroup
	for reader := 0; reader < 20; reader++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i:=0;i<3000;i++ {
				if !acl.Allowed(id,"vpn"){t.Error("existing device rejected during refresh");return}
				_ = acl.Allowed(other,"vpn")
			}
		}()
	}
	wg.Add(1)
	go func(){
		defer wg.Done()
		for i:=0;i<200;i++ {
			if err:=acl.Replace(expanded,routes);err!=nil {t.Error(err);return}
			if err:=acl.Replace(initial,routes);err!=nil {t.Error(err);return}
		}
	}()
	wg.Wait()
}
