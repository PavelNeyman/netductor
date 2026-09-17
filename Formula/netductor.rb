class Netductor < Formula
  desc "Netductor control plane CLI (VPN fleet, edge, TUI)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.7.10-dev"
  license "MIT"
  url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-darwin-arm64"
  sha256 "4b37d5d2ddc6bf1de566140729e2c060cf4447d74f765c2777c5bf0846425581"
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-linux-amd64"
      sha256 "774b9ceab3eb9e7a7fec084b70666e032a082182e885f9a030e7ddefaf2d3bad"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match "netductor", shell_output("#{bin}/netductor version 2>&1")
  end
end
