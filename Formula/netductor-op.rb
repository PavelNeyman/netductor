class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.334"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.334/netductor-op-darwin-arm64"
      sha256 "296c44d1ff63af5bc48b7197b58afb5116e888670512c96d823b0e42aa4e0316"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.334/netductor-op-darwin-amd64"
      sha256 "de3b24c5fe7d42384bf4b4d1ab34fb65cbd812f28b6516f7c3fefa967cce0ddb"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.334/netductor-op-linux-amd64"
      sha256 "c5602ad151d9fe4539479f156849ae66e53af77d064cdb0b8487236986fb8df5"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
