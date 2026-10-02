class NetductorOp < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.187"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.187/netductor-op-darwin-arm64"
      sha256 "37fe7854b6810e04d1c1526b31ea68acad9b3a3130cec2d672377dd14f970167"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.187/netductor-op-darwin-amd64"
      sha256 "4bbead2c217ce30a00a9a2cdf18c1efd4122c01ebaab9787bbe8825a4df638d7"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.187/netductor-op-linux-amd64"
      sha256 "90d9e6ba03c61cd22b6505667273a2faf0dfe3034d4e913ffbc39f17725de867"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
