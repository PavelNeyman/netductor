class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.356"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.356/netductor-op-darwin-arm64"
      sha256 "29574ee5ab0392a7fb78244e8b3ecdee5aacba434167f8044c3b61900a1c7cfc"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.356/netductor-op-darwin-amd64"
      sha256 "fc848e23ccee0a1c12a25a0f51e44e534bfdca3ab90a31e30ee88e5b674980dc"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.356/netductor-op-linux-amd64"
      sha256 "94883840738c9aee3effe9ac1bb4e45c5bc7637f5f2f3ec359357c5bbddded9a"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
