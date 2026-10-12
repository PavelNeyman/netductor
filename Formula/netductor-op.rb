class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.344"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.344/netductor-op-darwin-arm64"
      sha256 "700bdafba4cf361ed886160bc72a5afb55b615c799a65c05b789b43e83c75171"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.344/netductor-op-darwin-amd64"
      sha256 "7de605877ed9d4e7eeb73a519879c8684ba4c454f35a22bcd51a6f38f35d9f86"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.344/netductor-op-linux-amd64"
      sha256 "f590d4c6f498ea33e1bb7073a87ec0c1428bec6c471a9bd412e3538b2e871179"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
