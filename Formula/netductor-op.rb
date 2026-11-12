class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.716"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.716/netductor-op-darwin-arm64"
      sha256 "c8110b106acac3a44f1572ee423cca9f8c282b210e3f51f16a8f71a3d96d8103"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.716/netductor-op-darwin-amd64"
      sha256 "4fb8ffaf44c522de3d7fff03bf234324acbd5bc6e1580882860a62676c304611"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
