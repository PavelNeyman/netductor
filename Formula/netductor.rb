class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.119"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.119/netductor-op-darwin-arm64"
      sha256 "b79bf1c6f99f258e838c8bfd776f1b0d5c8fdcc1792af014b9797ec6dfcbead4"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.119/netductor-op-darwin-amd64"
      sha256 "1da3d350962b9c0ac78016c9b0a5379f8ca33b17d4236773ac714f6a20964b41"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.119/netductor-op-linux-amd64"
      sha256 "56eef0fb655e76136ce053f13fa5f82cdd0cf44125c225442f9a9246bdc77839"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
