class NetductorOp < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.203"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.203/netductor-op-darwin-arm64"
      sha256 "d1711e958150979c8f50fcfb8152c45bcd0f25b93aed0d558d2821ba969c9e07"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.203/netductor-op-darwin-amd64"
      sha256 "38918fb1a29b3a6fa70bc69dbd03723a11f36b4f051f61efed9074f3654b9ae7"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.203/netductor-op-linux-amd64"
      sha256 "720276bfbae6ae8960c319d6e7a1225f2bb79028dca518723b374b075c825343"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
