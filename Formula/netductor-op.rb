class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.403"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.403/netductor-op-darwin-arm64"
      sha256 "8aad5326c6f7f36030d373c8bb92e654e23c1b9590ef3107dbee7b6902fee16a"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.403/netductor-op-darwin-amd64"
      sha256 "d0d3e36040229a9a3e2096ba62d0bd00ade820fa8308462aea5dac5e6c2f28af"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.403/netductor-op-linux-amd64"
      sha256 "9a634f3257f3390286196339dba5944b7c0ba38bf0fc127c64c4171a89fa5fd8"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
