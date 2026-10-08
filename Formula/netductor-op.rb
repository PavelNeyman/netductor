class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.298"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.298/netductor-op-darwin-arm64"
      sha256 "5b7f60038feadb6a29c3f2c1cb346d74a3bdef4a35870d1fd4711491c51b4f2e"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.298/netductor-op-darwin-amd64"
      sha256 "10da2ce89abfe11e543a7188a9c25c68369068b3e00d7fa7ebefed2e0351e18f"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.298/netductor-op-linux-amd64"
      sha256 "4af90069e5e17a69fd39a6009aa29e84da6af635ac4005c2b111dfb44fefe104"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
