class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.164"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.164/netductor-op-darwin-arm64"
      sha256 "b3845b1bbf2e73ed346e011f2870fd7a129b200d64166a4008f720ef567d7a30"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.164/netductor-op-darwin-amd64"
      sha256 "270bc3a9918f94394667b5709b0e0cd6c29201f79bdd03faa63cb15a7be5005e"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.164/netductor-op-linux-amd64"
      sha256 "f505193715b0b597d452f31efeb3c4036da385781b7e5632ffa5e0d20b5def6e"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
