class Netductor < Formula
  desc "Netductor control plane CLI (VPN fleet, edge, TUI)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.7.11-dev"
  license "MIT"
  url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-darwin-arm64"
  sha256 "c158e08b7cace68d1252686be69eaabc507127625ef78db6a8a5263cf4652b65"
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-linux-amd64"
      sha256 "381d777cf42ae9e2f2f0fbdf4c94c6c53e64eafb321d974e92305bbe4093c27f"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match "netductor", shell_output("#{bin}/netductor version 2>&1")
  end
end
