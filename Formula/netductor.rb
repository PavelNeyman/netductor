class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.158"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.158/netductor-op-darwin-arm64"
      sha256 "e81827ffb1c2cd03e69a5d8371d33c3664fe25282a18cad74191994b78f1e595"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.158/netductor-op-darwin-amd64"
      sha256 "e19951ae191633daf993890d3a737a7d4bc1a7f86d30c783b294097c8714b27c"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.158/netductor-op-linux-amd64"
      sha256 "d4da48ad70d220f8c785909194619a177abec2d34ce73cd0e1e0c72ce00f8579"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
