package forwarder

import (
  "net"
  "strconv"
  "sync"
  "sync/atomic"
  "time"
)

type UDPForwarder struct {
  listenAddr string
  targetAddr *net.UDPAddr
  conn       *net.UDPConn
  closed     atomic.Bool
  upBytes    atomic.Int64
  downBytes  atomic.Int64
  conns      atomic.Int64
  limiter    *TokenBucket
  wg         sync.WaitGroup
}

func NewUDPForwarder(listenHost string, listenPort int, targetHost string, targetPort int, limit int64) (*UDPForwarder, error) {
  ta, err := net.ResolveUDPAddr("udp", net.JoinHostPort(targetHost, strconv.Itoa(targetPort)))
  if err != nil {
    return nil, err
  }
  return &UDPForwarder{
    listenAddr: net.JoinHostPort(listenHost, strconv.Itoa(listenPort)),
    targetAddr: ta,
    limiter:    NewTokenBucket(limit),
  }, nil
}

func (f *UDPForwarder) Start() error {
  la, err := net.ResolveUDPAddr("udp", f.listenAddr)
  if err != nil {
    return err
  }
  conn, err := net.ListenUDP("udp", la)
  if err != nil {
    return err
  }
  f.conn = conn
  f.closed.Store(false)
  f.wg.Add(1)
  go f.loop()
  return nil
}

func (f *UDPForwarder) loop() {
  defer f.wg.Done()
  buf := make([]byte, 65535)
  for !f.closed.Load() {
    _ = f.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
    n, clientAddr, err := f.conn.ReadFromUDP(buf)
    if err != nil {
      if ne, ok := err.(net.Error); ok && ne.Timeout() {
        continue
      }
      if f.closed.Load() {
        return
      }
      continue
    }
    f.conns.Store(1)
    f.limiter.Wait(n)
    f.upBytes.Add(int64(n))

    upstream, err := net.DialUDP("udp", nil, f.targetAddr)
    if err != nil {
      continue
    }
    _, _ = upstream.Write(buf[:n])
    _ = upstream.SetReadDeadline(time.Now().Add(3 * time.Second))
    rn, _, err := upstream.ReadFromUDP(buf)
    if err == nil && rn > 0 {
      _, _ = f.conn.WriteToUDP(buf[:rn], clientAddr)
      f.downBytes.Add(int64(rn))
    }
    _ = upstream.Close()
  }
}

func (f *UDPForwarder) Stop() error {
  if f.closed.Swap(true) {
    return nil
  }
  if f.conn != nil {
    _ = f.conn.Close()
  }
  f.wg.Wait()
  return nil
}

func (f *UDPForwarder) Stats() Stats {
  return Stats{UpBytes: f.upBytes.Load(), DownBytes: f.downBytes.Load(), Connections: f.conns.Load(), LastActivity: time.Now()}
}
