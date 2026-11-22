class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.761"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.761/netductor-op-darwin-arm64"
      sha256 "a141191af250b8f0d301be70a99266ac6f015e95992dffc8105a37e476b31b08"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.761/netductor-op-darwin-amd64"
      sha256 "433b6344469d2e1ecc9d9da92b6422b084fbe385a8ce0253604c946500e618d3"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
