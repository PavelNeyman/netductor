class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.345"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.345/netductor-op-darwin-arm64"
      sha256 "674d8efcd2edc995bb17b75caa98f0c84f4f4111f11df34a3be35567881367ed"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.345/netductor-op-darwin-amd64"
      sha256 "93e2a8565208e68d665cc0d05020d387d065c88ded1ca5ca4daf956897616f51"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.345/netductor-op-linux-amd64"
      sha256 "e5d3fb13e4087e2353d59927e3dc553aac10eb47a042cf33c49fd3b24707be6d"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
