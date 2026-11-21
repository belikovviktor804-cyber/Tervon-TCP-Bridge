package bridge

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync/atomic"
)

// RuntimeACL stores immutable authorization snapshots. A swap changes only
// NEW stream admission. Already authorized TCP streams continue unchanged.
// No new signing, certificates or authentication schemes are introduced.
type RuntimeACL struct {
	current atomic.Pointer[aclSnapshot]
}

type aclSnapshot struct {
	services map[[sha256.Size]byte]map[string]bool
}

func parseACL(authorized map[string][]string, routes map[string][]Backend) (*aclSnapshot, error) {
	if len(authorized) == 0 || len(routes) == 0 {
		return nil, errors.New("empty authorization or routes")
	}
	next := &aclSnapshot{services: make(map[[sha256.Size]byte]map[string]bool, len(authorized))}
	for fingerprint, allowed := range authorized {
		der, err := hex.DecodeString(fingerprint)
		if err != nil || len(der) != sha256.Size || len(allowed) == 0 {
			return nil, errors.New("invalid certificate fingerprint or services")
		}
		var id [sha256.Size]byte
		copy(id[:], der)
		if next.services[id] != nil {
			return nil, errors.New("duplicate certificate fingerprint")
		}
		set := make(map[string]bool, len(allowed))
		for _, service := range allowed {
			if len(routes[service]) == 0 {
				return nil, errors.New("authorized service has no configured backend")
			}
			set[service] = true
		}
		next.services[id] = set
	}
	return next, nil
}

// NewRuntimeACL makes a deep immutable copy of the verified server ACL.
func NewRuntimeACL(authorized map[string][]string, routes map[string][]Backend) (*RuntimeACL, error) {
	next, err := parseACL(authorized, routes)
	if err != nil {
		return nil, err
	}
	a := new(RuntimeACL)
	a.current.Store(next)
	return a, nil
}

// Allowed never mutates the snapshot, so concurrent HTTP/2 stream admission
// is safe even while the registrar refreshes the ACL.
func (a *RuntimeACL) Allowed(fingerprint [sha256.Size]byte, service string) bool {
	if a == nil {
		return false
	}
	snapshot := a.current.Load()
	return snapshot != nil && snapshot.services[fingerprint][service]
}

// Replace validates and fully builds the new ACL before atomic publication.
// On any error, the previous snapshot stays active. Routes remain immutable.
func (a *RuntimeACL) Replace(authorized map[string][]string, routes map[string][]Backend) error {
	if a == nil {
		return errors.New("runtime ACL unavailable")
	}
	next, err := parseACL(authorized, routes)
	if err != nil {
		return err
	}
	a.current.Store(next)
	return nil
}
