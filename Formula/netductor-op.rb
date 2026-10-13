class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.373"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.373/netductor-op-darwin-arm64"
      sha256 "a43d7bd94e51abe331e18b6b23b989cbc4acfe5f115ab7364213730e3c189c3e"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.373/netductor-op-darwin-amd64"
      sha256 "c20430f13279d8930918d23d14f15ce1e7c756fbb2fa0935618bd55c8b89acc4"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.373/netductor-op-linux-amd64"
      sha256 "aae76e18455fca31988c8d93d0058c3b179eb319856e376ece51ecbd84698638"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
