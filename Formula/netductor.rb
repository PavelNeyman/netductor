class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.143"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.143/netductor-op-darwin-arm64"
      sha256 "12e246e0026cbfb9cacb9bfb7f33fae12307cb4bca7b5ad943641c63eb21a210"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.143/netductor-op-linux-amd64"
      sha256 "ecb91cf99b8157a9aecc7fb1e65c1fe9ff94b3db5636f7cdfe1780fdb54c5ab6"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
