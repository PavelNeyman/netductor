class NetductorOp < Formula
  desc "netductor"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.20"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.20/netductor-op-darwin-arm64"
      sha256 "ee9fda5c3aabd9d4d5b4c9c93fc79ab26da8e63db6875d28fc03a4f3f7b21c88"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.20/netductor-op-darwin-amd64"
      sha256 "dde85ec99dbf12a756af8b290f4ff61018fa9861944c399b923b672eee73cb57"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.20/netductor-op-linux-amd64"
      sha256 "9c475b70afc5b6fccadc9355cb577b504e8e5b91894741549eead34d0f17e2df"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.20/netductor-op-linux-arm64"
      sha256 "5213d06b08e905dc0f7b5c215f7f8ce1ae763fde486f71f858d89380983178c5"
    end
  end
  def install
    bin.install Dir["netductor-op*"].first => "netductor-op"
  end
end
