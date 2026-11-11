class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.710"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.710/netductor-op-darwin-arm64"
      sha256 "f699b8db7316b96ea78a48fc5f6c56ab81194c3f0c966a96e2059abda0864446"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.710/netductor-op-darwin-amd64"
      sha256 "2bb363193448a2aa705d6390e283234fa34529f58717ee9d1b9a2cfe0a0faebd"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
