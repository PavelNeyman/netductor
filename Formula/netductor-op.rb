class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.282"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.282/netductor-op-darwin-arm64"
      sha256 "2e4f22848e63aea077417fdacad6ccc0bb4a785ea358eba828f96f19b7284a13"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.282/netductor-op-darwin-amd64"
      sha256 "b0842bba75eff21cb0758a22685619020e4a55d264ed701db4c491d0384b4df2"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.282/netductor-op-linux-amd64"
      sha256 "1d74eec6c442f01329303cff8f63916afe193350e259c89f73e57c312b337640"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
