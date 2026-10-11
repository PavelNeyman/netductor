class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.329"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.329/netductor-op-darwin-arm64"
      sha256 "2eb549ef48fce904fc886f1e0b6cbce6fc3a441744cf47cb40f33242ebca2f79"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.329/netductor-op-darwin-amd64"
      sha256 "b99871925331f2860800193d1d1965e733c9b8115244bbf8444792cb8ef350cb"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.329/netductor-op-linux-amd64"
      sha256 "967d873a0417f9beef563439baed40c8c4dfb7d9c3b64ecc7a1d7435c541303e"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
