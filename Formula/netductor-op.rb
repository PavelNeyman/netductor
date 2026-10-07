class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.278"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.278/netductor-op-darwin-arm64"
      sha256 "f1ba423919b7cacf20f6736c3fda10929bebc717695eaf8e2fd87f0f316dcb27"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.278/netductor-op-darwin-amd64"
      sha256 "5a09aeb0e5f014d3205f2db13bd53fcb5e7f67a634a11d4f440a6f7a2ccf6f95"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.278/netductor-op-linux-amd64"
      sha256 "488bb8a6b0e68e4e102ec2a847a44db435e140f76d8f05507fc034455e52a9c9"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
