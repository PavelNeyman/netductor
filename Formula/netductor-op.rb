class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.424"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.424/netductor-op-darwin-arm64"
      sha256 "465e552ea7e8b0c7f4cad69f0647520e46a52f19b524e54fed646fd4e3e93ad6"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.424/netductor-op-darwin-amd64"
      sha256 "952a68b912aa755ec779750b8dfe7f847d7f83cdaccc629db6b85ab192289456"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.424/netductor-op-linux-amd64"
      sha256 "743f8cea7281b6e80cb50c43a34e497e609db73e1e981558c63af5bb60e2f5a0"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
