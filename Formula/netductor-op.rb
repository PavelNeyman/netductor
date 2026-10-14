class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.380"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.380/netductor-op-darwin-arm64"
      sha256 "f767f337ecade55abfb65402403d881b61f517856fdd046ca0470b90c0f45f19"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.380/netductor-op-darwin-amd64"
      sha256 "620d935378f7f87f9d1f8f65ea4fea2335b186bec4e79c532de0294b1aba3f27"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.380/netductor-op-linux-amd64"
      sha256 "e72c4230ceea60aa55f6c8902dc4a32cd6e56009a350139126f507fa3c7f7fb6"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
