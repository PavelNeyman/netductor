class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.290"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.290/netductor-op-darwin-arm64"
      sha256 "25fecd4bbb6c028d228435321c21149ac8bb2cc110611c5f89bcae2ccec3e46e"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.290/netductor-op-darwin-amd64"
      sha256 "cbcc0723d4f24ed7f5615d14a487d11e1ae83d25c0e2a68a783de02270372d17"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.290/netductor-op-linux-amd64"
      sha256 "76c2fa31848f312f8e85c3368e7c981397f634f6c5392e672c4ba021fb124599"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
