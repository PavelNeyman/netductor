class Netductor < Formula
  desc "Netductor control plane CLI (VPN fleet, edge, TUI)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.7.7-dev"
  license "MIT"

  url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-darwin-arm64"
  sha256 "e6c77f128623bbb787dbac18c63c03189e5bf48191db000dd5751f2c23ff6928"

  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-linux-amd64"
      sha256 "3edad7299a0607d0313f3757ef7878f7dfdc695dabcc3bdd1276c164f6b61125"
    end
  end

  def install
    bin.install Dir["netductor*"].first => "netductor"
  end

  test do
    assert_match "netductor", shell_output("#{bin}/netductor version 2>&1")
  end
end
