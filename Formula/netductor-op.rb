class NetductorOp < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.183"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.183/netductor-op-darwin-arm64"
      sha256 "71f4b4193ae8f1eac3b7102007149ad3b5c3934e87c08a269269c0804b9e49f1"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.183/netductor-op-darwin-amd64"
      sha256 "0a638600f64e28a96b82dd0462f06de083555bda0584971cb3a0bcc84f63304d"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.183/netductor-op-linux-amd64"
      sha256 "eb6c10c6c485d4710c04a34ddb82a95d406f75df2ec696b208bbb5dc88f5dfce"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
