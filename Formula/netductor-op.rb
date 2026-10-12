class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.353"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.353/netductor-op-darwin-arm64"
      sha256 "e2d847273f37cb72d8086d7b474c801263dc51875d2ff233ceccb97e669ae247"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.353/netductor-op-darwin-amd64"
      sha256 "00164f317dbb2808338bee0e59384a31b355846f2d1c572145fd977223d1a074"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.353/netductor-op-linux-amd64"
      sha256 "88bb3617269189ec1c585a77ee50d3d3572fb2283cb00605fd4c67fe8a227e45"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
