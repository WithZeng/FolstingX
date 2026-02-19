package forwarder

import (
  "sync"
  "time"
)

// TokenBucket 是线程安全的简化令牌桶限速器。
type TokenBucket struct {
  mu       sync.Mutex
  rate     int64
  burst    int64
  tokens   int64
  lastFill time.Time
}

func NewTokenBucket(bytesPerSec int64) *TokenBucket {
  if bytesPerSec <= 0 {
    return &TokenBucket{}
  }
  return &TokenBucket{
    rate:     bytesPerSec,
    burst:    bytesPerSec,
    tokens:   bytesPerSec,
    lastFill: time.Now(),
  }
}

func (t *TokenBucket) Wait(n int) {
  if t == nil || t.rate <= 0 || n <= 0 {
    return
  }
  need := int64(n)
  for {
    t.mu.Lock()
    now := time.Now()
    elapsed := now.Sub(t.lastFill).Seconds()
    if elapsed > 0 {
      t.tokens += int64(float64(t.rate) * elapsed)
      if t.tokens > t.burst {
        t.tokens = t.burst
      }
      t.lastFill = now
    }
    if t.tokens >= need {
      t.tokens -= need
      t.mu.Unlock()
      return
    }
    missing := need - t.tokens
    wait := time.Duration(float64(missing)/float64(t.rate)*float64(time.Second))
    t.mu.Unlock()
    if wait < time.Millisecond {
      wait = time.Millisecond
    }
    time.Sleep(wait)
  }
}
