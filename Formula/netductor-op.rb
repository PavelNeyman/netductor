class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.346"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.346/netductor-op-darwin-arm64"
      sha256 "283aa3787a3671a8feee64703426dc36703b93acc56c663da35d5a63d7315589"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.346/netductor-op-darwin-amd64"
      sha256 "f66a2ae35f87fa853cb679f33592ed967ebf54616c3e3ea7404b1626025195a6"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.346/netductor-op-linux-amd64"
      sha256 "e19b12e907abf5acabf19bcfb02e2b06e656f9fc96da5e5255d870a131993e6c"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
