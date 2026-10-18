class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.417"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.417/netductor-op-darwin-arm64"
      sha256 "113b6eb09e186337df02cf4d2b330f0e221c603dac4e37c5e25d353974147b27"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.417/netductor-op-darwin-amd64"
      sha256 "08e8acf8cd6ad7e19159aded21db24870c0111e06d14abbafda2507a189f1ce2"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.417/netductor-op-linux-amd64"
      sha256 "1814db30641d5c285a6185162b3cd7a562c2a47ce25320ab4dc1313a8fe4a370"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor-op version 2>&1")
  end
end
