class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.378"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.378/netductor-op-darwin-arm64"
      sha256 "2961fb2d90f60254e09c351d55f75dd1426e3dc859609d059aac6fd9d2643315"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.378/netductor-op-darwin-amd64"
      sha256 "81b0f7de852fb32c8cf6f3aa9e297b9f9a0fde3bf267fdc58a1d0e5fe7308083"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.378/netductor-op-linux-amd64"
      sha256 "cd426721e837f83df8d4cbf555f7eb3394816287ea41820eb9ce90abf17e503d"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
