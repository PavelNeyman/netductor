class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.357"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.357/netductor-op-darwin-arm64"
      sha256 "660eb1dbd06d1153eeafdb8b2b0bb27febff47f2a6ad9b9e93c641ef3c8566d1"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.357/netductor-op-darwin-amd64"
      sha256 "cec8166947217fd37357ce6bffe4e812a38ff6d40343f9c358af93b2f0011d06"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.357/netductor-op-linux-amd64"
      sha256 "dcea47a33cabc77e2665dfcab130409a91f83ee5f26ee860b246c887bbb45806"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
