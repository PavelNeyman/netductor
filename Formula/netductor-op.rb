class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.658"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.658/netductor-op-darwin-arm64"
      sha256 "13049c43ce96b476cc3437894b1805403ab1b613fc63075eed7cdae128360d13"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.658/netductor-op-darwin-amd64"
      sha256 "46ae81b87cca250f526ffcbe0e10ed18a46a63703c2f6aab12e1932951b24a7c"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
