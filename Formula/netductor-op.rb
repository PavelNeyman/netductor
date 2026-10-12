class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.352"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.352/netductor-op-darwin-arm64"
      sha256 "a32b85bee2e9fae33b8a05ecb3c9f4b6289a24d24abb9290a534867119015ed2"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.352/netductor-op-darwin-amd64"
      sha256 "967bb107b8794ad5b5d22fa6f3f7791514a8c82a90fdb4fd4870db16a6199c0e"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.352/netductor-op-linux-amd64"
      sha256 "a32167e3db34b12637c73fe493985045066817c83443131f4af22b38239d3493"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
