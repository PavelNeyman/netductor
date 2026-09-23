class Netductor < Formula
  desc "Netductor control plane CLI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.65"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.65/netductor-darwin-arm64"
      sha256 "60c0c88633abbc55601279c371aac7aebc56012aebc34715b561e856b540fb6d"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.65/netductor-darwin-amd64"
      sha256 "dbe2fd87255a3de975b9d27cb69da0a588e203dac4ddb17973b62a772d69e6e5"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.65/netductor-linux-amd64"
      sha256 "9897b11bcb62b5d12c8679a2d1b8a5216c20b4e2d75c3bfb9570848f792518c2"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.65/netductor-linux-arm64"
      sha256 "30b3f13940e07883f623ba673c216892b70992666263684f5ded06434fdfd789"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
