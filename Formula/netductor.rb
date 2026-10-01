class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.167"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.167/netductor-op-darwin-arm64"
      sha256 "eeb5da9b6705805d11a7ca0d94f045910d782527df1737e9c9a612bc674372ef"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.167/netductor-op-darwin-amd64"
      sha256 "0f6a897ba736e23a6bff2d1df72c838c014607b83dc83da5b9139abc15aa7208"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.167/netductor-op-linux-amd64"
      sha256 "a1a6ca9cb8d552ea52fd56597086d41fd6b6326fe90abe5449d6161df185e2fc"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
