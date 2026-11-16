class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.754"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.754/netductor-op-darwin-arm64"
      sha256 "d1cc4b972bf04e441147adb6b3cc4ea369daa2a91b1cb88c2dc128d3dce9b7a9"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.754/netductor-op-darwin-amd64"
      sha256 "2c3e134bc1235b860af2930c0bc0a8bafbb437d05ca1ce3bc8667c1412d94be1"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
