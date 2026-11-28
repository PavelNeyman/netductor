class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.788"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.788/netductor-op-darwin-arm64"
      sha256 "b31d9522b074e83838b21b464bd4422eff82a7c0bac3bef1e0c529b5ed04a970"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.788/netductor-op-darwin-amd64"
      sha256 "30fd0eda8c64fb370c81e54388895826ac174be64c6e869fb5b8ad5af7e93e32"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
