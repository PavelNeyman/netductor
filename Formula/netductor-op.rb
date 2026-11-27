class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.775"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.775/netductor-op-darwin-arm64"
      sha256 "377ac7b7651b9ad0f6de467db423ca820e6ba3f05e04aae6733154ce40d01881"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.775/netductor-op-darwin-amd64"
      sha256 "36d05d202a7ccc7625864271909b1789189379781927d8076b3ac825e8e7caf6"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
