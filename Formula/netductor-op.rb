class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.306"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.306/netductor-op-darwin-arm64"
      sha256 "5db8bfc004995b995d756dadac7ece958e7ea02ee7a61a5246da3f1395b6da14"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.306/netductor-op-darwin-amd64"
      sha256 "0b9fc47d8a7e6efbaf3ae35315f71d6e0cf6e2895c6da70b2f8ca19e5a733474"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.306/netductor-op-linux-amd64"
      sha256 "ad81bd818ce8b3611a479d4b7213c89451d79f2e50801bc9e963cb56c187d012"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
