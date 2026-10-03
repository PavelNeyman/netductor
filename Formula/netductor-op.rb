class NetductorOp < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.204"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.204/netductor-op-darwin-arm64"
      sha256 ""
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.204/netductor-op-darwin-amd64"
      sha256 ""
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.204/netductor-op-linux-amd64"
      sha256 ""
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
