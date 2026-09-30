class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.138"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.138/netductor-op-darwin-arm64"
      sha256 "48d1f85eee5aa1fa40da839fec659615059a4088eca1fe65940535fbbd10893d"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.138/netductor-op-linux-amd64"
      sha256 "46a317924fe9656f1fa32ad5a666ec992114e3c80bd26627c3257e8020327e45"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
