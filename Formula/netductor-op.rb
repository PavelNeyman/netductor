class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.297"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.297/netductor-op-darwin-arm64"
      sha256 "2e0c2a22c1b6df60d8ece354fb9806a9359906e28810702a7f5de3f25887cff9"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.297/netductor-op-darwin-amd64"
      sha256 "d618d10948b617255db027cdcf2ca0b663cb26cd1744a20c6ca039c350e81076"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.297/netductor-op-linux-amd64"
      sha256 "dd1c8cf53bfb3433bde3839ffd70b1238af7ec4a3baa5548c2567212e1683945"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
