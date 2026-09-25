class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.11"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.11/netductor-op-darwin-arm64"
      sha256 "4aa1addd7f8e97e2a97f4b772fc8b6c1d92c61dcb517ce042da42d7a69d9a161"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.11/netductor-op-darwin-amd64"
      sha256 "54a8a001cf5bf53f4553a16e98c05bd5400628a571c45a99d4b324d73abde2f8"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.11/netductor-op-linux-amd64"
      sha256 "c3d299d417565a1d56962eb8508f6d2ddfa9a232d028530bdefb0f6cc2a8f6d7"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.11/netductor-op-linux-arm64"
      sha256 "2050ec7b0450337db0bcb53fb937249257dd14854183d12527f60889ab7a6d36"
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
