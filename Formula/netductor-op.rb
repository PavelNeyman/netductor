class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.421"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.421/netductor-op-darwin-arm64"
      sha256 "f1dc4fb786bd55dd08b2d87378951f5ea36f1948538e1121e46b97cebb79080c"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.421/netductor-op-darwin-amd64"
      sha256 "41228e3d9763a371723573f5a72391d9f6889196ed1636896ad0131ef33e7e43"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.421/netductor-op-linux-amd64"
      sha256 "d8650c05f2a8c3d8906e2231769ab7be02526574a3aa5ea65b818cbcbba1cafa"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor-op version 2>&1")
  end
end
