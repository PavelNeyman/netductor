class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.330"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.330/netductor-op-darwin-arm64"
      sha256 "948497f67380c587169fc0dc83e72a10c9d11248c636b1349036d834cfc76dad"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.330/netductor-op-darwin-amd64"
      sha256 "c88b01e95eb9bc428737ec8791f8f20284b376c600ce0ac07d55274be048740c"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.330/netductor-op-linux-amd64"
      sha256 "c4b6166937f4b81cf4adf21643c694969a49a6cece8f16950051f2ad9ac3b115"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
