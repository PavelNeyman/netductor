class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.391"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.391/netductor-op-darwin-arm64"
      sha256 "69d256b0aefbe43e5fe97f01c3a148d941aded5bf71a51167564fb7d31d4d37d"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.391/netductor-op-darwin-amd64"
      sha256 "953d98705e9d1f184bdd9fe40fb1b46b868b64a2abb8228858e0fb05e2989687"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.391/netductor-op-linux-amd64"
      sha256 "9ffc2a39727c3a07dced7e904b77e2aaa78c484a7694bba52fb02d65dd3a3e2a"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
