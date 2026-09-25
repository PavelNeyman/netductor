class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.5"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.5/netductor-op-darwin-arm64"
      sha256 "d44c625b2aa359209ac62d3320ffde32fbec3b5c8bc3f8eb126ab4ded13ca703"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.5/netductor-op-darwin-amd64"
      sha256 "db5a942b5ffc81f1601b99a485aff563298c150ebbb5cb6daac1ed26e75b9672"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.5/netductor-op-linux-amd64"
      sha256 "ea61d53245f3be561c12ef08925746fc09e2491435bdf16e37eb659489753543"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.5/netductor-op-linux-arm64"
      sha256 "014785248ee9facd5b6dc25773ac1e021d41addb58d7cdfec82c1f086b2ca227"
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
