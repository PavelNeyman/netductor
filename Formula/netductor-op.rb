class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.774"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.774/netductor-op-darwin-arm64"
      sha256 "aca6f2c99477f2073e7382521334696a4aa7d4dd8181eebd7c0862781e86f7e3"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.774/netductor-op-darwin-amd64"
      sha256 "059d751bc0c95ccf1ed20bf4b68ce1d54d41907a7cf86f76b9749791d3c12e88"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
