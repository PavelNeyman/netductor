class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.711"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.711/netductor-op-darwin-arm64"
      sha256 "d0df558bd72f8faf9a7e5968f1c0e5a67c09cf7be23661ad3606842fcb147cc2"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.711/netductor-op-darwin-amd64"
      sha256 "fe90707f82f0bf6c409f46f1e2cbfe0758c583e9eac42b0bcb7bd81888b79517"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
