class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.140"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.140/netductor-op-darwin-arm64"
      sha256 "6e8357f97e1edf39846443bfd6e2c8ae82fda7b9a7f1203e69651d0473b48057"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.140/netductor-op-linux-amd64"
      sha256 "dd1a1e40f3be071d50550dabecb16bd5d745cbc978305d2782198f83d0158416"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
