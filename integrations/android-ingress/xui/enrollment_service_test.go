package xui

import (
    "context"
    "crypto/x509"
    "encoding/pem"
    "errors"
    "testing"
    "time"
)

type subscriptionProofFixture struct {
    calls *int
    record ClientRecord
    usage *UsageSnapshot
    err error
}

func (v subscriptionProofFixture) VerifySubscription(context.Context) (ClientRecord, *UsageSnapshot, error) {
    (*v.calls)++
    return v.record, v.usage, v.err
}

func TestIssuerRefusesUnauthorizedAndIneligible(t *testing.T) {
    now := time.Unix(1_700_000_000, 0)
    eligible := ClientRecord{ID:1, Email:"test", Enable:true, ExpiryTime:0, TotalGB:0}
    cases := []struct {
        name string
        verifier SubscriptionProofVerifier
    }{
        {"missing auth verifier", nil},
        {"denied", subscriptionProofFixture{calls:new(int), err:errors.New("denied")}},
        {"disabled", subscriptionProofFixture{calls:new(int), record:ClientRecord{ID:1, Email:"test"}}},
        {"expired", subscriptionProofFixture{calls:new(int), record:ClientRecord{ID:1, Email:"test", Enable:true, ExpiryTime:100}}},
        {"quota unknown", subscriptionProofFixture{calls:new(int), record:ClientRecord{ID:1, Email:"test", Enable:true, TotalGB:500}}},
        {"authorized but invalid CSR/CA", subscriptionProofFixture{calls:new(int), record:eligible}},
    }
    for _, tc := range cases {
        t.Run(tc.name,func(t *testing.T) {
            issued,err := IssueForAuthenticatedSubscription(context.Background(),tc.verifier,nil,nil,nil,now)
            if err == nil || len(issued.PEM) != 0 || issued.Fingerprint != "" {
                t.Fatal("certificate issued without verified subscription and valid CSR/CA")
            }
        })
    }
}

func TestIssuerUsesExistingCSRSignerAfterSubscriptionVerification(t *testing.T) {
    now := time.Now().UTC().Truncate(time.Second)
    csr,caPEM,caKey := testCertificateInputs(t,now)
    calls := 0
    eligible := ClientRecord{ID:10, Email:"active", Enable:true, TotalGB:0}
    issued,err := IssueForAuthenticatedSubscription(context.Background(),
        subscriptionProofFixture{calls:&calls,record:eligible},csr,caPEM,caKey,now)
    if err != nil { t.Fatal(err) }
    if calls != 1 { t.Fatal("subscription checked incorrect number of times") }
    block,rest := pem.Decode(issued.PEM)
    if block == nil || block.Type != "CERTIFICATE" || len(rest) != 0 {t.Fatal("invalid leaf PEM")}
    leaf,err := x509.ParseCertificate(block.Bytes)
    if err != nil { t.Fatal(err) }
    if !leaf.NotAfter.After(now) || leaf.NotAfter.After(now.Add(24*time.Hour)) {
        t.Fatal("wrong leaf validity")
    }
    if len(leaf.ExtKeyUsage)!=1 || leaf.ExtKeyUsage[0]!=x509.ExtKeyUsageClientAuth {
        t.Fatal("leaf not limited to TLS client authentication")
    }
    if issued.Fingerprint=="" {t.Fatal("missing ACL fingerprint")}
}
