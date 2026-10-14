class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.392"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.392/netductor-op-darwin-arm64"
      sha256 "4acfa6e68140bdc0298cf280a56f3949a7a48fcfdf73039e1b3f275933195f91"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.392/netductor-op-darwin-amd64"
      sha256 "14f62ac08c6b2b131e4287031eb05accebdf84e6af5ebbe04b1d6ada9f84450a"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.392/netductor-op-linux-amd64"
      sha256 "2e5301719e027b0acec5b84994fa9c09437760797d6ad8cba9a4d592e587994f"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
