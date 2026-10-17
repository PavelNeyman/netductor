class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.405"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.405/netductor-op-darwin-arm64"
      sha256 "72c3c28b13ba7fff5e5ea51bb335b260ff82f4277415a1b7cb7c39c2ad4f82f4"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.405/netductor-op-darwin-amd64"
      sha256 "9a7ebec9fefd120dbd11a16166f21d2d036deb365b4bd3676d0940d587a9952a"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.405/netductor-op-linux-amd64"
      sha256 "85d41b1de7d547d1e03bd38891bcfec9307b49370f0bb74a1c75b35bd0bd5ba8"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
