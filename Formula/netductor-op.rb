class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.318"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.318/netductor-op-darwin-arm64"
      sha256 "2418ef1115aa8c1da8ae7992850f362e4aeea5e877315fd8688d31cfef229978"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.318/netductor-op-darwin-amd64"
      sha256 "1c99c3ce2a46662acdbb79ca132322abdf53c32adf12b6efda77dde53bf31351"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.318/netductor-op-linux-amd64"
      sha256 "2b6c73834b13673044cf93c73d4e856f8666fe95bc093e3a3c572bdc8b42d4d8"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
