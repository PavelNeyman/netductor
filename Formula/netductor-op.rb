class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.302"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.302/netductor-op-darwin-arm64"
      sha256 "5431e3164cc4edb1b7638f78a098e166d9cd538f154e0318d8cc7a821dacff3a"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.302/netductor-op-darwin-amd64"
      sha256 "75027ed7b40205a5e57ad0964cd2d459abcb90feea6c6f81b2817f65a1ebd7f5"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.302/netductor-op-linux-amd64"
      sha256 "c8af0ac6eb89bf467fb4abc3dcc7ada6ef2bb248650e6e5f50e2af6c90794615"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
