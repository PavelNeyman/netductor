class NetductorOp < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.179"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.179/netductor-op-darwin-arm64"
      sha256 "1228ac4ed3b06eec1f8861f3b181489b403c287b1a1180e0c3d028643226f539"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.179/netductor-op-darwin-amd64"
      sha256 "04b383b01d1fe6a28d805a9feb0034e0b968b1bd7da936ab350f38c8bb47a4cc"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.179/netductor-op-linux-amd64"
      sha256 "5ba505667093134876de756ded3b99f8f5e4c7a6a0358282380123624b77257b"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
