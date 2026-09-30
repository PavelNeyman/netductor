class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.123"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.123/netductor-op-darwin-arm64"
      sha256 "f114c0b54018eb36d431975e9ee1e56ac5a99a8095bd68d40d44245a34405642"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.123/netductor-op-darwin-amd64"
      sha256 "fb8dc3caff7a9e77d3a826b725d55d64fc970364ad177312bc672daf74b7b512"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.123/netductor-op-linux-amd64"
      sha256 "98c0e80174af3aa8f6569699f58cdd02a437b7c755468bce0bacebc73c9a5d01"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
