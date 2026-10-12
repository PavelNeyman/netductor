class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.362"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.362/netductor-op-darwin-arm64"
      sha256 "df3855f5e00b79497bed326657d96c923f481f105a48853a9c43a338403dfe27"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.362/netductor-op-darwin-amd64"
      sha256 "62c01f0dfc99c61738cc55c5fd917801b6cf2db394f38b23417573485de230c8"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.362/netductor-op-linux-amd64"
      sha256 "9c9252b54c4b2ff220e20d7bc6dcd31767e9120fb33963316c77684c9e2237a5"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
