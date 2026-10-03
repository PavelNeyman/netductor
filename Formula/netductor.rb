class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.206"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.206/netductor-op-darwin-arm64"
      sha256 "5f05bf4e70bac35ef240d23ac9be06b72527ff803a9cbd639056827e1e945f13"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.206/netductor-op-darwin-amd64"
      sha256 "e60f573accde45e469698440751cbe9a0ec8639a21a2214ead5208620bd9668c"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.206/netductor-op-linux-amd64"
      sha256 "c8685b3acba36775da4b0e1f7deafeeaaaaeb584f51e892582807dc4641bf9a7"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
