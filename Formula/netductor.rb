class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.113"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.113/netductor-op-darwin-arm64"
      sha256 "8f61064e1b72f06a19505a64cfc349f7d417087370141d0f0230d339ae45e136"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.113/netductor-op-linux-amd64"
      sha256 "2f759b66dd85db422082c80d6965774dd80e6389ada53638651aa0c6191d648f"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
