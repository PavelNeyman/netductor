class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.420"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.420/netductor-op-darwin-arm64"
      sha256 "bb38c00bf5076d6f1d8ecd2645a3d964aaa2b58a47c68d3835ee32ddf4629a48"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.420/netductor-op-darwin-amd64"
      sha256 "4d8cb91445ba699e75fcf224ba249a1f471b5d4543d28e884c20c9b287322855"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.420/netductor-op-linux-amd64"
      sha256 "5973ec6466b154b0b10a6c44ff747918edaabc6afa35d9e08d40271200cc2ebb"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor-op version 2>&1")
  end
end
