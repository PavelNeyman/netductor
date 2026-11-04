class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.660"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.660/netductor-op-darwin-arm64"
      sha256 "e0db7ca5f95a0863e597f9c3cace5e1c98855fd66943e30462cd75154a7f0ca6"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.660/netductor-op-darwin-amd64"
      sha256 "b0f4e35918a81c6788dee25dc3959e36250eb396baf56381432c524d297510e8"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
