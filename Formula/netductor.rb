class Netductor < Formula
  desc "Netductor control plane CLI / TUI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.96"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.96/netductor-darwin-arm64"
      sha256 "d6df7f3091a2fa087e506882bf778b36e65efa57d9f4daf4c5f1d1152ad46c52"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.96/netductor-darwin-amd64"
      sha256 "17383ff516410c72d7ad05aa844b1d2afb1ef9d93f2ce84d31338eecaaaedd2d"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.96/netductor-linux-amd64"
      sha256 "a70a81f51fc38da3d5be524bba7617c2faa6c1c101b419ba93e553f67e43f0cd"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.96/netductor-linux-arm64"
      sha256 "7e4fec615bda10b8f50bff8f8150cba5180f61680c4402d59f0b7e9c3585fd17"
    end
  end
  def install
    bin.install Dir["netductor-*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
