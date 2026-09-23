class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.77"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.77/netductor-darwin-arm64"
      sha256 ""
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.77/netductor-darwin-amd64"
      sha256 ""
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.77/netductor-linux-amd64"
      sha256 ""
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.77/netductor-linux-arm64"
      sha256 ""
    end
  end
  def install
    bin.install Dir["netductor-*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
