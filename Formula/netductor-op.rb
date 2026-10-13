class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.377"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.377/netductor-op-darwin-arm64"
      sha256 "f6bd92d47c2fa98fc5cbf5b478eb79d28697b3f286bf10fd93de8ca75d346fa9"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.377/netductor-op-darwin-amd64"
      sha256 "4990b5c59c30d714e67079b979e6f9d2f9fbd1f0684cd72da0afc65f8a4b393e"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.377/netductor-op-linux-amd64"
      sha256 "d7e32a47c917e113a92f42775d866d55d1a474d2c55797b7d20c5aea5e95c626"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
