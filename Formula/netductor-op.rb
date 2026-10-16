class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.399"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.399/netductor-op-darwin-arm64"
      sha256 "a8eeb438eb507ada76f04b17cb009477bd112c61530db4ae3b2027890cd87640"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.399/netductor-op-darwin-amd64"
      sha256 "6dc18de9241a9c4def77d8dad48dd2ca714bfdab8c3e6aada59e32e7c118632c"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.399/netductor-op-linux-amd64"
      sha256 "cf6c16126d680c8271e9bfab3e6188be0de89f675b70ce957a0002f34dc9d400"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
