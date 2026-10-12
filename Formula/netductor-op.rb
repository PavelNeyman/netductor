class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.340"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.340/netductor-op-darwin-arm64"
      sha256 "941762774875fa01896eff62a015dc794a8a412737267c10da79512f95467215"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.340/netductor-op-darwin-amd64"
      sha256 "b21e73a8f7ecc899feafbeff6c9920e17c95ae288e310bb26457ac7ec924324d"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.340/netductor-op-linux-amd64"
      sha256 "1019c5aedaa4d62d52b651e5a14eaaa96fd7fedb318255f3877739700d359738"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
