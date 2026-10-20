class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.440"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.440/netductor-op-darwin-arm64"
      sha256 "fc3a45ca71ce506da0a0a8f001a43faad7a168841a78372043a3da57877fd996"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.440/netductor-op-darwin-amd64"
      sha256 "8c0b0f3d2a50edba0549ee958e3b0c8e55a91fcda728b1cbdeee326f93ccf073"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.440/netductor-op-linux-amd64"
      sha256 "1046c78716cccf96280537a6a797a0a7199fb82b4f7a747171d9766de56cfda7"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
