class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.441"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.441/netductor-op-darwin-arm64"
      sha256 "59450e7386737b0f0a3fa01896cba54828ab472ec90bb85dc24463b76bdcadd9"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.441/netductor-op-darwin-amd64"
      sha256 "dc1116283ccd17dc5b6ca2b6b47168186da704f856330f4c8b8f3d617879b741"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.441/netductor-op-linux-amd64"
      sha256 "efc80e032fb488be2ea50d2f385750dbb8c12a05bf490267aeab1c25cf50f643"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
