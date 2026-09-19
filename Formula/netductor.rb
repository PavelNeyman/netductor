class Netductor < Formula
  desc "Netductor control plane CLI (VPN fleet, edge, TUI)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.0"
  license "MIT"
  url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.0/netductor-darwin-arm64"
  sha256 :no_check
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.0/netductor-linux-amd64"
      sha256 :no_check
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match "netductor", shell_output("#{bin}/netductor version 2>&1")
  end
end
