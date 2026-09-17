class Netductor < Formula
  desc "Netductor control plane CLI (VPN fleet, edge, TUI)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.7.8-dev"
  license "MIT"
  url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-darwin-arm64"
  sha256 "494bc2e21ef32e95426b16284a86bddccd5c2003ce17a1fc3da26311b5ae7332"
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-linux-amd64"
      sha256 "426069c7d459ff86f0788af4590c677c29a426bc4197f6ee823e884041908740"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match "netductor", shell_output("#{bin}/netductor version 2>&1")
  end
end
