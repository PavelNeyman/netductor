class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.358"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.358/netductor-op-darwin-arm64"
      sha256 "170f770c7d375bde0c36a3cfe878bb5745b267f6aed9b64eadb60d4b4e3cfcc3"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.358/netductor-op-darwin-amd64"
      sha256 "659502f3fff6195ac1d6e8b514f0a0eeeb2b4a96df16f4775e358bc71dd6c913"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.358/netductor-op-linux-amd64"
      sha256 "4e04be584142ead1fb9ec1f1ae18bb73365319e011084b2a0746103dcbc08c95"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
