class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.75"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.75/netductor-darwin-arm64"
      sha256 "bbb5c958c07c0927aaaf0fc3a3808bb5fe0420cb760c5cdf7e50df38f6fac7be"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.75/netductor-darwin-amd64"
      sha256 "9451bf54a66524cc4937bb639b5363eb5cc7bd3d4864778227b7ac0b24bb6e0b"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.75/netductor-linux-amd64"
      sha256 "715fce796ce5cf6b6f7b0a77b71d5d8af790fc9f68b0e75d848b712b138368d1"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.75/netductor-linux-arm64"
      sha256 "11a8d49248b0bd01fa224b8a4b3415c3c5eeb3a24dcd782563e2073bf1e23aea"
    end
  end

  def install
    bin.install Dir["netductor-*"].first => "netductor"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
