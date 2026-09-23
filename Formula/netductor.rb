class Netductor < Formula
  desc "Netductor control plane CLI"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.8.64"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.64/netductor-darwin-arm64"
      sha256 "e99c0106f1359dd0f0228707a423fe47c2926b32cd792b1b08803ab31e120ca2"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.64/netductor-darwin-amd64"
      sha256 "9715160be2471697bfc059ced9816f5871f614b3dd9446c99cb9f9cbaae9aa13"
    end
  end
  on_linux do
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.64/netductor-linux-amd64"
      sha256 "b9634e79d69a05baeed0f2356eb55c357a2cb4e531e256c21a44593d03d572de"
    end
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.8.64/netductor-linux-arm64"
      sha256 "28960b151efad422c91300e8421caed273aa8ed154bf6b100b51a692e1faf78a"
    end
  end
  def install
    bin.install Dir["netductor*"].first => "netductor"
  end
  test do
    assert_match version.to_s, shell_output("#{bin}/netductor version 2>&1")
  end
end
