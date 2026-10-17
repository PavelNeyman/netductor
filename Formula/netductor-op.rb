class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.408"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.408/netductor-op-darwin-arm64"
      sha256 "2fb00a51c23f4a1ef331b6d91325f837df7119abdeac9f5a31755d9ad2b1cbdc"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.408/netductor-op-darwin-amd64"
      sha256 "e73f880583f168c153023603672d80ed878ff17e6568f434b6652cb76a077223"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.408/netductor-op-linux-amd64"
      sha256 "9cc7899f133958769cf96d61a049e0205023fdca8ce5127754acb4d2e297073c"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
