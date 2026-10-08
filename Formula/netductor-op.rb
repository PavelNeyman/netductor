class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.281"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.281/netductor-op-darwin-arm64"
      sha256 "b8be02e786fb0653e3eb7ce698f1380ec84ac17106fc02c1a04cbd1d9c82e248"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.281/netductor-op-darwin-amd64"
      sha256 "77fab7f1d4596bf9fefad795b4c0ebf604377b54bd7ca91c0129f340a8bd79fa"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.281/netductor-op-linux-amd64"
      sha256 "a51a3f161c4253d1b1213dabd527ac9db3a14e7c816ebf4aadebbf99ba3c65cf"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
