class Netductor < Formula
  desc "Netductor control plane CLI (VPN fleet, edge, TUI)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.7.13-dev"
  license "MIT"
  url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-darwin-arm64"
  sha256 "402711b7a35c51aba356a7a3d5654ebea374427f06e557d8fffa189eb037e621"
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-linux-amd64"
      sha256 "121b86079e5c7db97e626be5ee31e936a22b790561488adb19ebfb342550fce8"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match "netductor", shell_output("#{bin}/netductor version 2>&1")
  end
end
