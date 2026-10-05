class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.249"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.249/netductor-op-darwin-arm64"
      sha256 "56add52a9ebb949d56ae7602fe27b0c2ee6e42258d8cadd029165a09667f9a59"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.249/netductor-op-darwin-amd64"
      sha256 "77ba10a500927d49eaa0f1cb8b094d08ff4b918086f40b3046ddc0216110c7c8"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.249/netductor-op-linux-amd64"
      sha256 "a1fe26e25d195b86b0c0bb51a59b269a06a3810d7cbaffc9da280efb2eb0a59d"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
