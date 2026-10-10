class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.321"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.321/netductor-op-darwin-arm64"
      sha256 "74b03f0b222977b963c661ef87c6b9a767f1bd6323a6f099c5fbd6da22810417"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.321/netductor-op-darwin-amd64"
      sha256 "859df17984f6045655ff3afb100c201998b11fc6dc07be1e864da74efc14e5db"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.321/netductor-op-linux-amd64"
      sha256 "195ef412b1bb327589a62931d55cbe173d2845394124b8946a2d9b4acffae5d1"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
