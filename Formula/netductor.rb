class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.188"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.188/netductor-op-darwin-arm64"
      sha256 "1dd74b207f8890dbd56a2718aab10a8737f1ff7da231dc16c32f2b3209f7393a"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.188/netductor-op-darwin-amd64"
      sha256 "594e4cba24322e560543aa025ce11488b7142eb8941e097591162565f9a0035b"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.188/netductor-op-linux-amd64"
      sha256 "93bd6f4707d3182bb334edf867631fdbbd6bb45f12957247d276b4b20f0218fc"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
