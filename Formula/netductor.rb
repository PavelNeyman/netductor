class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.92"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.92/netductor-darwin-arm64"
      sha256 "52326088dc0460dd51b23d101c2c60dfe2c0f64dfc8f9fd409d414f534e9e899"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.92/netductor-darwin-amd64"
      sha256 "e6a4422b2f87a27d6efc64e4aa1e4f431ca648df73529613283713459ee7cd14"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.92/netductor-linux-amd64"
      sha256 "ba7a40cdee0d37d1d70b37a781f6f99f0febce17276fe386add23d625293dfb1"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.92/netductor-linux-arm64"
      sha256 "9b48176a90cc9cbf4cc88e84d771a9d573ec5bb2969ed01f42343156237cae90"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
