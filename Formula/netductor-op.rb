class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.404"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.404/netductor-op-darwin-arm64"
      sha256 "7d9e2b9333de632b3631bb0274f92b1d42905c946f7118e7b7354c7438635d14"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.404/netductor-op-darwin-amd64"
      sha256 "2288b55d11632516fd1c0a0a198682a8b702bdaf38d6d1969e9dc5239c9eca63"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.404/netductor-op-linux-amd64"
      sha256 "24cf035a160b9f53325318d28223997a117232492f0cbfed72a581d3ea2ced76"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
