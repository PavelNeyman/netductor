class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.355"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.355/netductor-op-darwin-arm64"
      sha256 "a926341bcd8873e82fa88d2173a542a345b7994b75fb23ac805c325e25884a96"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.355/netductor-op-darwin-amd64"
      sha256 "5ffa1d65a5f0b0ad60449149f485c62155f7eed207c754377442bfca4e59034d"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.355/netductor-op-linux-amd64"
      sha256 "60163b92653f79159e689b878e6daee2bbc9769c2bef3c3ecf6399177a0cfa37"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
