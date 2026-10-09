class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.309"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.309/netductor-op-darwin-arm64"
      sha256 "8b6887648d00d0a97501a1bf02910f3344dd40c3bd9b9985cee05a0b992f2f67"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.309/netductor-op-darwin-amd64"
      sha256 "94fac5983baccae460d887019aa9e173f7c0f8b243d864dc7d7ef5ef369bc71e"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.309/netductor-op-linux-amd64"
      sha256 "4ce7b5357449e1d1111cd6b78cfc8adf55a9e78e83fee8c3ec57977d97cbc534"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
