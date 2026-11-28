class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.787"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.787/netductor-op-darwin-arm64"
      sha256 "19d9fbfbd231d2e498bcb11a84b6b322c7965626d480a5188a31febfb943eefb"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.787/netductor-op-darwin-amd64"
      sha256 "3b3e8dc36d0c6df8110d4a08e5aaa39cd814f0029e990709d4b4d2140287f308"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
