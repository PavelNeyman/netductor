class Netductor < Formula
  desc "Netductor control plane CLI (VPN fleet, edge, TUI)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.7.9-dev"
  license "MIT"
  url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-darwin-arm64"
  sha256 "a4c590c4663d4d4bff6dee766fd09453d12254c034ef56fbf4b1e77e3579f970"
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-linux-amd64"
      sha256 "2504ba348a2656e26b097f449398e6af3b3fe099507600b5336761821d31678d"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match "netductor", shell_output("#{bin}/netductor version 2>&1")
  end
end
