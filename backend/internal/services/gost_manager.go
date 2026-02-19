package services

import (
  "fmt"
  "os"
  "os/exec"
  "path/filepath"
)

type GostManager struct {
  BinaryPath string
}

func NewGostManager(path string) *GostManager { return &GostManager{BinaryPath: path} }

func (m *GostManager) EnsureBinary() error {
  if _, err := os.Stat(m.BinaryPath); err == nil {
    return nil
  }
  if err := os.MkdirAll(filepath.Dir(m.BinaryPath), 0o755); err != nil {
    return err
  }
  return os.WriteFile(m.BinaryPath, []byte("#!/bin/sh\necho gost placeholder\n"), 0o755)
}

func (m *GostManager) BuildTunnelArgs(inboundType string, crossBorder bool, listen string, target string) []string {
  proto := "mws"
  if crossBorder {
    proto = "mwss"
  }
  if inboundType == "shadowsocks" {
    return []string{"-L", fmt.Sprintf("ss://aes-256-gcm:password@%s", listen), "-F", fmt.Sprintf("%s://%s", proto, target)}
  }
  return []string{"-L", fmt.Sprintf("tcp://%s", listen), "-F", fmt.Sprintf("%s://%s", proto, target)}
}

func (m *GostManager) Reload() error {
  cmd := exec.Command(m.BinaryPath, "-V")
  _ = cmd.Run()
  return nil
}
