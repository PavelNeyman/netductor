class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.310"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.310/netductor-op-darwin-arm64"
      sha256 "1d4a4d038ffdc21242c45350221e6227c07b7c1768eec2cdfc4113f770158e80"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.310/netductor-op-darwin-amd64"
      sha256 "435d6a728d03a21706cb047ebb4e3e56951200df55f3f5d84b9f2bc1c873eb9b"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.310/netductor-op-linux-amd64"
      sha256 "0390ac3348512dfa402cb009bfc957b96a8a79e824cabc279b71f3a1d8d8fea2"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
