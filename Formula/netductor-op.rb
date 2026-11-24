class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.766"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.766/netductor-op-darwin-arm64"
      sha256 "e9011c31ababcc83c2e8b1975e01386c3c5771ccb148941a1558f62f29a72895"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.766/netductor-op-darwin-amd64"
      sha256 "225f02ae65fe8e57e7aa96b2298eb7b5708fb15d7f86c83e6d6e9ac5be988cf7"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
