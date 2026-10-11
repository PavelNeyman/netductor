class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.331"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.331/netductor-op-darwin-arm64"
      sha256 "d2a5956b29b1587f0a8c32dfd6a2366c740e93a9db81037b716ae6e9ac4b9582"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.331/netductor-op-darwin-amd64"
      sha256 "6e24dff84cb735b445751ca1a374896af3d6385bb68e22f4192acc35f1761906"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.331/netductor-op-linux-amd64"
      sha256 "ca1b3b3e49ce78a2aa70918da6b2293c693e00c3db525d620fffec8fc99d882d"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
