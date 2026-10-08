class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.284"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.284/netductor-op-darwin-arm64"
      sha256 "931c9b5eb74156c33079472c0baded85eeb6208f0651643fa7acf40ddb7440d5"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.284/netductor-op-darwin-amd64"
      sha256 "4d7841b8c282b00e2f86494fcbbbf033ba9ec01cb8a4f370d8ec113ee8699162"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.284/netductor-op-linux-amd64"
      sha256 "650bbc7320acbffd404930a797c206a0313fec40ed8556ed11adfcc5a396af10"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
