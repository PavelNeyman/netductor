class NetductorOp < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.181"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.181/netductor-op-darwin-arm64"
      sha256 "96eb6dc662e61ac6c0b633698e4dee1655869c202ef145cc7eb0287d7063cdb8"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.181/netductor-op-darwin-amd64"
      sha256 "02a2c097666bee7d57d6d7b314416ee4d5bfa6b7ab7c9682864ebd136c903f25"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.181/netductor-op-linux-amd64"
      sha256 "3c3e97eae13f1a8998dc0b7850d271ea9bc2774a78fa5e1002e69d001f00a107"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
