class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.83"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.83/netductor-darwin-arm64"
      sha256 "0fe6991413f2a69d28ef70fcd714b77e92166cd8b49c35dcf3e6d3d2936cc4b6"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.83/netductor-darwin-amd64"
      sha256 "b15b2b99fa7f1de463dab40129a9702f5adb04a8e64fabf0119c217ceffc2832"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.83/netductor-linux-amd64"
      sha256 "dfa8ac689435e5a319a12b7e63a7d122680962665aa05e53b48c41eac9fc3277"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.83/netductor-linux-arm64"
      sha256 "a56dd0a2afdd17cb1f315448330db6679452f5ea00637b387d58e2d7088efca7"
    end
  end
  def install
    bin.install Dir["netductor-*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
