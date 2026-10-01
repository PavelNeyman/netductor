class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.156"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.156/netductor-op-darwin-arm64"
      sha256 "6af101124aeb5bb213fbd988075e88ea9e4142d5dac640e1f1bb0a740ac3d9ed"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.156/netductor-op-darwin-amd64"
      sha256 "789385df9a55e1600a377365755352023beb3ab0a7899fea13dafb3f4d391d8d"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.156/netductor-op-linux-amd64"
      sha256 "ec983449dcf62b5d0cdf055c38e7470c91639fa19858cb2c27a6dc01163561ac"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
