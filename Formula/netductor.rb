class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.166"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.166/netductor-op-darwin-arm64"
      sha256 "d8b7051cc69a7e7d7e27bf2a1c8a1b8326b6a8a2a54f787e5fd7f4ea01a39c88"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.166/netductor-op-darwin-amd64"
      sha256 "b8223cdeba760675e8176fdfabcce0bf19a6a51fb77ce5cf8138925cda31cbf1"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.166/netductor-op-linux-amd64"
      sha256 "a5f51f722970b1e477da843a452a895708aa00758b7cdef353b64fb732ee09e7"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
