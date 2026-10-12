class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.340"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.340/netductor-op-darwin-arm64"
      sha256 "3b0e7c40065fb6d5187d1c940f5038422e78c2e08c8164394d8cf92aac2736b2"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.340/netductor-op-darwin-amd64"
      sha256 "c120d86d78b96007d688df6ff3c53de8751160b238a2facdc50859ae86bf1339"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.340/netductor-op-linux-amd64"
      sha256 "6e01bc871c07378b302f6c31f5998d86fe4f18947d6e69197961cc2e6d472b23"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
