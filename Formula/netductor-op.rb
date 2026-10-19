class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.426"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.426/netductor-op-darwin-arm64"
      sha256 "37e205e958e8c649798bbd54bc380d31b425b3cdfb6287926283ee5214148572"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.426/netductor-op-darwin-amd64"
      sha256 "be0245a08d29f5dd4fb5d07c5e904ffd242153988d4d1697b08773ca07772ebc"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.426/netductor-op-linux-amd64"
      sha256 "eac45f7aaf0dee7faf7d1e3910d75a5204c4fb58eb60442d38d0dc3babb2a274"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
