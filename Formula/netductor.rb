class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.114"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.114/netductor-op-darwin-arm64"
      sha256 "2ab4f69516933a262bd4adb3445e0ad5872bc1583ca9a6eefda7f78a497cea69"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.114/netductor-op-linux-amd64"
      sha256 "055d3d42337e65ae44cfb3fd2b39165d49f980df3a71ae763edfcf1ecf1935ca"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
