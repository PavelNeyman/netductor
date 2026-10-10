class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.323"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.323/netductor-op-darwin-arm64"
      sha256 "ec8fe149809f8d54f593e434724e2881833e0756f2f97a5a540d4d962350a81d"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.323/netductor-op-darwin-amd64"
      sha256 "c6916e12bdc33cc3d41d93577b2f8cd1404ea0cc6c51e7901d6014b0c3e45c60"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.323/netductor-op-linux-amd64"
      sha256 "47d5667e6f59b0ddfdd689e9e4148c26b7d15bb22aeb591dec078b91c6ddc661"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
