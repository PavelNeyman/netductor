class NetductorOp < Formula
  desc "netductor"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.18"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.18/netductor-op-darwin-arm64"
      sha256 "48f2884ed63717196e10f945d2cb05ba62e803892a94ceb7a46bca0e2aeb56b6"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.18/netductor-op-darwin-amd64"
      sha256 "804238d2b3e5642286f5ffee80b3967da61dc2012e4af8bf19aea8f6eafd3cb9"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.18/netductor-op-linux-amd64"
      sha256 "2ff55dfbe52ec38ec6e739fe373e64f01aa19b7114b4bcf5d18abb8a182067ff"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.18/netductor-op-linux-arm64"
      sha256 "061c0706619486fc3b8e5e2643ae7caf83747415399987f8557eaef420245891"
    end
  end
  def install
    bin.install Dir["netductor-op*"].first => "netductor-op"
  end
end
