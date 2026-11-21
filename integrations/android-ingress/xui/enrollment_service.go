package xui

import (
    "context"
    "crypto"
    "errors"
    "time"
)

// SubscriptionProofVerifier is implemented by the EXISTING authenticated
// subscription server. It must verify the current request, not merely look up
// a public email, domain, VLESS UUID or arbitrary client-provided identifier.
// No separate Android credentials or cryptographic system is introduced here.
type SubscriptionProofVerifier interface {
    VerifySubscription(ctx context.Context) (ClientRecord, *UsageSnapshot, error)
}

// IssueForAuthenticatedSubscription reuses the existing client CA and
// SignDeviceCSR, only after the existing subscription authority authenticates
// access and the existing eligibility check passes.
//
// NOT an HTTP endpoint; it does not install/reload the Bridge ACL. The caller
// must establish the existing subscriber's device entitlement and arrange
// authorized ACL activation before returning a usable certificate to Android.
func IssueForAuthenticatedSubscription(
    ctx context.Context,
    verifier SubscriptionProofVerifier,
    csrPEM, clientCAPEM []byte,
    clientCAKey crypto.Signer,
    now time.Time,
) (IssuedCertificate, error) {
    if verifier == nil {
        return IssuedCertificate{}, errors.New("subscription verification unavailable")
    }
    if err := ctx.Err(); err != nil {
        return IssuedCertificate{}, err
    }
    record, usage, err := verifier.VerifySubscription(ctx)
    if err != nil {
        return IssuedCertificate{}, errors.New("subscription authentication failed")
    }
    if err := CheckSubscriptionEligibility(record, usage, now); err != nil {
        return IssuedCertificate{}, errors.New("subscription not eligible for Bridge enrollment")
    }
    return SignDeviceCSR(csrPEM, clientCAPEM, clientCAKey, now, 24*time.Hour)
}
