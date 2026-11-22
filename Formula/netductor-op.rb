class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.764"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.764/netductor-op-darwin-arm64"
      sha256 "6a981ba7af776fc9879bd4528c57a4e81baad142f692b066c4f6fd0037279398"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.764/netductor-op-darwin-amd64"
      sha256 "5db068151d0d4d84fac6989a368ee8c65416ac457ae6705f45059e5cbf413b93"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
