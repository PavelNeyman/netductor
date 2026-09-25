class Netductor < Formula
  desc "Netductor node / CLI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.15"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.15/netductor-darwin-arm64"
      sha256 "b58fa44a047ad4cfcea7bb4d488a5507d60987c857aa3ab9e3dd68b2cdfed71d"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.15/netductor-darwin-amd64"
      sha256 "0dd6ee98494ad56f677e1744397f53ee0786926f6a6b53676356a282c0d3b4f0"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.15/netductor-linux-amd64"
      sha256 "b57c2ae42686955d9fd1d3d89f98cd046d5f520bdd5c8348cc3f3bdc8f733629"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.15/netductor-linux-arm64"
      sha256 "dc9949d353f32f296e4f191735b272871e08f95c6ed2e41e83982d59349c4a2a"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
