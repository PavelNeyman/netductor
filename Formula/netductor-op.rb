class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.402"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.402/netductor-op-darwin-arm64"
      sha256 "9bb8e1f7b04a148fed1e5b6e0d6723b8db6d944ff4a0f8b2b73df2cbecf3513b"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.402/netductor-op-darwin-amd64"
      sha256 "e60a0c06e01da3edbe3986f30dd8d3ac9535de2dc00412ffb59c5f8fe61d3b11"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.402/netductor-op-linux-amd64"
      sha256 "38db47d48893e41752a2c64033835ac4d45fcd1555d362cfecbe27799514b6e2"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
