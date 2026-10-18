class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.414"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.414/netductor-op-darwin-arm64"
      sha256 "186946b02190252aeed4fd33f327d11427e9f3aeb9b1c47bccd417aa40f6fd7b"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.414/netductor-op-darwin-amd64"
      sha256 "d0ad41571e81189db4c4a6c063fbeded4889d79a2d8cd7d410fd49a8762325e4"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.414/netductor-op-linux-amd64"
      sha256 "18c42c2e063055492dcc566311b00fd484a45343f793d320f6e72ce8c61a9215"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
