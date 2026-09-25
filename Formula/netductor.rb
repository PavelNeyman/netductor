class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.14"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.14/netductor-op-darwin-arm64"
      sha256 "fa217ee9f7f17147791e9c3724d7c03951524ec4cb9cb2f460f842771f6cf5c5"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.14/netductor-op-darwin-amd64"
      sha256 "d1370b3585999ff7d4b70e9da3565e61d3f6bd474a73b255cdb37d8aff4d6066"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.14/netductor-op-linux-amd64"
      sha256 "24aac7d88995f3e213f769b2c60a70f9858b0ed09dcf099698608fea190db48a"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.14/netductor-op-linux-arm64"
      sha256 "09ecd11143f4b44dfab9d15eba06e42e380c4b61c90474af66a020974715c2b9"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
    bin.install_symlink "netductor-op" => "netductor"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
