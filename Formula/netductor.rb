class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.79"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.79/netductor-darwin-arm64"
      sha256 "e2547bb0b02d189f11d0f132f6cadc7774e1534285f1ef8ba92cee1c61ad42dc"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.79/netductor-darwin-amd64"
      sha256 "45440667a20d4238f3e3814a776139a476f6f7f693c1f9a6f5b32bc7b57e3d0b"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.79/netductor-linux-amd64"
      sha256 "9035b73e571362d2667bf1e777e28f3dbecdc38319a8ffb87953e5b04e122042"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.79/netductor-linux-arm64"
      sha256 "c05acf985e5ecb907f7df84c66380d83afb4ed03707a4622134793bb0e4f29d0"
    end
  end
  def install
    bin.install Dir["netductor-*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
