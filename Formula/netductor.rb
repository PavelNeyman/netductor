class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.189"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.189/netductor-op-darwin-arm64"
      sha256 "196b30b203b190d4ae59b6b6feb0323093fc372917d04a30e38359833fc24107"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.189/netductor-op-darwin-amd64"
      sha256 "981e2865cc63ea5e7c0f19e02853aef1eaf995cda01663ce36d150af36143bbe"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.189/netductor-op-linux-amd64"
      sha256 "221383564dc2700b8808734b2aa19ef8b5468f0af12b32b129560d6a9e8f8135"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
