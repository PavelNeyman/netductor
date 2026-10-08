class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.296"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.296/netductor-op-darwin-arm64"
      sha256 "660f1be5b7bc368e0cdad8c32517fe372b480c70baded2a20ae090ab0674720a"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.296/netductor-op-darwin-amd64"
      sha256 "d04daa18544f5bdd35895d5f880c72c93c558f0da7d00fef5a67a5c54388702e"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.296/netductor-op-linux-amd64"
      sha256 "b1444a7cf5147a13e439ab8544dc04cbc7b63a08022448727fb6fad969819635"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
