class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.348"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.348/netductor-op-darwin-arm64"
      sha256 "8844b69e72fe4031408d30a4241f60ef6204c27d3f169d4343c31f1bd6090d30"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.348/netductor-op-darwin-amd64"
      sha256 "48494a00b3c9f08c9bfdf301b75c21fe02f051802ad84ea8f55baac634e5d7b7"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.348/netductor-op-linux-amd64"
      sha256 "14502f56d80a8dafce93b7de1417e92def2385f7d08233a6e6d8d7411c0fd9a5"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
