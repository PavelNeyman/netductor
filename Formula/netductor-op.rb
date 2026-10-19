class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.423"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.423/netductor-op-darwin-arm64"
      sha256 "4c0ffff327fe3e702b646d5f94207eb0455355e0a8fbf75db4236e43ff17d4b6"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.423/netductor-op-darwin-amd64"
      sha256 "383e5cd66b9df9e3a4c386ffb822cad44052742dc499187a7402ef11bbf7b648"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.423/netductor-op-linux-amd64"
      sha256 "815ed46f8fee7d42471f2282cf00613084e10f296e15d24d7aace86b92a4e643"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
