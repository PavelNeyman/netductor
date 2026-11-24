class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.771"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.771/netductor-op-darwin-arm64"
      sha256 "110d8a64c1949e843798b31d2934d3615b5e20b8d9a10e6d9963a993a8fcd479"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.771/netductor-op-darwin-amd64"
      sha256 "515e3cb84ce7bf5e12f360f39cb5ae776a3a212a149bd7e6fdba8ace9e447524"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
