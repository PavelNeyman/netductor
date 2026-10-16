class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.400"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.400/netductor-op-darwin-arm64"
      sha256 "6f8acf6a01f25264f9a549cffff9ecb30545b0de49dbce5f8b37dee6585d2d05"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.400/netductor-op-darwin-amd64"
      sha256 "782b282894c5d130946af6aa2be3199d50b12e820ad9c5cac7962bfffa7ece7e"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.400/netductor-op-linux-amd64"
      sha256 "0c60ad94a522a99d228b5657cadd0f9bfb20f2810fb9cc0b132317c9e8e5c174"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
