class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.401"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.401/netductor-op-darwin-arm64"
      sha256 "fc0190dabe21c35019a9a2d0b13ca23bad87b5cdac83032407ec45fc9cb637c6"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.401/netductor-op-darwin-amd64"
      sha256 "c0253ed97077f36423eaf344138c39ea0af4f02523da29eaef00adfd1ec1d515"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.401/netductor-op-linux-amd64"
      sha256 "923d36c0c1b47b5cab662f3952e7750624319810eebcc96420eb508b697af3de"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
