class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.360"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.360/netductor-op-darwin-arm64"
      sha256 "f24ec694bfabb16f2f8d7b9ee2ad79d241343384158d119da6566f6248b3c17d"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.360/netductor-op-darwin-amd64"
      sha256 "c76ad861aa010af44b221cc80da4b529b1db5673220436ff4b6ab707efcdbcb0"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.360/netductor-op-linux-amd64"
      sha256 "fc8ea3a991e41fcab1648a301bacfdaf2604fbd3d123e37ac1105e43842ad59b"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
