class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.185"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.185/netductor-op-darwin-arm64"
      sha256 "81537b887f0dda7660e7b3d848eecaa1edf19d06dd87c211f28505a58622bb69"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.185/netductor-op-darwin-amd64"
      sha256 "10b49ca83b3f99915873401e6f657b4f65f922dd57f0c93a6c894a73256f3338"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.185/netductor-op-linux-amd64"
      sha256 "1f856c253dfafd96852816d1b0950d03fb767ee351e2089d2d30ccafd2651516"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
