class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.283"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.283/netductor-op-darwin-arm64"
      sha256 "b594b6d02083e0af3e52e4c989adcae3a9f3ecb3d7c9efb2dfaf627dab6c474c"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.283/netductor-op-darwin-amd64"
      sha256 "c1107e5213f502b1aaa296eab959bab0f840c1bb86ccdb854deb2f40f7724123"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.283/netductor-op-linux-amd64"
      sha256 "adb5329db03c0d681d82c2a25d0d1d2e8b1f48f4bada0342eed4de7eeca2be3d"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
