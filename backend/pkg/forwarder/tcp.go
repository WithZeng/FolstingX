package forwarder

import (
  "errors"
  "io"
  "net"
  "strconv"
  "sync"
  "sync/atomic"
  "time"
)

type TCPForwarder struct {
  listenAddr string
  targetAddr string
  listener   net.Listener
  closed     atomic.Bool
  upBytes    atomic.Int64
  downBytes  atomic.Int64
  conns      atomic.Int64
  limiter    *TokenBucket
  wg         sync.WaitGroup
}

func NewTCPForwarder(listenHost string, listenPort int, targetHost string, targetPort int, limit int64) *TCPForwarder {
  return &TCPForwarder{
    listenAddr: net.JoinHostPort(listenHost, strconv.Itoa(listenPort)),
    targetAddr: net.JoinHostPort(targetHost, strconv.Itoa(targetPort)),
    limiter:    NewTokenBucket(limit),
  }
}

func (f *TCPForwarder) Start() error {
  ln, err := net.Listen("tcp", f.listenAddr)
  if err != nil {
    return err
  }
  f.listener = ln
  f.closed.Store(false)
  f.wg.Add(1)
  go f.acceptLoop()
  return nil
}

func (f *TCPForwarder) acceptLoop() {
  defer f.wg.Done()
  for !f.closed.Load() {
    conn, err := f.listener.Accept()
    if err != nil {
      if f.closed.Load() {
        return
      }
      time.Sleep(50 * time.Millisecond)
      continue
    }
    f.conns.Add(1)
    f.wg.Add(1)
    go f.handleConn(conn)
  }
}

func (f *TCPForwarder) handleConn(in net.Conn) {
  defer f.wg.Done()
  defer f.conns.Add(-1)
  defer in.Close()

  out, err := net.DialTimeout("tcp", f.targetAddr, 5*time.Second)
  if err != nil {
    return
  }
  defer out.Close()

  done := make(chan struct{}, 2)
  go func() {
    n, _ := io.Copy(out, &rateLimitedReader{r: in, limiter: f.limiter})
    f.upBytes.Add(n)
    done <- struct{}{}
  }()
  go func() {
    n, _ := io.Copy(in, out)
    f.downBytes.Add(n)
    done <- struct{}{}
  }()
  <-done
}

func (f *TCPForwarder) Stop() error {
  if f.closed.Swap(true) {
    return nil
  }
  if f.listener != nil {
    _ = f.listener.Close()
  }
  f.wg.Wait()
  return nil
}

func (f *TCPForwarder) Stats() Stats {
  return Stats{UpBytes: f.upBytes.Load(), DownBytes: f.downBytes.Load(), Connections: f.conns.Load(), LastActivity: time.Now()}
}

type rateLimitedReader struct {
  r       io.Reader
  limiter *TokenBucket
}

func (r *rateLimitedReader) Read(p []byte) (int, error) {
  n, err := r.r.Read(p)
  if n > 0 {
    r.limiter.Wait(n)
  }
  if errors.Is(err, net.ErrClosed) {
    return n, io.EOF
  }
  return n, err
}
