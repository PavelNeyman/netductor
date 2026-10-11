class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.337"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.337/netductor-op-darwin-arm64"
      sha256 "846ae7d743fa91d5a3688be8fba7b9c050e6ac776c7048277776a72e8eafbc0d"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.337/netductor-op-darwin-amd64"
      sha256 "a595295f46b36f3861d3283b01d45a89fae4db3adc8ae55714975737d6416341"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.337/netductor-op-linux-amd64"
      sha256 "0f529da4cd6a3bb2758c24050f46839e56526eaf67761df9d16f8de27da2f2a1"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
