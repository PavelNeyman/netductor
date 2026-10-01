class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.146"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.146/netductor-op-darwin-arm64"
      sha256 "3c2b2cc3b716684789e0c5d40382046971e8c183f5dd483f733904c0dad38200"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.146/netductor-op-darwin-amd64"
      sha256 "fcc5bfffba83cab7917b7f3d10e9abedc707a33c37b8079e1e7595a65f88b150"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.146/netductor-op-linux-amd64"
      sha256 "e51953ad0e3e7f8a661af0d46d90cf7dfacf89af4f7d2cf4e388da34f38cbc78"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
