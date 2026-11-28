class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.783"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.783/netductor-op-darwin-arm64"
      sha256 "f6603b4455509a4043cf5cd78aa1587c381a3cd1f04e7ba2bdffaefc0dd946e1"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.783/netductor-op-darwin-amd64"
      sha256 "de20e37960e94274ae78839fd0cd05320959973dd542476e216a2ef9b839da0e"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
