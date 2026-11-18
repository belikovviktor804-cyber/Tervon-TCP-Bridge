package framing

import (
    "bytes"
    "context"
    "errors"
    "io"
    "testing"
    "time"

    "tervon-tcp-bridge/jitter"
)

func TestJitterFramingByteExactAndOptIn(t *testing.T) {
    policy := jitter.DefaultPolicy()
    policy.MinDuration = time.Nanosecond
    policy.MinSamples = 2
    policy.MinUsefulBytes = 1024
    var out bytes.Buffer
    e := NewEncoder(&out)
    var requested time.Duration
    if err := e.EnableJitter(context.Background(), policy, func(d time.Duration) {
        requested += d
    }); err != nil { t.Fatal(err) }
    now := time.Unix(100, 0)
    e.now = func() time.Time {
        now = now.Add(time.Millisecond)
        return now
    }
    block := bytes.Repeat([]byte{0, 1, 255, 8}, 1024)
    for i:=0; i<50; i++ {
        n, err := e.Write(block)
        if err != nil || n!=len(block) { t.Fatalf("write %d: %d %v", i,n,err) }
    }
    if err:=e.CloseWrite();err!=nil {t.Fatal(err)}
    got,err:=io.ReadAll(NewDecoder(&out))
    want:=bytes.Repeat(block,50)
    if err!=nil || !bytes.Equal(got,want) {
        t.Fatalf("payload changed: got=%d want=%d err=%v",len(got),len(want),err)
    }
    if requested > 2*policy.MaxDelayPerWindow {
        t.Fatalf("unexpected delay total %v", requested)
    }
    var plain bytes.Buffer
    normal:=NewEncoder(&plain)
    if _,err:=normal.Write(block);err!=nil{t.Fatal(err)}
    if err:=normal.CloseWrite();err!=nil{t.Fatal(err)}
    got,err=io.ReadAll(NewDecoder(&plain))
    if err!=nil || !bytes.Equal(got,block) {t.Fatal("legacy encoder regression")}
}

func TestJitterFramingCancelDoesNotEmitDataOrFIN(t *testing.T) {
    ctx,cancel:=context.WithCancel(context.Background())
    var out bytes.Buffer
    enc:=NewEncoder(&out)
    if err:=enc.EnableJitter(ctx,jitter.DefaultPolicy(),nil);err!=nil{t.Fatal(err)}
    cancel()
    if n,err:=enc.Write([]byte("interactive"));n!=0 || !errors.Is(err,context.Canceled) {
        t.Fatalf("canceled: n=%d err=%v",n,err)
    }
    if out.Len()!=0 {t.Fatal("cancellation emitted data")}
    if err:=enc.CloseWrite();!errors.Is(err,io.ErrClosedPipe) {t.Fatal("FIN after cancel")}
}

func TestJitterFramingPolicyValidation(t *testing.T) {
    p:=jitter.DefaultPolicy()
    p.MaxDelay=100*time.Millisecond
    if err:=NewEncoder(&bytes.Buffer{}).EnableJitter(context.Background(),p,nil);!errors.Is(err,jitter.ErrPolicy) {
        t.Fatalf("invalid jitter enabled: %v",err)
    }
}
