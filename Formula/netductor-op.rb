class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.384"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.384/netductor-op-darwin-arm64"
      sha256 "0f902fbeb3dcf742000500818b30a6f78a85d0cdd2b5ebaa1f13881d24b95a93"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.384/netductor-op-darwin-amd64"
      sha256 "dfcacaf08002309778bb3e59d79a916b932c11d092aba161e7224073f54938eb"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.384/netductor-op-linux-amd64"
      sha256 "e369f313f3b2b22e15502fa187cc68d62eeab54a159fab8c19a50eba2ca2afbf"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
