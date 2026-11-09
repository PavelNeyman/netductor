class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.701"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.701/netductor-op-darwin-arm64"
      sha256 "37c8fb364fd73fa6332d5f96e79d6d1ddd06ae6e0a89f08427804a4d7c0cff3e"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.701/netductor-op-darwin-amd64"
      sha256 "602b0b1f35dc28c0200de63e3607ed9e99a5da7bd8b69019810fab20ab013be2"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
