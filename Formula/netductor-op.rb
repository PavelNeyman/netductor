class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.425"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.425/netductor-op-darwin-arm64"
      sha256 "24fa1b5ec5a3a65adffbd8c476cd194188b15376f6bbce573f9d34234697ec45"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.425/netductor-op-darwin-amd64"
      sha256 "053534f8119f6a857ee2bf9982b4082a060b196eb0a5b9a043702af7eb934baa"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.425/netductor-op-linux-amd64"
      sha256 "2ad77ed09c55cd3e6c84fda880de8424eaa7fb579e7553e8b112ead6a0ec2f00"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
