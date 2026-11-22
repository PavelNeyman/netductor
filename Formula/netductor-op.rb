class NetductorOp < Formula
  desc "Netductor operator (Mac client) — netductor-op only"
  homepage "https://github.com/PavelNeyman/netductor"
  version "0.9.763"
  license "MIT"
  on_macos do
    on_arm do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.763/netductor-op-darwin-arm64"
      sha256 "4384fc2ee93d568f775209ce30841cbc0cc332464357be7448a8dbc163f785f4"
    end
    on_intel do
      url "https://github.com/PavelNeyman/netductor/releases/download/v0.9.763/netductor-op-darwin-amd64"
      sha256 "bb984342c0c72f364c6f9e7a8436a22e49fc21df2872c26df170aa386f918fe8"
    end
  end
  def install
    bin.install Dir["netductor-op-*"].first => "netductor-op"
  end
  test do
    assert_match "operator", shell_output("#{bin}/netductor-op version 2>&1")
  end
end
