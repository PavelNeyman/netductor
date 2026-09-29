class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.96"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.96/netductor-op-darwin-arm64"
      sha256 "ae983440cb4051c8a3eb8ac74cffce2c8b9a5db27b72dafbc2738f7fcef2e6bc"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.96/netductor-op-darwin-amd64"
      sha256 "ece209a159a7d1595456a91e03c3406f7e2bdbdb1bc3f2622a06352ca0c09c18"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.96/netductor-op-linux-amd64"
      sha256 "51673614a4b61102a797760b2042beea486d1e6b7dcb0bb4db195c2292abe1e1"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.96/netductor-op-linux-arm64"
      sha256 "73d433e938cc9c6d13ccb7434d676f8308834c8b1c704ad150c417af0a5644b3"
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
