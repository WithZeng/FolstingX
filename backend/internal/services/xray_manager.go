package services

import (
  "fmt"
  "os"
  "os/exec"
  "path/filepath"
)

type XrayManager struct {
  BinaryPath string
}

func NewXrayManager(path string) *XrayManager { return &XrayManager{BinaryPath: path} }

func (m *XrayManager) EnsureBinary() error {
  if _, err := os.Stat(m.BinaryPath); err == nil {
    return nil
  }
  if err := os.MkdirAll(filepath.Dir(m.BinaryPath), 0o755); err != nil {
    return err
  }
  return os.WriteFile(m.BinaryPath, []byte("#!/bin/sh\necho xray placeholder\n"), 0o755)
}

func (m *XrayManager) BuildVLESSRealityConfig(listenPort int, targetAddr string, targetPort int, uuid string, publicKey string, shortID string, serverName string) string {
  return fmt.Sprintf(`{"inbounds":[{"port":%d,"protocol":"vless","settings":{"clients":[{"id":"%s"}]},"streamSettings":{"security":"reality","realitySettings":{"serverNames":["%s"],"publicKey":"%s","shortIds":["%s"]}}}],"outbounds":[{"protocol":"freedom","settings":{"domainStrategy":"UseIPv4"}}],"routing":{"rules":[{"type":"field","outboundTag":"direct","network":"tcp,udp"}]},"target":"%s:%d"}`,
    listenPort, uuid, serverName, publicKey, shortID, targetAddr, targetPort)
}

func (m *XrayManager) Reload() error {
  cmd := exec.Command(m.BinaryPath, "api", "restart")
  _ = cmd.Run()
  return nil
}
