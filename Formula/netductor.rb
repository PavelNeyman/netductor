class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.77"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.77/netductor-darwin-arm64"
      sha256 "8541c4e39eb418d3e54b908b43e2d8701553e43f37153e138c3d761beaae033d"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.77/netductor-darwin-amd64"
      sha256 "d29c4641688701cd07a311fa39d5e10878477726e0af412fecb046f4b78c46fb"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.77/netductor-linux-amd64"
      sha256 "aae6508181d166801fd55c17e5dc0d8ff22dce90a3cb03f0172bf5dea5168378"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.77/netductor-linux-arm64"
      sha256 "a68f10d96f6dc4b378e36d2f4dffd0d332591de5ca1519625feea37dc950345b"
    end
  end
  def install
    bin.install Dir["netductor-*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
