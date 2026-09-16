class Netductor < Formula
  desc "Netductor control plane CLI (VPN fleet, edge, TUI)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.7.3-dev"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-darwin-arm64"
      # sha256 updated on release; placeholder forces brew to still try
      sha256 :no_check
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-darwin-amd64"
      sha256 :no_check
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-linux-amd64"
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
