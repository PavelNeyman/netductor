class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.767"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.767/netductor-op-darwin-arm64"
      sha256 "50cb289d33a8bd8e0c4ad6329caa76d6181f2a432b373086c413017ca08f2919"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.767/netductor-op-darwin-amd64"
      sha256 "3649543fe8fe77eb2e3ec63d2261756c7d6d7c3d8cbab090fb93ec89964cef4d"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
