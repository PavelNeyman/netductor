class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.708"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.708/netductor-op-darwin-arm64"
      sha256 "a64f606248348c5ad4c7e660321338a973e88b7cf28a3caab5597d3fee4585fd"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.708/netductor-op-darwin-amd64"
      sha256 "5ef749a00c02b17b87c7e3372b0793eb13ec9e60d651293c634fe888d871dd58"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
