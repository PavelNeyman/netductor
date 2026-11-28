class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.780"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.780/netductor-op-darwin-arm64"
      sha256 "d70c4cab71e0e5544ea0f8eaebc21882b32aefa8a9b174c51056b8d773776b1b"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.780/netductor-op-darwin-amd64"
      sha256 "dc3082f0e51d0c87d0206d99c964b58fbf1c2b5fdb86ae4c4936315495930bf3"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
