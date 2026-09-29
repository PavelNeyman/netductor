class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.112"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.112/netductor-op-darwin-arm64"
      sha256 "4ad2a3f92d7626e25c0d3d6fbd1bda72205af7dbdcb52d2cc42835e30836e1c2"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.112/netductor-op-linux-amd64"
      sha256 "7d29150cedc3d5f578983bb16134924ac54551405fe0cfd7b6d3755e2a2aef30"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
