class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.375"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.375/netductor-op-darwin-arm64"
      sha256 "f85bde77659617e35f3c03751ebb92cc4c34466a2ef9fba1b182730ff044d10f"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.375/netductor-op-darwin-amd64"
      sha256 "7bcfce8c7a6ab1cfd2eb5a63c5bb0777a711846dc0b1e7041b06c2c1ddc776d2"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.375/netductor-op-linux-amd64"
      sha256 "7cc02991bbecac13dc6b5c7a64c9f9a4844713b1c0c93d99bc7340c2b53b5acd"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
