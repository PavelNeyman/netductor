class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.386"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.386/netductor-op-darwin-arm64"
      sha256 "b676f6f55a7dfed8750382af9e67907c0d8a3c74109ab2888d1a1de7abfadb7b"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.386/netductor-op-darwin-amd64"
      sha256 "4945f5dc7573fa21c9438cf4340866914fd11f5127ff5aab3da6f7e498de236c"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.386/netductor-op-linux-amd64"
      sha256 "329689705e20e274a8a785c57e4c80288ac320673ac664788e574d2d8c23be18"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
