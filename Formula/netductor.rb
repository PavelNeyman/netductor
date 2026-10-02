class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.190"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.190/netductor-op-darwin-arm64"
      sha256 "96b8afc9721fdb0401ec9f645b3d679479c2dfddc863ec3999122f1bc5ee886a"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.190/netductor-op-darwin-amd64"
      sha256 "103a351866549304acb77aa442bf93f5ed90cef33effa6df5f94fd1008249619"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.190/netductor-op-linux-amd64"
      sha256 "6101827ee0e1e0c9512f3e27102d2c75291e2d0009f0f740352ff966db36fe54"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
