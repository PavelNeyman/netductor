class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.663"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.663/netductor-op-darwin-arm64"
      sha256 "5f6146e68525b33b59965740f14b88988382aa700ec1202098a6806d509cd019"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.663/netductor-op-darwin-amd64"
      sha256 "16f8b397c76d391311d172c5d19da0365db2b06fdff2a38b6f1253b65431ecce"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
