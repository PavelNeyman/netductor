class Netductor < Formula
  desc "Netductor control plane CLI (VPN fleet, edge, TUI)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.7.3-dev"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-darwin-arm64"
      sha256 "9701044e14b025ecba96062e108efa9aff5671f5d98b4e72b47c96e0548d01b1"
    end
    # darwin-amd64 asset not published yet — build from source or use arm64 Mac
  end

  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.7.0-dev/netductor-linux-amd64"
      sha256 "cb7ecfa1d792e27de83a93e68fd49d5f779edc85ef5123a9a4b94bd63f46ce68"
    end
  end

  def install
    bin.install Dir["netductor*"].first => "netductor"
  end

  test do
    assert_match "netductor", shell_output("#{bin}/netductor version 2>&1")
  end
end
