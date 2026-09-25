class Netductor < Formula
  desc "netductor"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.19"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.19/netductor-darwin-arm64"
      sha256 "d9dbab8d34f78c763b8b004e02e2e52460305df59acf7d84acc0641e4650dc02"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.19/netductor-darwin-amd64"
      sha256 "dcb08a1791d6f87c5e151d205fae70d1bc9fb5a5b4a261de1407927bb9f36221"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.19/netductor-linux-amd64"
      sha256 "1e8d2920fd65170b6f6dc2df36ff8a7131f5aca4f064530a7aa691bfeb309164"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.19/netductor-linux-arm64"
      sha256 "e2eeeccb5e51c0f31afe045e6a1ec44e3f199982ebb44a55e8948fe7d9f13706"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
end
