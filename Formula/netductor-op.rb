class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.316"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.316/netductor-op-darwin-arm64"
      sha256 "4d12aa7349b313625fce4a737de30c87c8cbb53831565e043059b2689f5b0554"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.316/netductor-op-darwin-amd64"
      sha256 "49349e43d7d204f455e077e151e14fa810535ab26a75589242f9416ba2bb19f7"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.316/netductor-op-linux-amd64"
      sha256 "2c5db35c99930fb17ddd4201493e71c89eaacafe0b6fece2fa94167418553414"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
