class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.97"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.97/netductor-darwin-arm64"
      sha256 "b0f87a45a3085ff72dde69cb441ea5d960097c7edd46006ae9f901cc7619f24c"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.97/netductor-darwin-amd64"
      sha256 "8c347eaa6d8c1190dfa7c6541f86dff1bdf8973360d33d9e6d158f66c63a4917"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.97/netductor-linux-amd64"
      sha256 "f80a860cbaca1db2543dba5c1f0d44c0b7bfefc2f5ff91c89d9f6f852a0da687"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.97/netductor-linux-arm64"
      sha256 "26cd349f312bb86303a1f0fe9aa1123112612546748d67a0f540ac1d1b64fe41"
    end
  end
  def install
    bin.install Dir["netductor-*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
