class Netductor < Formula
  desc "Netductor control plane CLI (VPN fleet, edge, TUI)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.7.5-dev"
  license "MIT"

  url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-darwin-arm64"
  sha256 "a5ea9f03e4f04aa6fd217a83fbe2d445834ad3380ae8be9273787a34abaf6c0c"

  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-linux-amd64"
      sha256 "063aee9f361af78c7fe4ddac731acfb5b8e9aa0ad8633bfd415b34298b390057"
    end
  end

  def install
    bin.install Dir["netductor*"].first => "netductor"
  end

  test do
    assert_match "netductor", shell_output("#{bin}/netductor version 2>&1")
  end
end
