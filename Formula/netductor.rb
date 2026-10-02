class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.175"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.175/netductor-op-darwin-arm64"
      sha256 "88dd52d66127895231c3782c26beb0ffbadc0300ef1f6211bfeb6221cd0695cd"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.175/netductor-op-darwin-amd64"
      sha256 "15d263841cabbfe76a66c13705e20ca7ccb58d60d3483f18bc0318e3559fa61b"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.175/netductor-op-linux-amd64"
      sha256 "1600c17e93d1992865275dc156536c312d487822a96219545a31accd6796509a"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
