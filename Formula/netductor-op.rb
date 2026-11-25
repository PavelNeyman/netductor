class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.773"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.773/netductor-op-darwin-arm64"
      sha256 "7a445c9ab3fd020e17ad519dbe9794e08c1297404759cf7fdd35f4218929e2cd"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.773/netductor-op-darwin-amd64"
      sha256 "c12b7a9b933f4dba008b997f99c96ed1f1bc60edc681286840ae2c30cb5a377a"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
