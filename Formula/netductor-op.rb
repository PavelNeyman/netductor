class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.342"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.342/netductor-op-darwin-arm64"
      sha256 "2f59d62dc97cb0a123362e02154f88b7d26e00bd4a5860894fe60197d3ef7b10"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.342/netductor-op-darwin-amd64"
      sha256 "294d07b89100064acb01dedf830bf462bb419c3dd08990c35692865566b23fa3"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.342/netductor-op-linux-amd64"
      sha256 "a3374dcc693789aec025ed04d276db2b988fc2545513f571da33bdb4bc92e327"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
