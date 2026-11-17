package padding

import (
    "testing"
    "time"
)

// BenchmarkPaddingObserve measures the hot-path cost of the content-blind
// activity detector. It does not benchmark TLS, transport or live throughput.
func BenchmarkPaddingObserve(b *testing.B) {
    state,err:=New(DefaultPolicy())
    if err!=nil{b.Fatal(err)}
    now:=time.Unix(100,0)
    b.ReportAllocs()
    b.ResetTimer()
    for i:=0;i<b.N;i++ {
        now=now.Add(time.Millisecond)
        _=state.Observe(now,16*1024)
    }
}
