class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.350"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.350/netductor-op-darwin-arm64"
      sha256 "8a5979e8d220fb75d3b741c7d655d3448c77662c548baa0de84a8081382b4980"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.350/netductor-op-darwin-amd64"
      sha256 "a9acec6aacd287a40b3c2da6a29348f494018e9341c32c80757ed15255f12813"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.350/netductor-op-linux-amd64"
      sha256 "ed76f0f7d16f7a398e5d1f43a0ef3f59c359fb0bdc9e5a5b25b9c28e92a74745"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
