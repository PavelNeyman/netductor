class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.781"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.781/netductor-op-darwin-arm64"
      sha256 "60a95f2bc93b89047e62dc09918796d87584396fb97808df4ecc6ad0cf906eca"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.781/netductor-op-darwin-amd64"
      sha256 "2e94d5d64a6a84d6ac83b03d4ac1fa687c9ae8c3993d8b7d9114525dd7b1e6c1"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
