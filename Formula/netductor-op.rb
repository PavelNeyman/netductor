class NetductorOp < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.180"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.180/netductor-op-darwin-arm64"
      sha256 "605bf99a3b73fbf6970b4aa5c9ccc443da05d788829fb89b23830af43a693e1e"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.180/netductor-op-darwin-amd64"
      sha256 "cff4b3672ab87e80f28d6a24ed7848f3a178d282efb548214285e9844803c7cc"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.180/netductor-op-linux-amd64"
      sha256 "ebfa04f42071fc7dee19ff721c03f3cd4b709f14a8bcc6ea5a1fbadb7e912b7a"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
