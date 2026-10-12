class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.354"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.354/netductor-op-darwin-arm64"
      sha256 "e78f4dbb183f55842dc42a38dd59c90826a4102234929c51a34a597f7efedb60"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.354/netductor-op-darwin-amd64"
      sha256 "8e6bbf11f77878b087fceaa0f001f6c244d23370d7cef36044c280c07a7f8123"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.354/netductor-op-linux-amd64"
      sha256 "e5560f897836700216d0e2de5296bd1e63b660ac9d914370aa729f41f32d44c2"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
