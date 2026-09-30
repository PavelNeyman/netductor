class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.142"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.142/netductor-op-darwin-arm64"
      sha256 "fd244707cfe3433c8c9e3ce6dd99584a86f2c5f83daf06cd8991b4406a30d50c"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.142/netductor-op-linux-amd64"
      sha256 "b29cb412fc22a4e554550739c570038f31b717477b38eff7dd424fa1dcf063ef"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
