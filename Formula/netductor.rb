class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.103"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.103/netductor-op-darwin-arm64"
      sha256 "d17aef2c21ff54c87b4918d638300eca1dbb4457a66e7d0c4c5a357c01f54395"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.103/netductor-op-linux-amd64"
      sha256 "24d02df56c8280ce2e549b81f9720c5bf61b4f35121e307d3f6f1d2e8c82ce5c"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
