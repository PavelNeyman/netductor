class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.174"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.174/netductor-op-darwin-arm64"
      sha256 "4e1d6391bb9664934b22252e2c82c9befbdd19cb160495de357b831f2c988b04"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.174/netductor-op-darwin-amd64"
      sha256 "a883053eda24c2ac74afe9a8895be2a466c6c1ba32cc817b28132253eb1576cf"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.174/netductor-op-linux-amd64"
      sha256 "a5944d72dab60319493c2e983db6d5576443494cdb4656d30657267f587c633b"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
