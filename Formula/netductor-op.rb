class NetductorOp < Formula
  desc "netductor"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.19"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.19/netductor-op-darwin-arm64"
      sha256 "564fb49d87774a3582a6e4209261a3c7ec699e35fa57ea55a1c6541d034134bf"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.19/netductor-op-darwin-amd64"
      sha256 "294cb6e930c6680af35540af75bb538717eb96fdfba835680e751948225d1737"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.19/netductor-op-linux-amd64"
      sha256 "a08ca6ea72e21d36c6f0cb076ae51004da8c4caf2d7b3043f61e16288e9c9d25"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.19/netductor-op-linux-arm64"
      sha256 "d1ffb13390d0ad01fe11512d514547a402e891af96c020790eb784528924caa7"
    end
  end
  def install
    bin.install Dir["netductor-op*"].first => "netductor-op"
  end
end
