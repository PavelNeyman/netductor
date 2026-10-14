class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.383"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.383/netductor-op-darwin-arm64"
      sha256 "d919026644d6e7362d11cf3b9c6a2e8d6c6585eaee01ae0c53155e49bb9ce0fa"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.383/netductor-op-darwin-amd64"
      sha256 "0179f449e31aae2c3436cbd3d380c4c9436ef186da5d875e72b40b836224ee5c"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.383/netductor-op-linux-amd64"
      sha256 "a5730db7946802d2f4267c1f92d493f2b888664e9b10bdb2f1622b72e0de7c1a"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
