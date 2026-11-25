class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.772"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.772/netductor-op-darwin-arm64"
      sha256 "4755b78f53bb4d9db3401d7dd17093c5f9e6267fabdbba7b0ad0ee26016ba4e4"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.772/netductor-op-darwin-amd64"
      sha256 "d884fb83de85d48d5585710b0653aa5c85eb2de95aaca5a8caa84db895f4d704"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
