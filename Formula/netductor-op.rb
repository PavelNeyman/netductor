class NetductorOp < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.248"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.248/netductor-op-darwin-arm64"
      sha256 "3dcf556d7f86e3c02bad8b437f20529467b3b73de0961f97074ed15e4901bf34"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.248/netductor-op-darwin-amd64"
      sha256 "dc09b90fdd888b73b5e29854814cf0382f85badedec1693d2a2ac36e4702b7bc"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.248/netductor-op-linux-amd64"
      sha256 "5005d2056ffb3884d5c9f7c131abc8f1eac0fbbacc18e4f4f01055f5a487e597"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
