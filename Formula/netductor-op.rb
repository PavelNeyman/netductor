class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.301"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.301/netductor-op-darwin-arm64"
      sha256 "523ee22be8f98d1e95ffb175d7eed945b46a96d7680a70938f3457c212353003"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.301/netductor-op-darwin-amd64"
      sha256 "31e64efa6f8b487250e3ed1e4f14e09b941c39b250aafd9c1b438df26b6d99da"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.301/netductor-op-linux-amd64"
      sha256 "d83759bf7e555d0edf259ff37ab1bbc5e374b32aebc426cd2466a09aab9d9a75"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
