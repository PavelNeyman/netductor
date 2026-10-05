class NetductorOp < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.250"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.250/netductor-op-darwin-arm64"
      sha256 "2de18a679295518743b9793292eef8e6cc05e190329ffbb4d22f62e0728b6087"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.250/netductor-op-darwin-amd64"
      sha256 "23a55faa61dbdf876e17857ed42cffe9b2cb277e601819e753b17dd2ff5754b3"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.250/netductor-op-linux-amd64"
      sha256 "583ebba7d3ff8693e0992640af0cfe7eed5e4be7c99e40f795c8be787f661650"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
