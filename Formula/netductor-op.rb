class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.786"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.786/netductor-op-darwin-arm64"
      sha256 "657babb51798ac990b3f8150ea692bf88e867d52f9248c7f782dd39605ec35f4"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.786/netductor-op-darwin-amd64"
      sha256 "844a08420a25987a610550880543f9521af1f0c98bcf3e27aa985e472be37517"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
