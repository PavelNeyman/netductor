class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.359"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.359/netductor-op-darwin-arm64"
      sha256 "0364dfec220306120f24820bfe5e8bdd29f9c0a9e86cf9e70a75a319653f58c0"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.359/netductor-op-darwin-amd64"
      sha256 "cd3aec230fd3723e8b136096296e726f2ac327a5026a0175485e157f1aea77fb"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.359/netductor-op-linux-amd64"
      sha256 "cd76bafd273a7597466a6c853a966d86938b4bce40f9693ecced93671dc087bc"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
