class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.135"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.135/netductor-op-darwin-arm64"
      sha256 "dc3c9e2262497f4cf3d3c5145ab268fba7a4a8a98f6d04e1bf557303f29b69dc"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.135/netductor-op-linux-amd64"
      sha256 "1f231c16d4d0baac2e93b8d34f11c924866b1de017814c00938d40acfeceebb7"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
