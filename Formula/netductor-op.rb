class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.422"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.422/netductor-op-darwin-arm64"
      sha256 "f419813938469772e9ee9ab838c35b795a97326fe9a14bee7cdf1ac48e531f6f"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.422/netductor-op-darwin-amd64"
      sha256 "36e50bbf5cebe51d1915ec1c13b237b79cdf3555265b5f58cc02dea04b6a3587"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.422/netductor-op-linux-amd64"
      sha256 "e0f4ec1b155104f27270ef482e4cf040c3e8a2e1c4b59e71ec157820ddc72f9b"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
