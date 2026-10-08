class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.291"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.291/netductor-op-darwin-arm64"
      sha256 "3b866bc5a9b2639d320eea4a41253551f8ab94b1258665ed84407ed7f10b9b3b"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.291/netductor-op-darwin-amd64"
      sha256 "1c92654bf3d5d7b9754ee5b7353c754247f66d9dc292b146aab631c9dd4a66c7"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.291/netductor-op-linux-amd64"
      sha256 "1998032128bbd538f415bac5f2d5e2936a85776787a7867fd9f1baf6856f9c7c"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
