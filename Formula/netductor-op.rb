class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.385"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.385/netductor-op-darwin-arm64"
      sha256 "1ca9d133c70571332252d54f413a92ac001171a372a9366c9611ae5bc3652664"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.385/netductor-op-darwin-amd64"
      sha256 "a63205af8c2144b6c51f1886e25f1fb3307ed18e94063f3ddadc1bb9e7f9ec6d"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.385/netductor-op-linux-amd64"
      sha256 "57a35755c0bf4d170aa1d161719b06153f71bd2a1b18a8527431022444a11a9c"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
