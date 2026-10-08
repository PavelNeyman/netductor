class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.288"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.288/netductor-op-darwin-arm64"
      sha256 "b214e86a179006e7c3a2387338481ec42dde18fab792c2f3b35f6bc09ebbd96b"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.288/netductor-op-darwin-amd64"
      sha256 "a4519bbb8f0a73c73ca871d02aa29ecdaa81a0c99ae857e5a6b66651bd738fb2"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.288/netductor-op-linux-amd64"
      sha256 "fc2438fe1ea868bae72fc6719fa1797e34aa0cdcbe0c2df242a292003131af18"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
