class Netductor < Formula
  desc "Netductor operator"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.161"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.161/netductor-op-darwin-arm64"
      sha256 "fed7d82d6dc7d46ca53fb12f7161af7326fa9e00563d6248f680b68ca8d377b2"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
end
