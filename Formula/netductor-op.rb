class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.374"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.374/netductor-op-darwin-arm64"
      sha256 "f05d283efae84fa19bd28f165d649d567b88612bd92caa2577aad6251e9a5cb4"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.374/netductor-op-darwin-amd64"
      sha256 "ddff24d003b1bd91ac2ad3bddb40cb2b675f9b161fadd3b5fd19fc9d7df80f51"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.374/netductor-op-linux-amd64"
      sha256 "7ddce9f08798b89db5e93db7acecb3172179a3b981914ec2f79dd74f26672b36"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
