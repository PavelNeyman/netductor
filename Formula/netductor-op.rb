class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.322"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.322/netductor-op-darwin-arm64"
      sha256 "c34c1dd5cbdef9ac7eb8ce4536e609d1dba09c0a3da5e409b8757bbfef76ba96"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.322/netductor-op-darwin-amd64"
      sha256 "97ad99f425193e6430630430f2d7fa2273950251d7b5550f86d6d480cb70a0b8"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.322/netductor-op-linux-amd64"
      sha256 "ea353c57624fbdd35f2c6c929e9d0676ac5540e52b1ef26d85c6337e3917dc46"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
