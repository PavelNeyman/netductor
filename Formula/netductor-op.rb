class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.319"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.319/netductor-op-darwin-arm64"
      sha256 "72a99d5f8b97744c9a14d2e5bc6f64941514aa28c4cae12042ec962385649de4"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.319/netductor-op-darwin-amd64"
      sha256 "ceccc3523a7713dbc563ed327d72ab76b93e44f4fdf0f193c70510ee669a4cc1"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.319/netductor-op-linux-amd64"
      sha256 "6a5f53cd5b0a52de19ccb299b27431dbedfc79dd943755e980b92eabec19872a"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
