class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.332"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.332/netductor-op-darwin-arm64"
      sha256 "e04ed4a9bf1922e284384cc87676fd401eed4c897f5596f478f5f3d7f3eef619"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.332/netductor-op-darwin-amd64"
      sha256 "06919564b1a38ec636992e0d42e241b9ba6b1b8f359a003bcb425ff11ea3c763"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.332/netductor-op-linux-amd64"
      sha256 "799f80670cdf5e71f7cd00439e96070bfc269851f306be4fe9e69c1c8f2ef2f8"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
