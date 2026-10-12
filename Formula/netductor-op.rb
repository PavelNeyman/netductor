class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.347"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.347/netductor-op-darwin-arm64"
      sha256 "9360b8638db3fd61a73855c9c9a9721c23dbc816339bcc09184d9dc6eae07201"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.347/netductor-op-darwin-amd64"
      sha256 "d6ca6875251bedccfccee2ada7eb1ad123e1ca01724032dcc37f9068d16fa987"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.347/netductor-op-linux-amd64"
      sha256 "154ae7c84009fc4db970febea77bcda87996086d6b20ba79e85432f9982b0467"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
