class NetductorOp < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.208"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.208/netductor-op-darwin-arm64"
      sha256 "837e2acfa47a5347dd56522a561c80d582876174a68d367e6d8af0786902e06c"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.208/netductor-op-darwin-amd64"
      sha256 "ab3027981063159ac11ba52c15d03dc0bb8275a416ee91fa5c2a5c0b74937889"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.208/netductor-op-linux-amd64"
      sha256 "3275808e5da639c7d7132f610e12730460fc58d4a5499201c0d0309038431130"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
