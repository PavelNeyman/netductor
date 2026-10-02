class NetductorOp < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.177"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.177/netductor-op-darwin-arm64"
      sha256 "117506f62fc1ba951c3ff8b31d957d709c20eee8600106ea365c8dbfbcd1175f"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.177/netductor-op-darwin-amd64"
      sha256 "c5386e9260f386552202a6b923a0d28dd141c843f43e5c59da8563491da3a3df"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.177/netductor-op-linux-amd64"
      sha256 "918f573900eb841f7fa5a694844344da92e902a72ea5bcdf0cac86cc2f2ed9b6"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
