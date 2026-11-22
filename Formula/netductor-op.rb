class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.765"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.765/netductor-op-darwin-arm64"
      sha256 "f5e1112bb4384dfe4775b7c86cd72109bf69654965d1b8684d7e213263b09ab0"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.765/netductor-op-darwin-amd64"
      sha256 "3ccc7921ae224000d4b8f08b5a12cd89d35a7541300f9029a1aaa7b469d19848"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
