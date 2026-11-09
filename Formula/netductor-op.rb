class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.700"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.700/netductor-op-darwin-arm64"
      sha256 "a21af6afdbeb0a884a7f4da102c68e38873d6105d2b8d2ff834bad7ec82daea8"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.700/netductor-op-darwin-amd64"
      sha256 "259415616da09d74f748187ddce3c5e0e13647fb748b6caf4ce5dde144f8380d"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
