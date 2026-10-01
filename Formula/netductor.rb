class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.159"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.159/netductor-op-darwin-arm64"
      sha256 "91c94c1e9a4048a14af9e0c9bf51501ea4332c97dbd11ad565da29db60d532a0"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.159/netductor-op-darwin-amd64"
      sha256 "28f48a7b5606228732f46ae084d47cb3e8a7bf5ae25c1c53ade2938bfbd36539"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
