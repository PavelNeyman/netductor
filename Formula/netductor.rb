class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.170"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.170/netductor-op-darwin-arm64"
      sha256 "b58cb3b480f8c63ec36fcd6afbd18a1ca0af44e2448b3f0e3dd7426c1a344cc1"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.170/netductor-op-darwin-amd64"
      sha256 "27bd085109db6e8c948b4505720c6b4021aa28a6ea6962a6218ff468d677e069"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.170/netductor-op-linux-amd64"
      sha256 "ac9d21f9e5ec28b3154f73cc7a845512eefab5206f4b9d363e3c30d02cec1164"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
