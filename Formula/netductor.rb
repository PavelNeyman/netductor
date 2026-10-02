class Netductor < Formula
  desc "Netductor operator (Mac client)"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.184"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.184/netductor-op-darwin-arm64"
      sha256 "093c0005f6dc2eff84b16d09cd77cbc3db739f4a3951333fc2217ec1ea26d4f2"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.184/netductor-op-darwin-amd64"
      sha256 "18f7f06773d03d388aa75b229f01137be73f11fec06760c3505b1f7d61be301f"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.184/netductor-op-linux-amd64"
      sha256 "df47c3210137d8840cba74ee623ab2bbba76fb73b35c30b6d414abcdf2137e33"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
