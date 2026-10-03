class NetductorOp < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.202"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.202/netductor-op-darwin-arm64"
      sha256 "48f71489c8827dbca99253c95f9fcf0771acaaec8771971e644a4a3a28d334b1"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.202/netductor-op-darwin-amd64"
      sha256 "f36f495c3f09db63c2c6b666e4ccd6e7e179ad5c7c71d6ecac7234fabc83e97e"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.202/netductor-op-linux-amd64"
      sha256 "3e5f27ae155321c9de2bdac949174df0bf79bbac12138fcb620cd4fb402874d6"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
