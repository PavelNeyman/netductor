class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.46"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.46/netductor-op-darwin-arm64"
      sha256 "7ed96293219e38975117019f44970b98c8b92ba497eb71e24b64624df46f8802"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.46/netductor-op-darwin-amd64"
      sha256 "7c8062dd969b8eda320b0fd333537cc2a6f8114cc4bac8fab9376af0ba03ddfa"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.46/netductor-op-linux-amd64"
      sha256 "30a195774cad88fb07e57a9a01c9c10872de0f299d910bc20f8adf4418c0c2b1"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.46/netductor-op-linux-arm64"
      sha256 "aa2a7cd32bce3112416fd3f6036d3c1ad5becfb5b97974417bd515ad2d6e5c32"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
    bin.install_symlink "netductor-op" => "netductor"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
