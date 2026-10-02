class NetductorOp < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.176"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.176/netductor-op-darwin-arm64"
      sha256 "ee9f1f23b1c257e7dd474e7419cf9e842afe3bd3e09b8b5176e33feda3c32800"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.176/netductor-op-darwin-amd64"
      sha256 "4935d50ce68b502e04ff8ede66891f31a38e065768c238baa3e15bb6797f4261"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.176/netductor-op-linux-amd64"
      sha256 "3565e9408bec4f566a3d62d381c57153ac93d589d8912d492356cd4f0b5ac232"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
