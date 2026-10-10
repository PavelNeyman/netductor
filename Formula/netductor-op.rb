class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.325"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.325/netductor-op-darwin-arm64"
      sha256 "837befe000fabafa9c004da1a2cf09fdb38e67bbae461e0cc54bbe65ac0570c5"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.325/netductor-op-darwin-amd64"
      sha256 "f885af73ba830fbe4ca07b6950baa1926202a2dcf4333e4079ddaed77cbe12eb"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.325/netductor-op-linux-amd64"
      sha256 "73338020ad222d1dce6829ac20169c16bde6fbfc025bfec19c7a7b25ea73308d"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
