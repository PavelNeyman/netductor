class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.388"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.388/netductor-op-darwin-arm64"
      sha256 "88502ecba2004eeede36bd91f041e07c2f9e829e496c591a42c5814a5988d2b5"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.388/netductor-op-darwin-amd64"
      sha256 "750f88d0a3912ed31f8268c4f796c443ea846693f53402aab5c70b0043533ceb"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.388/netductor-op-linux-amd64"
      sha256 "e7b8f49bffcdf614a146560598ce1b0bc4f120155f849d27b9b1e013d3cbf996"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
