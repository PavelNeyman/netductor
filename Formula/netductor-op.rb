class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.657"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.657/netductor-op-darwin-arm64"
      sha256 "4e3473ac41fcb8fce8ecc05f17b0a59ae9658c33517fef14427144ba86eb68b5"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.657/netductor-op-darwin-amd64"
      sha256 "e457575f250d334dc68b1c5935294842e01ac3fb77431cecc79a8bb358009d3c"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
