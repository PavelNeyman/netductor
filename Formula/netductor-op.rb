class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.407"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.407/netductor-op-darwin-arm64"
      sha256 "569b6bf245f3c310253d4cf0fcc28eebed6434e28bc4faa332953cdd5fe0de85"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.407/netductor-op-darwin-amd64"
      sha256 "8cbb8b3a146e433c37b4f1f9c2303768c0fcc8f0ad6336774671536d6f751353"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.407/netductor-op-linux-amd64"
      sha256 "d4c6e91ee784cb4f2ced98e1e92c823247b64fb84df7d1fc505aa24e3113f7b5"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
