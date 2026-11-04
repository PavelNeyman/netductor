class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.661"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.661/netductor-op-darwin-arm64"
      sha256 "9d10a48eedd088e75910790614b35656083186b3bfe5f008bfd847e24eb50943"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.661/netductor-op-darwin-amd64"
      sha256 "7d0e034b8a9a8ab7ade8c0d3cd486b28f08e76c5e176e42f555addf54eb95ba1"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
