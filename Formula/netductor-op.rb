class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.379"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.379/netductor-op-darwin-arm64"
      sha256 "e94c91dd2cd3e3d251ba1e3604237154925391d2b23ce651ee21ce93d705a35a"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.379/netductor-op-darwin-amd64"
      sha256 "5917555d16ced213795df57fa9b2d5aea6778a74b9c9e12bf697aca0c7d0e9a1"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.379/netductor-op-linux-amd64"
      sha256 "e0918016690e46413f1e46e1f7eb4ac9a586466bce5fe31d64359193389f61c4"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
