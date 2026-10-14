class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.381"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.381/netductor-op-darwin-arm64"
      sha256 "e47e410f53cf67ebdcdfea4d3f05e497d3e2510d0601c3bb8ebd60b0e7e920a5"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.381/netductor-op-darwin-amd64"
      sha256 "a398b462574397a730440b2efb0efe18ac84df693b75a82c1d2e48012415e277"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.381/netductor-op-linux-amd64"
      sha256 "f6da1d19c06e343f99afc4fa32a4b36cde11b0ba44ff3a151f53c32cba067ff0"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
