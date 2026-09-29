class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.95"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.95/netductor-op-darwin-arm64"
      sha256 "10a1d5c3e1db46c399be3e86b61d782f4fa16a8549d25c3a6bb0e7d858fd1de2"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.95/netductor-op-darwin-amd64"
      sha256 "79efbb562d9ad030b5437924a6187ab13eede9ad78c88f4201cb3d45b3279ae5"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.95/netductor-op-linux-amd64"
      sha256 "59b451968313525b961a4e3712990a27ed7bb911c542248a666c667c6fd466db"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.95/netductor-op-linux-arm64"
      sha256 "070eaeab0dc56ddfb2b9eed49f784930c284cd9de577c67dce00e67cdfadb057"
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
