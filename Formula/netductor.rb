class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.169"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.169/netductor-op-darwin-arm64"
      sha256 "be54459c3cceaacd88559dc3df2a514b0726b757883ba6244c7a64e73914b2e7"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.169/netductor-op-darwin-amd64"
      sha256 "c915b661433178fdfc8c92d5eb6a7c83c9b6c657b0d1c905c2b342847a721deb"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.169/netductor-op-linux-amd64"
      sha256 "6fe63e054658f28645e880c50e466cff5586a91acf0f8958f97e752a9cda36fa"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
