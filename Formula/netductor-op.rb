class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.289"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.289/netductor-op-darwin-arm64"
      sha256 "49dc0ff58db04352780feb5eb95fc2d5781dc845e81e06ecdcf1d983185d0442"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.289/netductor-op-darwin-amd64"
      sha256 "9ffc221f7ec855d5d5e4f7f785168c2038a127a49420051c75499676f5412d1a"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.289/netductor-op-linux-amd64"
      sha256 "75d85a27cb06c9570fd29ea6c7df01f8bb72aaf79241bb757329ed8b46d0e600"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
