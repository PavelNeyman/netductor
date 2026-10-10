class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.317"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.317/netductor-op-darwin-arm64"
      sha256 "538dfb3b6b2b3b91b10a32f3bb87c094ef0e44cfcc7dad4e184f0c0098cec90e"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.317/netductor-op-darwin-amd64"
      sha256 "445d128c54d061565a1b99828fee09802bdb64be19c0d6d5b5f3d149a035b3fa"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.317/netductor-op-linux-amd64"
      sha256 "3895a5d7b9ba5ee3792b1408a45a1b4751890dc954e32b53b6eeb3178a9517e5"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
