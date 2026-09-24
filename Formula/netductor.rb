class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.94"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.94/netductor-darwin-arm64"
      sha256 "73ab9d39701158550fa1100a56991c47e780d61a4e46bd56290fff80dac212ee"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.94/netductor-darwin-amd64"
      sha256 "46d526a0fc1dcfa65490b8294c51af3b737707e01734ea1ae78598a49c7ff54d"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.94/netductor-linux-amd64"
      sha256 "70e01301549fd3211f36da1bf89b030f86ca9da76e6511999d6762b83e799f23"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.94/netductor-linux-arm64"
      sha256 "a2735705a6d65b19416329a66ad75e4fbf7f413a18d22dbc02dd871a2f2623d1"
    end
  end
  def install
    bin.install Dir["netductor-*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
