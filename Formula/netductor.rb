class Netductor < Formula
  desc "Netductor operator"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.127"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.127/netductor-op-darwin-arm64"
      sha256 "f9e240911162d2fdf1cebcd56fda736a21db76328248c5e35c013dc4ec6ac80a"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.127/netductor-op-darwin-amd64"
      sha256 "e646d1b1b821639cabedb043471588897b60ea28f743eed08d8ab535dbe73f8c"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.127/netductor-op-linux-amd64"
      sha256 "56a1afe16656faf4279b012850b3b57b992f77c59996901b868c4a42a589ce8f"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
