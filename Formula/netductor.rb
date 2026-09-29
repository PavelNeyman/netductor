class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.104"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.104/netductor-op-darwin-arm64"
      sha256 "00683856f3bd6a149be7a21a9965418cc323a66f83d43d6990cf680b888b9867"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.104/netductor-op-linux-amd64"
      sha256 "5aac7c1efb87774c831e0544fe2f95e84ec5c41568f924d07742481bcb1966b8"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
