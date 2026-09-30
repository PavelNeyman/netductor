class Netductor < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.122"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.122/netductor-op-darwin-arm64"
      sha256 "d1c649e23d8c088023d096d00f1e0dcb75ecaf08765c22aab1bc090d8a62022a"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.122/netductor-op-darwin-amd64"
      sha256 "d3770f0f97088c5dd62e7eb18b2919bfd52fd7cf4397462f18d05531f9d9c787"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.122/netductor-op-linux-amd64"
      sha256 "ddc8221dfc2eb9b88b917feae0b00455e24ad584f4f68d70db410ce423db11a3"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
