class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.304"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.304/netductor-op-darwin-arm64"
      sha256 "fcaf96679284dc477571d598710846ace114f782e9d11b8446ccb845e8a7a6bf"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.304/netductor-op-darwin-amd64"
      sha256 "1675718cd2fde47a8c5b9a376f60ecd8115d980a972b59e08a4786d1576e1734"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.304/netductor-op-linux-amd64"
      sha256 "aaa401705b8730598abeef47fdb35455c9f68e200bbd34b11c7c670b529ab822"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
